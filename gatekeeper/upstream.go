package gatekeeper

import "time"

type UpstreamID string

func (u UpstreamID) String() string {
	return string(u)
}

var NilUpstreamID UpstreamID = ""
var NilUpstream Upstream = Upstream{}

func NewUpstreamID() UpstreamID {
	return UpstreamID(GetUUID())
}

type UpstreamMatchType uint

const (
	NilUpstreamMatch UpstreamMatchType = iota + 1
	PrefixUpstreamMatch
	HostnameUpstreamMatch
	OtherUpstreamMatch
)

func (u UpstreamMatchType) String() string {
	switch u {
	case NilUpstreamMatch:
		return "no_match"
	case PrefixUpstreamMatch:
		return "prefix_match"
	case HostnameUpstreamMatch:
		return "hostname_match"
	case OtherUpstreamMatch:
		return "other_match"
	}

	return ""
}

type Upstream struct {
	ID        UpstreamID
	Name      string
	Protocols []Protocol
	Hostnames []string
	Prefixes  []string
	Timeout   time.Duration
	Extra     map[string]interface{}
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
