package gatekeeper

type PluginOpts struct {
	Name string
	Cmd  string
	Opts map[string]interface{}
}

type Plugin interface {
	Start() error
	Stop() error
	Configure(map[string]interface{}) error
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
