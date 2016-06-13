package gatekeeper

type Plugin interface {
	Start() error
	Stop() error
	Configure(map[string]interface{}) error
	Heartbeat() error
}

type PluginType uint

const (
	UpstreamPlugin PluginType = iota + 1
	LoadBalancerPlugin
	ModifierPlugin
	EventPlugin
)
