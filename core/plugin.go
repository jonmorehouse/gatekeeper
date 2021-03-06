package core

import "github.com/jonmorehouse/gatekeeper/gatekeeper"

type Plugin interface {
	Start() error
	Stop() error
	Configure(map[string]interface{}) error
	Heartbeat() error
	Kill()
}

type PluginType uint

const (
	LoadBalancerPlugin PluginType = iota + 1
	MetricPlugin
	ModifierPlugin
	RouterPlugin
	UpstreamPlugin
)

var pluginTypeMapping = map[PluginType]string{
	LoadBalancerPlugin: "loadbalancer-plugin",
	ModifierPlugin:     "modifier-plugin",
	MetricPlugin:       "metric-plugin",
	UpstreamPlugin:     "upstream-plugin",
	RouterPlugin:       "router-plugin",
}

func (p PluginType) String() string {
	desc, ok := pluginTypeMapping[p]
	if !ok {
		gatekeeper.ProgrammingError("PluginType string mapping not found")
	}
	return desc
}
