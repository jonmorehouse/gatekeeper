package shared

type ProtocolType uint

const (
	HTTPPublic ProtocolType = iota + 1
	HTTPPrivate
	TCPPublic
	TCPPrivate
)
