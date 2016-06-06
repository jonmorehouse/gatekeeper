package shared

type Protocol uint

const (
	HTTPPublic Protocol = iota + 1
	HTTPPrivate
	TCPPublic
	TCPPrivate
)
