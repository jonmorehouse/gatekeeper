package upstream

import (
	"net/rpc"
	"os/exec"

	"github.com/hashicorp/go-plugin"
)

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "gatekeeper|plugin-type",
	MagicCookieValue: "upstream",
}

// this is the interface that public plugins have to call
type Plugin interface {
	// configures the plugin with options from the parent machine
	Configure(map[string]interface{}) error

	FetchUpstreams() ([]Upstream, error)
	FetchUpstreamBackends(UpstreamID) ([]Backend, error)

	AddManager(Manager) error

	// this method should start a background goroutine which will emit
	// messages to the parent by calling methods on the manager type
	Start() error
	// this should wait a maximum of N seconds ...
	Stop() error
}

// this is the pluginwrapper that individual plugins will use to create their
// instance of a go-plugin server
type PluginDispenser struct {
	// this is only used for servers (actualy plugin implementers)
	UpstreamPlugin Plugin
}

func (d PluginDispenser) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &PluginRPCServer{broker: b, plugin: d.UpstreamPlugin}, nil
}

func (d PluginDispenser) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &PluginRPCClient{broker: b, client: c}, nil
}

// NOTE this should only be run from the plugin binaries, not from the gatekeeper api itself
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
