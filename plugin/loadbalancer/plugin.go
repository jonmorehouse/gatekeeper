package loadbalancer

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
	MagicCookieValue: "loadbalancer",
}

// Plugin is the interface which individual plugins implement and pass into the
// Plugin interface. This package is responsible for building the scaffolding
// around exposing the interface over RPC so that parent processes can talk to
// the LoadBalancer plugin as needed.
type Plugin interface {
	Start() error
	Stop() error
	Configure(map[string]interface{}) error
	Heartbeat() error

	AddBackend(shared.UpstreamID, *shared.Backend) error
	RemoveBackend(*shared.Backend) error
	GetBackend(shared.UpstreamID) (*shared.Backend, error)
}

// PluginClient is the interface which users of the
// loadbalancer_plugin.NewClient are exposed to. Specifically, this wraps a
// PluginRPC type and a plugin.Client allowing us to transparently make calls
// over RPC to the Plugin interface, as well as gives us the ability to kill
// the underlying connection to the plugin process.
type PluginClient interface {
	Configure(map[string]interface{}) error
	Heartbeat() error
	Start() error
	Stop() error

	// kill the underlying RPC connection, this should normally be a NOOP
	// under the hood, but in the case of a plugin Stop() timing out, we
	// can ensure that no ghost processes are left behind.
	Kill()

	AddBackend(shared.UpstreamID, *shared.Backend) error
	RemoveBackend(*shared.Backend) error
	GetBackend(shared.UpstreamID) (*shared.Backend, error)
}

// pluginClient implements the PluginClient interface, wrapping a PluginRPC and plugin.Client connection
type pluginClient struct {
	pluginRPC PluginRPC
	client    *plugin.Client
}

func NewPluginClient(pluginRPC PluginRPC, client *plugin.Client) PluginClient {
	return &pluginClient{
		pluginRPC: pluginRPC,
		client:    client,
	}
}

func (p *pluginClient) Start() error {
	return p.pluginRPC.Start()
}

func (p *pluginClient) Stop() error {
	return p.pluginRPC.Stop()
}

func (p *pluginClient) Configure(opts map[string]interface{}) error {
	return p.pluginRPC.Configure(opts)
}

func (p *pluginClient) Heartbeat() error {
	return p.pluginRPC.Heartbeat()
}

func (p *pluginClient) Kill() {
	p.client.Kill()
}

func (p *pluginClient) AddBackend(upstreamID shared.UpstreamID, backend *shared.Backend) error {
	return p.pluginRPC.AddBackend(upstreamID, backend)
}

func (p *pluginClient) RemoveBackend(backend *shared.Backend) error {
	return p.pluginRPC.RemoveBackend(backend)
}

func (p *pluginClient) GetBackend(upstreamID shared.UpstreamID) (*shared.Backend, error) {
	return p.pluginRPC.GetBackend(upstreamID)
}

// PluginRPC is an RPC compatible interface that exposes the Plugin interface
// in an RPC safe way.
type PluginRPC interface {
	Start() *shared.Error
	Stop() *shared.Error
	Configure(map[string]interface{}) *shared.Error
	Heartbeat() *shared.Error

	AddBackend(shared.UpstreamID, *shared.Backend) *shared.Error
	RemoveBackend(*shared.Backend) *shared.Error
	GetBackend(shared.UpstreamID) (*shared.Backend, *shared.Error)
}

type PluginDispenser struct {
	// this is the actual plugin's implementation of the plugin interface.
	// Everything in this package just proxies requests to this object.
	impl Plugin
}

func (d PluginDispenser) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &RPCServer{broker: b, impl: d.impl}, nil
}

func (d PluginDispenser) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &RPCClient{broker: b, client: c}, nil
}

// This is the method that a plugin will call to start serving traffic over the
// plugin interface. Specifically, this will start the RPC server and register
// etc.
func RunPlugin(name string, impl Plugin) error {
	pluginDispenser := PluginDispenser{impl: impl}

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
		client.Kill()
		return nil, fmt.Errorf("Unable to cast dispensed plugin to a PluginRPC type")
	}

	return NewPluginClient(pluginRPC, client), nil
}
