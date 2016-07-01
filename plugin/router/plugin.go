package router

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

	// this plugin accepts RPC calls to add and remove known upstreams for
	// routing purposes. This enables us to route a request to the upstream
	// of our choice.
	AddUpstream(*shared.Upstream) error
	RemoveUpstream(shared.UpstreamID) error

	// RouteRequest accepts a request and is expected to resolve an
	// Upstream to route this request towards. Specifically, this doesn't
	// make any opinions on which backend that is desired. Users whom would
	// like to route to a specific backend would most likely need to create
	// their own loadbalancer plugin in unison as well.
	RouteRequest(*shared.Request) (*shared.Upstream, *shared.Request, error)
}

// PluginClient in this case is the gatekeeper/core application. PluginClient
// is the interface that the user of this plugin sees and is simply a wrapper
// around *RPCClient. This is merely a wrapper which returns a clean interface
// with error interfaces instead of *shared.Error types
type PluginClient interface {
	internal.BasePlugin

	AddUpstream(*shared.Upstream) error
	RemoveUpstream(shared.UpstreamID) error
	RouteRequest(*shared.Request) (*shared.Upstream, *shared.Request, error)
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

func (p *pluginClient) AddUpstream(upstream *shared.Upstream) error {
	return p.pluginRPC.AddUpstream(upstream)
}

func (p *pluginClient) RemoveUpstream(upstreamID shared.UpstreamID) error {
	return p.pluginRPC.RemoveUpstream(upstreamID)
}

func (p *pluginClient) RouteRequest(req *shared.Request) (*shared.Upstream, *shared.Request, error) {
	return p.pluginRPC.RouteRequest(req)
}
