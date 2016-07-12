package gatekeeper

import "time"

type UpstreamID string

func (u UpstreamID) String() string {
	return string(u)
}

var NilUpstreamID UpstreamID = ""
var NilUpstream Upstream = Upstream{}

func NewUpstreamID() UpstreamID {
	var uuid string
	RetryAndPanic(func() error {
		var err error
		uuid, err = NewUUID()
		return err
	}, 3)
	return UpstreamID(uuid)
}

type UpstreamMatchType uint

const (
	NilUpstreamMatch UpstreamMatchType = iota + 1
	PrefixUpstreamMatch
	HostnameUpstreamMatch
)

type Upstream struct {
	ID        UpstreamID
	Name      string
	Protocols []Protocol
	Hostnames []string
	Prefixes  []string
	Timeout   time.Duration
}

func (u Upstream) HasHostname(name string) bool {
	for _, hostname := range u.Hostnames {
		if name == hostname {
			return true
		}
	}
	return false
}

func (u Upstream) HasPrefix(name string) bool {
	for _, prefix := range u.Prefixes {
		if name == prefix {
			return true
		}
	}
	return false
}
