package upstream

import (
	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	"github.com/jonmorehouse/gatekeeper/internal"
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
	UpstreamMetric(*gatekeeper.UpstreamMetric) error
}

// PluginClient in this case is the gatekeeper/core application. PluginClient
// is the interface that the user of this plugin sees and is simply a wrapper
// around *RPCClient. This is merely a wrapper which returns a clean interface
// with error interfaces instead of *gatekeeper.Error types
type PluginClient interface {
	internal.BasePluginClient

	SetManager(Manager) error
	WriteUpstreamMetrics([]*gatekeeper.UpstreamMetric) []error
}

func NewPluginClient(rpcClient *RPCClient, client *plugin.Client) PluginClient {
	return &pluginClient{
		rpcClient,
		internal.NewBasePluginClient(rpcClient, client),
	}
}

type pluginClient struct {
	pluginRPC *RPCClient
	internal.BasePluginClient
}

func (p *pluginClient) SetManager(manager Manager) error {
	if err := p.pluginRPC.SetManager(manager); err != nil {
		return err
	}
	return nil
}

func (p *pluginClient) WriteUpstreamMetrics(metrics []*gatekeeper.UpstreamMetric) []error {
	if errs := p.pluginRPC.UpstreamMetric(metrics); errs != nil {
		return gatekeeper.ErrorsToErrors(errs)
	}
	return nil
}
