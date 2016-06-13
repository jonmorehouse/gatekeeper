package request

import (
	"fmt"
	"net/rpc"
	"os/exec"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/shared"
)

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "gatekeeper|plugin-type",
	MagicCookieValue: "modifier",
}

// Plugin is the interface that plugins implement and pass along to the
// RunPlugin method. Behind the scenes, this package will wrap the interface in
// the correct places so as to expose these methods over RPC.
type Plugin interface {
	Configure(map[string]interface{}) error
	Heartbeat() error
	Start() error
	Stop() error

	ModifyRequest(*shared.Request) (*shared.Request, error)
	ModifyResponse(*shared.Request, *shared.Response) (*shared.Response, error)
}

// PluginClient is the interface that is exposed to users of this plugin.
// Specifically, this wraps the underlying `plugin.Client` type as well as the
// actual RPCClient type itself.
type PluginClient interface {
	// standard plugin methods for configuring / start / stop
	Configure(map[string]interface{}) error
	Heartbeat() error
	Start() error
	Stop() error

	// kill the underlying `goplugin` client forcefully
	Kill()

	ModifyRequest(*shared.Request) (*shared.Request, error)
	ModifyResponse(*shared.Request, *shared.Response) (*shared.Response, error)
}

type pluginClient struct {
	// the underlying plugin connection that manages the plugin lifecycle
	// using `go-plugin`
	client *plugin.Client

	// interface that is exposed over RPC
	pluginRPC PluginRPC
}

func NewPluginClient(client *plugin.Client, pluginRPC PluginRPC) PluginClient {
	return &pluginClient{
		client:    client,
		pluginRPC: pluginRPC,
	}
}

func (p *pluginClient) Configure(opts map[string]interface{}) error {
	return shared.ErrorToError(p.pluginRPC.Configure(opts))
}

func (p *pluginClient) Heartbeat() error {
	return shared.ErrorToError(p.pluginRPC.Heartbeat())
}

func (p *pluginClient) Start() error {
	return shared.ErrorToError(p.pluginRPC.Start())
}

func (p *pluginClient) Stop() error {
	return shared.ErrorToError(p.pluginRPC.Stop())
}

func (p *pluginClient) Kill() {
	p.client.Kill()
}

func (p *pluginClient) ModifyRequest(req *shared.Request) (*shared.Request, error) {
	req, err := p.pluginRPC.ModifyRequest(req)
	return req, shared.ErrorToError(err)
}

func (p *pluginClient) ModifyResponse(req *shared.Request, resp *shared.Response) (*shared.Response, error) {
	resp, err := p.pluginRPC.ModifyResponse(req, resp)
	return resp, shared.ErrorToError(err)
}

// this is the PluginRPC type that we actually implement as the "plugin"
// interface. We abstract the error handling away around these so as to allow
// for us passing concrete error types around.
type PluginRPC interface {
	Configure(map[string]interface{}) *shared.Error
	Heartbeat() *shared.Error
	Start() *shared.Error
	Stop() *shared.Error

	ModifyRequest(*shared.Request) (*shared.Request, *shared.Error)
	ModifyResponse(*shared.Request, *shared.Response) (*shared.Response, *shared.Error)
}

type PluginDispenser struct {
	RequestPlugin Plugin
}

func (d PluginDispenser) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &RPCServer{broker: b, impl: d.RequestPlugin}, nil
}

func (d PluginDispenser) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &RPCClient{broker: b, client: c}, nil
}

func RunPlugin(name string, requestPlugin Plugin) error {
	pluginDispenser := PluginDispenser{RequestPlugin: requestPlugin}
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			name: &pluginDispenser,
		},
	})
	return nil
}

func NewClient(name string, cmd string) (PluginClient, error) {
	pluginDispenser := PluginDispenser{}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			name: &pluginDispenser,
		},
		Cmd: exec.Command(cmd),
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, err
	}

	rawPlugin, err := rpcClient.Dispense(name)
	if err != nil {
		client.Kill()
		return nil, err
	}

	pluginRPC, ok := rawPlugin.(PluginRPC)
	if !ok {
		return nil, fmt.Errorf("Unable to cast raw plugin to PluginClient")
	}

	return NewPluginClient(client, pluginRPC), nil
}
