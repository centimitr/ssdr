package ssdr

import "sync"

type Service []string

type ServiceListValue map[string]Service

func (svl ServiceListValue) Get(name string) Service {
	return svl[name]
}

type ServiceList struct {
	Services     ServiceListValue
	lock         sync.Mutex
	OnAddService Handler
}

func (sl *ServiceList) get(name string) Service {
	return sl.Services[name]
}

func (sl *ServiceList) set(name string, service Service) {
	if sl.Services == nil {
		sl.Services = make(map[string]Service)
	}
	sl.Services[name] = service
}

func (sl *ServiceList) Add(name string, addr string) {
	sl.lock.Lock()
	addrs := sl.get(name)
	exists := false
	for _, ad := range addrs {
		if ad == addr {
			exists = true
		}
	}
	if !exists {
		sl.set(name, append(addrs, addr))
		sl.OnAddService.Handler()
	}
	sl.lock.Unlock()
}

func (sl *ServiceList) AddByMsg(msg MsgRegistry) {
	sl.Add(msg.Service, msg.Addr)
}

func (sl *ServiceList) Remove(name string, addr string) {
	sl.lock.Lock()
	addrs := sl.get(name)
	j := 0
	for _, ad := range addrs {
		if ad != addr {
			addrs[j] = ad
			j++
		}
	}
	sl.set(name, addrs[:j])
	sl.lock.Unlock()
}

func (sl *ServiceList) RemoveByMsg(msg MsgRegistry) {
	sl.Remove(msg.Service, msg.Addr)
}
