package loadbalancer

import (
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

type Opts map[string]interface{}

// this is the interface that gatekeeper sees
type Plugin interface {
	// standard plugin methods
	Start() error
	Stop() error
	// this isn't Opts, because we want to make this as general as possible
	// for expressiveness between different plugins
	Configure(map[string]interface{}) error
	// Heartbeat is called by a plugin manager in the primary application periodically
	Heartbeat() error

	// loadbalancer specific methods
	AddBackend(shared.UpstreamID, shared.Backend) error
	RemoveBackend(shared.Backend) error
	GetBackend(shared.UpstreamID) (shared.Backend, error)
}

type PluginDispenser struct {
	// this is the actual plugin's implementation of the plugin interface.
	// Everything in this package just proxies requests to this object.
	impl Plugin
}

func (d PluginDispenser) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &PluginRPCServer{broker: b, impl: d.impl}, nil
}

func (d PluginDispenser) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &PluginRPCClient{broker: b, client: c}, nil
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

func NewClient(name string, cmd string) (Plugin, error) {
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

	return rawPlugin.(Plugin), nil
}
