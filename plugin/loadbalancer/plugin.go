package loadbalancer

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

	AddBackend(gatekeeper.UpstreamID, *gatekeeper.Backend) error
	RemoveBackend(*gatekeeper.Backend) error
	UpstreamMetric(*gatekeeper.UpstreamMetric) error
	GetBackend(gatekeeper.UpstreamID) (*gatekeeper.Backend, error)
}

// PluginClient in this case is the gatekeeper/core application. PluginClient
// is the interface that the user of this plugin sees and is simply a wrapper
// around *RPCClient. This is merely a wrapper which returns a clean interface
// with error interfaces instead of *gatekeeper.Error types
type PluginClient interface {
	internal.BasePlugin

	AddBackend(gatekeeper.UpstreamID, *gatekeeper.Backend) error
	RemoveBackend(*gatekeeper.Backend) error
	GetBackend(gatekeeper.UpstreamID) (*gatekeeper.Backend, error)
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
	*internal.BasePluginClient
}

func (p *pluginClient) AddBackend(upstreamID gatekeeper.UpstreamID, backend *gatekeeper.Backend) error {
	return gatekeeper.ErrorToError(p.pluginRPC.AddBackend(upstreamID, backend))
}

func (p *pluginClient) RemoveBackend(backend *gatekeeper.Backend) error {
	return gatekeeper.ErrorToError(p.pluginRPC.RemoveBackend(backend))
}

func (p *pluginClient) WriteUpstreamMetrics(metrics []*gatekeeper.UpstreamMetric) []error {
	errs := p.pluginRPC.UpstreamMetric(metrics)
	return gatekeeper.ErrorsToErrors(errs)
}

func (p *pluginClient) GetBackend(upstreamID gatekeeper.UpstreamID) (*gatekeeper.Backend, error) {
	backend, err := p.pluginRPC.GetBackend(upstreamID)
	return backend, gatekeeper.ErrorToError(err)
}
