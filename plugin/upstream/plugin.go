package upstream

import (
	"github.com/jonmorehouse/gatekeeper/internal"
	"github.com/jonmorehouse/gatekeeper/shared"
)

// Plugin is the interface which a plugin will implement and pass to `RunPlugin`
type Plugin interface {
	// internal.Plugin exposes the following methods, per:
	// https://github.com/jonmorehouse/gateekeeper/tree/master/internal/plugin.go
	//
	// Start() error
	// Stop() error
	// Heartbeat() error
	// Configure(map[string]interface{}) error
	//
	internal.BasePlugin

	SetManager(Manager) error
	UpstreamMetric(*shared.UpstreamMetric) error
}

// PluginClient in this case is the gatekeeper/core application. PluginClient
// is the interface that the user of this plugin sees and is simply a wrapper
// around *RPCClient. This is merely a wrapper which returns a clean interface
// with error interfaces instead of *shared.Error types
type PluginClient interface {
	internal.BasePlugin

	SetManager(Manager) error
	WriteUpstreamMetrics([]*shared.UpstreamMetric) []error
}

func NewPluginClient(rpcClient *RPCClient) PluginClient {
	return &pluginClient{
		rpcClient,
		internal.NewBasePluginClient(rpcClient),
	}
}

type pluginClient struct {
	pluginRPC *RPCClient
	*internal.BasePluginClient
}

func (p *pluginClient) SetManager(manager Manager) error {
	return p.pluginRPC.SetManager(manager)
}

func (p *pluginClient) WriteUpstreamMetrics(metrics []*shared.UpstreamMetric) []error {
	errs := p.pluginRPC.UpstreamMetric(metrics)
	return shared.ErrorsToErrors(errs)
}
