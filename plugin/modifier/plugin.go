package modifier

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

	// Modify a request, changing anything about the requests' nature.
	// Specifically this could mean, swapping out the backend, swapping out
	// the upstream, returning an error, adding a response or anything
	// else. Adding a Response to the request or adding / returning an
	// error will stop the request life cycle immediately and will return
	// immediately. If a response is added, that will be written back
	// directly where as an error will trigger the ErrorResponse method.
	// Returning an error from this method should only be done in
	// extenuating circumstances and will trigger an internal error
	ModifyRequest(*gatekeeper.Request) (*gatekeeper.Request, error)

	// Modify the response, changing any attributes, headers, the body,
	// that are desired before sending the response back to the client.
	// This method should only return an error in the case of an
	// extenuating circumstance and/or when the response body can be
	// dropped all together. Most likely speaking, that would only be in
	// the case of a fatal failure such as datastore being down etc.
	ModifyResponse(*gatekeeper.Request, *gatekeeper.Response) (*gatekeeper.Response, error)

	// Modify a response that was flagged as an error. This is similar to
	// the ModifyResponse method, again giving complete control over the
	// response that is written back to the client.
	ModifyErrorResponse(error, *gatekeeper.Request, *gatekeeper.Response) (*gatekeeper.Response, error)
}

// PluginClient in this case is the gatekeeper/core application. PluginClient
// is the interface that the user of this plugin sees and is simply a wrapper
// around *RPCClient. This is merely a wrapper which returns a clean interface
// with error interfaces instead of *gatekeeper.Error types
type PluginClient interface {
	internal.BasePlugin

	ModifyRequest(*gatekeeper.Request) (*gatekeeper.Request, error)
	ModifyResponse(*gatekeeper.Request, *gatekeeper.Response) (*gatekeeper.Response, error)
	ModifyErrorResponse(error, *gatekeeper.Request, *gatekeeper.Response) (*gatekeeper.Response, error)
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

func (p *pluginClient) ModifyRequest(req *gatekeeper.Request) (*gatekeeper.Request, error) {
	req, err := p.pluginRPC.ModifyRequest(req)
	return req, gatekeeper.ErrorToError(err)
}

func (p *pluginClient) ModifyResponse(req *gatekeeper.Request, resp *gatekeeper.Response) (*gatekeeper.Response, error) {
	return p.pluginRPC.ModifyResponse(req, resp)
}

func (p *pluginClient) ModifyErrorResponse(respErr error, req *gatekeeper.Request, resp *gatekeeper.Response) (*gatekeeper.Response, error) {
	return p.pluginRPC.ModifyErrorResponse(gatekeeper.NewError(respErr), req, resp)
}
