package ssdr

type RegistryMsgType int

const (
	MsgRegisterReq RegistryMsgType = iota
	MsgRegisterResp
	MsgServicePushReq
	MsgServicePushCancelReq
	MsgServicePushResp
	MsgCloseReq
)

type MsgRegistry struct {
	Type RegistryMsgType
	// MsgRegisterReq
	Service string
	Addr    string
	// MsgServiceListResp
	PushStart bool
	PushEnd   bool
	Services  ServiceListValue
	// resp
	Success bool
	Error   error
}

type RegistryService struct {
	Service string
	Id      string
	Addr    string
}

type RegistryServiceList map[string][]RegistryService
