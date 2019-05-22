package ssdr

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net"
	"strconv"
	"strings"
	"sync"
)

const DefaultServicePort = 80

type Registry struct {
	Addr         string
	ServiceList  *ServiceList
	servicesLock sync.Mutex
	pushList     sync.Map
	//pushList     map[*websocket.Conn]int
}

func NewRegistry(addr string) *Registry {
	r := &Registry{
		Addr:        addr,
		ServiceList: new(ServiceList),
	}
	r.ServiceList.OnAddService = func() {
		r.pushAll()
	}
	return r
}

func (r *Registry) addPushList(conn *websocket.Conn) {
	//r.pushListLock.Lock()
	//if r.pushList == nil {
	//	r.pushList = make(map[*websocket.Conn]int)
	//}
	//r.pushList[conn] = 0
	r.pushList.Store(conn, 0)
	go r.push(conn, 0)
	//r.pushListLock.Unlock()
}

func (r *Registry) removePushList(conn *websocket.Conn) {
	//r.pushListLock.Lock()
	//if r.pushList != nil {
	//	r.pushList[conn] = -1
	//}
	r.pushList.Store(conn, -1)
	//r.pushListLock.Unlock()
}

func (r *Registry) push(conn *websocket.Conn, times int) {
	log("push:", conn.RemoteAddr())
	msg := &MsgRegistry{Type: MsgServicePushResp, Services: r.ServiceList.Services, Success: true}
	if times == -1 {
		msg.PushEnd = true
		r.pushList.Delete(conn)
		//delete(r.pushList, conn)
	} else {
		if times == 0 {
			msg.PushStart = true
		}
		r.pushList.Store(conn, times+1)
		//r.pushList[conn]++
	}
	check(conn.WriteJSON(msg), "push")
}

func (r *Registry) pushAll() {
	r.pushList.Range(func(k, v interface{}) bool {
		conn := k.(*websocket.Conn)
		times := v.(int)
		go r.push(conn, times)
		return true
	})
}

func (r *Registry) closeConn(conn *websocket.Conn, registerMsg MsgRegistry) {
	r.ServiceList.RemoveByMsg(registerMsg)
	r.removePushList(conn)
	_ = conn.Close()
}

func (r *Registry) handleIncomingConn(conn *websocket.Conn) {
	var msg MsgRegistry
	var err error
	// now limit a connection can only register one service at a time
	var registerMsg MsgRegistry
	for {
		err = conn.ReadJSON(&msg)
		if check(err) {
			r.closeConn(conn, registerMsg)
			return
		}
		switch msg.Type {
		case MsgCloseReq:
			_ = conn.Close()
		case MsgRegisterReq:
			var e error
			addr := conn.RemoteAddr().(*net.TCPAddr)
			if msg.Addr == "" {
				addr.Port = DefaultServicePort
				msg.Addr = addr.String()
			} else if strings.HasPrefix(msg.Addr, ":") {
				addr.Port, e = strconv.Atoi(msg.Addr[1:])
				msg.Addr = addr.String()
			}
			reply := &MsgRegistry{Type: MsgRegisterResp, Success: true, Error: e}
			if e == nil {
				registerMsg = msg
				r.ServiceList.RemoveByMsg(registerMsg)
				r.ServiceList.AddByMsg(registerMsg)
			}
			err = conn.WriteJSON(reply)
		case MsgServicePushReq:
			r.addPushList(conn)
		case MsgServicePushCancelReq:
			r.removePushList(conn)
		}
		if check(err) {
			r.closeConn(conn, registerMsg)
			return
		}
	}
}

func (r *Registry) Run() error {
	upgrader := websocket.Upgrader{}
	s := gin.New()
	s.NoRoute(func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if check(err, "upgrade") {
			return
		}
		log("connect:", conn.RemoteAddr())
		r.handleIncomingConn(conn)
	})
	log("listen:", r.Addr)
	return s.Run(r.Addr)
}
