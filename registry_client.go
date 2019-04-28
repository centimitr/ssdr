package ssdr

import (
	"errors"
	"github.com/gorilla/websocket"
	"sync"
)

type RegistryClient struct {
	RegistryAddr      string
	Conn              *websocket.Conn
	ServiceListValue  ServiceListValue
	recv              sync.Mutex
	ServiceListUpdate chan ServiceListValue
	Quit              chan struct{}
}

func NewRegistryClient(addr string) *RegistryClient {
	return &RegistryClient{
		RegistryAddr:      addr,
		ServiceListUpdate: make(chan ServiceListValue),
		Quit:              make(chan struct{}),
	}
}

func (ra *RegistryClient) Connect() (err error) {
	d := new(websocket.Dialer)
	ra.Conn, _, err = d.Dial(ra.RegistryAddr, nil)
	return
}

func (ra *RegistryClient) handleIncomingMsg(msg *MsgRegistry) error {
	if msg.Type != MsgRegisterResp {
		return errors.New("registry msg type not supported")
	}
	if !msg.Success {
		return msg.Error
	}
	return nil
}

func (ra *RegistryClient) call(msg *MsgRegistry, requireReply bool) (err error) {
	if requireReply {
		ra.recv.Lock()
	}
	err = ra.Conn.WriteJSON(*msg)
	if err != nil {
		return
	}
	if requireReply {
		var result MsgRegistry
		err = ra.Conn.ReadJSON(&result)
		if err != nil {
			return
		}
		ra.recv.Unlock()
		return ra.handleIncomingMsg(&result)
	}
	return
}

func (ra *RegistryClient) Disconnect() {
	if ra.Conn != nil {
		_ = ra.call(&MsgRegistry{Type: MsgCloseReq}, false)
		_ = ra.Conn.Close()
		ra.Conn = nil
	}
}

func (ra *RegistryClient) Register(serviceName string, id string, addr string) error {
	return ra.call(&MsgRegistry{Type: MsgRegisterReq, Service: serviceName, Id: id, Addr: addr}, true)
}

func (ra *RegistryClient) Subscribe() (err error) {
	err = ra.call(&MsgRegistry{Type: MsgServicePushReq}, false)
	if err != nil {
		return
	}
	go func() {
		var msg MsgRegistry
		for {
			err = ra.Conn.ReadJSON(&msg)
			if check(err) {
				return
			}
			if !msg.Success {
				check(msg.Error)
				break
			}
			if msg.PushStart {
				ra.recv.Lock()
			} else if msg.PushEnd {
				ra.recv.Unlock()
				break
			}
			ra.ServiceListValue = msg.Services
			go func() {
				ra.ServiceListUpdate <- msg.Services
			}()
		}
	}()
	return
}

func (ra *RegistryClient) Unsubscribe() error {
	return ra.call(&MsgRegistry{Type: MsgServicePushCancelReq}, false)
}

func (ra *RegistryClient) QuickSubscribe(service string, id string, addr string) (err error) {
	err = ra.Connect()
	if err != nil {
		return
	}
	err = ra.Register(service, id, addr)
	if err != nil {
		return
	}
	err = ra.Subscribe()
	return
}
