package ssdr

import "sync"

type ServiceNode struct {
	Id   string
	Addr string
}

type Service []ServiceNode

type ServiceListValue map[string]Service

func (svl ServiceListValue) Get(name string, excludeId string) Service {
	var nodes []ServiceNode
	for _, node := range svl[name] {
		if node.Id != excludeId {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (svl ServiceListValue) GetAddrs(name string, excludeId string) []string {
	nodes := svl.Get(name, excludeId)
	addrs := make([]string, len(nodes))
	for i, n := range nodes {
		addrs[i] = n.Addr
	}
	return addrs
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

func (sl *ServiceList) Add(name string, node ServiceNode) {
	sl.lock.Lock()
	nodes := sl.get(name)
	exists := false
	for _, n := range nodes {
		if n.Id == node.Id {
			exists = true
		}
	}
	if !exists {
		sl.set(name, append(nodes, node))
		sl.OnAddService.Handler()
	}
	sl.lock.Unlock()
}

func (sl *ServiceList) AddByMsg(msg MsgRegistry) {
	sl.Add(msg.Service, ServiceNode{msg.Id, msg.Addr})
}

func (sl *ServiceList) Remove(name string, node ServiceNode) {
	sl.lock.Lock()
	nodes := sl.get(name)
	j := 0
	for _, n := range nodes {
		if n.Id != node.Id {
			nodes[j] = n
			j++
		}
	}
	sl.set(name, nodes[:j])
	sl.lock.Unlock()
}

func (sl *ServiceList) RemoveByMsg(msg MsgRegistry) {
	sl.Remove(msg.Service, ServiceNode{msg.Id, msg.Addr})
}
