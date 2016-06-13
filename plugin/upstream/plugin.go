package upstream

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
	MagicCookieValue: "upstream",
}

// Plugin interface is the interface that plugins implement in order to
// register and deregister backends and upstreams with the parent server.
// Behind the scenes, the `RunPlugin` method takes this interface and wraps it
// with the correct RPC tooling in order to expose the correct functionality.
type Plugin interface {
	// Pass along configuration options that are loosely defined from the
	// parent plugin. Using anything in this dictionary needs to be done in
	// as safe a way as possible!
	Configure(map[string]interface{}) error

	// Return an error if the plugin is not acting properly and/or needs to
	// be rebooted by the parent.
	Heartbeat() error

	// Start the plugin, passing the Manager interface along through. The
	// Manager exposes methods for calling back into the parent process.
	Start(Manager) error
	Stop() error
}

// PluginClient is the interface that users of an upstream_plugin get access
// too. Specifically, this wraps the PluginRPC and a the goplugin.Client type
// in order to ensure that we can control both the Plugin and and the
// underlying plugin process architecture.
type PluginClient interface {
	// configures the plugin with options from the parent machine. Behind
	// the scenes, the parent will pass in a manager implementation here
	// which is then passed to the plugin implementer's start method. This
	// is a little magical, but its controlled magic!
	Configure(map[string]interface{}) error
	Heartbeat() error

	// NOTE this differs from the Plugin implementer side to make this a
	// standard plugin and to work with the gatekeeper.PluginManager type.
	Start() error
	Stop() error

	// Stop the plugin, disconnecting the process and killing it.
	Kill()
}

type pluginClient struct {
	// the underlying plugin connection that manages the plugin lifecycle
	// using `go-plugin`
	client *plugin.Client

	// the interface that is exposed over RPC, mapping back to the
	// implementer passed in on the plugin side.
	pluginRPC PluginRPC
}

func NewPluginClient(pluginRPC PluginRPC, client *plugin.Client) PluginClient {
	return &pluginClient{
		pluginRPC: pluginRPC,
		client:    client,
	}
}

func (p *pluginClient) Configure(opts map[string]interface{}) error {
	return p.pluginRPC.Configure(opts)
}

func (p *pluginClient) Heartbeat() error {
	return p.pluginRPC.Heartbeat()
}

func (p *pluginClient) Start() error {
	return p.pluginRPC.Start()
}

func (p *pluginClient) Stop() error {
	return p.pluginRPC.Stop()
}

func (p *pluginClient) Kill() {
	p.client.Kill()
}

// PluginRPC is an interface which is used to interact with the above Plugin
// interface over RPC. Specifically, this ensures that concrete types are
// passed over RPC.
type PluginRPC interface {
	Configure(map[string]interface{}) *shared.Error
	Heartbeat() *shared.Error
	Start() *shared.Error
	Stop() *shared.Error
}

// PluginDispenser dispense a PluginRPC type which is used and configured with
// the `goplugin` package.
type PluginDispenser struct {
	// UpstreamPlugin is the plugin's implementation of our Plugin
	// interface.
	UpstreamPlugin Plugin
}

func (d PluginDispenser) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &PluginRPCServer{broker: b, impl: d.UpstreamPlugin}, nil
}

func (d PluginDispenser) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &PluginRPCClient{broker: b, client: c}, nil
}

func RunPlugin(name string, upstreamPlugin Plugin) error {
	pluginDispenser := PluginDispenser{UpstreamPlugin: upstreamPlugin}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			name: &pluginDispenser,
		},
	})
	return nil
}

// Returns a new Client which wraps the PluginRPC type, exposing the plugin
// implementer over RPC
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
		return nil, fmt.Errorf("Invalid Plugin dispensed from PluginDispenser, unable to be cast as PluginRPC")
	}

	return NewPluginClient(pluginRPC, client), nil
}
