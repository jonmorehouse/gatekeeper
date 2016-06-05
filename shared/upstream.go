package shared

type UpstreamID string

var NilUpstreamID UpstreamID = ""
var NilUpstream Upstream = Upstream{}

type BackendID string

var NilBackendID BackendID = ""
var NilBackend Backend = Backend{}

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

type Backend struct {
	ID          BackendID
	Address     string
	HealthCheck string
}
