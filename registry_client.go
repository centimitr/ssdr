package ssdr

import (
	"errors"
	"github.com/gorilla/websocket"
	"sync"
)

type RegistryClient struct {
	RegistryAddr string
	Conn         *websocket.Conn
	Services     map[string][]string
	recv         sync.Mutex
}

func NewRegistryClient(addr string) *RegistryClient {
	return &RegistryClient{
		RegistryAddr: addr,
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

func (ra *RegistryClient) RegisterAddr(name string, addr string) error {
	return ra.call(&MsgRegistry{Type: MsgRegisterReq, Service: name, Addr: addr}, true)
}

func (ra *RegistryClient) Register(name string) error {
	return ra.RegisterAddr(name, "")
}

type SubscribeHandler func(services map[string][]string)

func (h *SubscribeHandler) Handle(services map[string][]string) {
	if *h != nil {
		(*h)(services)
	}
}

func (ra *RegistryClient) Subscribe(handler SubscribeHandler) (err error) {
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
			ra.Services = msg.Services
			handler.Handle(msg.Services)
		}
	}()
	return
}

func (ra *RegistryClient) Unsubscribe() error {
	return ra.call(&MsgRegistry{Type: MsgServicePushCancelReq}, false)
}

func (ra *RegistryClient) Query(name string) (addrs []string) {
	if ra.Services == nil {
		return
	}
	return ra.Services[name]
}

func (ra *RegistryClient) QuickSubscribeByAddr(service string, addr string, handler SubscribeHandler) (err error) {
	err = ra.Connect()
	if err != nil {
		return
	}
	err = ra.RegisterAddr(service, addr)
	if err != nil {
		return
	}
	err = ra.Subscribe(handler)
	return
}

func (ra *RegistryClient) QuickSubscribe(service string, handler SubscribeHandler) (err error) {
	return ra.QuickSubscribeByAddr(service, "", handler)
}
