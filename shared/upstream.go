package shared

import "net/url"

type UpstreamID string

var NilUpstreamID UpstreamID = ""

type BackendID string

var NilBackendID BackendID = ""

type Upstream struct {
	ID        UpstreamID
	Name      string
	Protocols []ProtocolType
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

// converts a plugin backend to a local backend object
type Backend struct {
	ID          BackendID
	Address     url.URL
	HealthCheck string
}
