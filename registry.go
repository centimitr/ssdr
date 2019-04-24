package ssdr

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"sync"
)

type Registry struct {
	Addr         string
	Services     map[string][]string
	servicesLock sync.Mutex
	pushList     map[*websocket.Conn]int
	pushListLock sync.Mutex
}

func NewRegistry(addr string) *Registry {
	return &Registry{
		Addr:     addr,
		Services: make(map[string][]string),
	}
}

func (r *Registry) addPushList(conn *websocket.Conn) {
	r.pushListLock.Lock()
	if r.pushList == nil {
		r.pushList = make(map[*websocket.Conn]int)
	}
	r.pushList[conn] = 0
	go r.push(conn, 0)
	r.pushListLock.Unlock()
}

func (r *Registry) removePushList(conn *websocket.Conn) {
	r.pushListLock.Lock()
	if r.pushList != nil {
		r.pushList[conn] = -1
	}
	r.pushListLock.Unlock()
}

func (r *Registry) push(conn *websocket.Conn, times int) {
	log("push:", conn.RemoteAddr())
	msg := &MsgRegistry{Type: MsgServicePushResp, Services: r.Services, Success: true}
	if times == -1 {
		msg.PushEnd = true
		delete(r.pushList, conn)
	} else {
		if times == 0 {
			msg.PushStart = true
		}
		r.pushList[conn]++
	}
	check(conn.WriteJSON(msg), "push")
}

func (r *Registry) pushAll() {
	for conn, times := range r.pushList {
		go r.push(conn, times)
	}
}

func (r *Registry) addService(registerMsg MsgRegistry) {
	r.servicesLock.Lock()
	addrs := r.Services[registerMsg.Service]
	exists := false
	for _, addr := range addrs {
		if addr == registerMsg.Addr {
			exists = true
		}
	}
	if !exists {
		r.Services[registerMsg.Service] = append(addrs, registerMsg.Addr)
		r.pushAll()
	}
	r.servicesLock.Unlock()
}

func (r *Registry) removeService(registerMsg MsgRegistry) {
	r.servicesLock.Lock()
	addrs := r.Services[registerMsg.Service]
	j := 0
	for _, addr := range addrs {
		if addr != registerMsg.Addr {
			addrs[j] = addr
			j++
		}
	}
	r.Services[registerMsg.Service] = addrs[:j]
	r.servicesLock.Unlock()
}

func (r *Registry) closeConn(conn *websocket.Conn, registerMsg MsgRegistry) {
	r.removeService(registerMsg)
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
			if msg.Addr == "" {
				msg.Addr = conn.RemoteAddr().String()
			}
			registerMsg = msg
			r.removeService(registerMsg)
			r.addService(registerMsg)
			err = conn.WriteJSON(&MsgRegistry{Type: MsgRegisterResp, Success: true})
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
