package gatekeeper

import (
	"github.com/jonmorehouse/gatekeeper/shared"
)

type Plugin interface {
	Start() *shared.Error
	Stop() *shared.Error
	Configure(map[string]interface{}) *shared.Error
}

type PluginType uint

const (
	UpstreamPlugin PluginType = iota + 1

	// NOTE none of these exist yet
	LoadBalancerPlugin
	RequestPlugin
	ResponsePlugin
	ProxyPlugin
)
