package router

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

	// this plugin accepts RPC calls to add and remove known upstreams for
	// routing purposes. This enables us to route a request to the upstream
	// of our choice.
	AddUpstream(*gatekeeper.Upstream) error
	RemoveUpstream(gatekeeper.UpstreamID) error

	// RouteRequest accepts a request and is expected to resolve an
	// Upstream to route this request towards. Specifically, this doesn't
	// make any opinions on which backend that is desired. Users whom would
	// like to route to a specific backend would most likely need to create
	// their own loadbalancer plugin in unison as well.
	RouteRequest(*gatekeeper.Request) (*gatekeeper.Upstream, *gatekeeper.Request, error)
}

// PluginClient in this case is the gatekeeper/core application. PluginClient
// is the interface that the user of this plugin sees and is simply a wrapper
// around *RPCClient. This is merely a wrapper which returns a clean interface
// with error interfaces instead of *gatekeeper.Error types
type PluginClient interface {
	internal.BasePluginClient

	AddUpstream(*gatekeeper.Upstream) error
	RemoveUpstream(gatekeeper.UpstreamID) error
	RouteRequest(*gatekeeper.Request) (*gatekeeper.Upstream, *gatekeeper.Request, error)
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

func (p *pluginClient) AddUpstream(upstream *gatekeeper.Upstream) error {
	if err := p.pluginRPC.AddUpstream(upstream); err != nil {
		return err
	}
	return nil
}

func (p *pluginClient) RemoveUpstream(upstreamID gatekeeper.UpstreamID) error {
	if err := p.pluginRPC.RemoveUpstream(upstreamID); err != nil {
		return err
	}
	return nil
}

func (p *pluginClient) RouteRequest(req *gatekeeper.Request) (*gatekeeper.Upstream, *gatekeeper.Request, error) {
	upstream, req, err := p.pluginRPC.RouteRequest(req)
	if err != nil {
		return upstream, req, err
	}
	return upstream, req, nil
}
