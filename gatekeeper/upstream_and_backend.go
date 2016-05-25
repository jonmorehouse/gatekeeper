package gatekeeper

import (
	"net/url"

	"github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

type ProtocolType uint

const (
	HTTPPublicProtocolType = iota + 1
	HTTPPrivateProtocolType
	TCPPublicProtocolType
	TCPPrivateProtocolType
)

// internal representation of an upstream.Upstream
type Upstream struct {
	ID        upstream.UpstreamID
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

// converts a plugin upstream to a local upstream
func PluginUpstreamToUpstream(u upstream.Upstream, uID upstream.UpstreamID) *Upstream {
	if uID == upstream.NilUpstreamID {
		RetryAndPanic(func() error {
			uuid, err := NewUUID()
			if err != nil {
				return err
			}
			uID = upstream.UpstreamID(uuid)
			return nil
		}, 3)
	}

	return &Upstream{
		ID:        uID,
		Name:      u.Name,
		Protocols: []ProtocolType{},
		Hostnames: []string{},
		Opts:      nil,
	}
}

// converts a plugin backend to a local backend object
type Backend struct {
	ID          upstream.BackendID
	Address     url.URL
	HealthCheck string
}

func PluginBackendToBackend(b upstream.Backend, bID upstream.BackendID) *Backend {
	if bID == upstream.NilBackendID {
		RetryAndPanic(func() error {
			uuid, err := NewUUID()
			if err != nil {
				return err
			}
			bID = upstream.BackendID(uuid)
			return nil
		}, 3)
	}

	return &Backend{
		ID:          bID,
		Address:     url.URL{},
		HealthCheck: "",
	}
}
