package server

type IpDiscovery interface {
	Join()

	Leave()

	Ping()

	GetAddresses() []string

	SetOnJoin(ls func(address string))

	SetOnLeave(ls func(address string))

	SetOnPing(ls func(address string))

	AutoLeave()
}

const IpPingMessageJoin int = 1
const IpPingMessageLeave int = 2

type IpPingMessage struct {
	Status         int    `json:"status"`
	MessageAddress string `json:"message_address"`
}
