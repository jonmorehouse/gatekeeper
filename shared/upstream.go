package shared

type UpstreamID string

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
	NilMatch UpstreamMatchType = iota + 1
	PrefixMatch
	HostnameMatch
)

type Upstream struct {
	ID        UpstreamID
	Name      string
	Protocols []Protocol
	Hostnames []string
	Prefixes  []string
	Opts      map[string]interface{}
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
