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

// this is the interface that public plugins need to implement.
type Plugin interface {
	// Pass along configuration options that are loosely defined from the
	// parent plugin. Using anything in this dictionary needs to be done in
	// as safe a way as possible!
	Configure(map[string]interface{}) *shared.Error

	// Return an error if the plugin is not acting properly and/or needs to
	// be rebooted by the parent.
	Heartbeat() *shared.Error

	// Start the plugin. Note the Manager interface is used to send
	// Upstream and Backend events back into the parent process.
	Start(Manager) *shared.Error
	Stop() *shared.Error
}

// this is the interface that clients of this plugin use
type PluginClient interface {
	// configures the plugin with options from the parent machine. Behind
	// the scenes, the parent will pass in a manager implementation here
	// which is then passed to the plugin implementer's start method. This
	// is a little magical, but its controlled magic!
	Configure(map[string]interface{}) *shared.Error
	Heartbeat() *shared.Error

	// NOTE this differs from the Plugin implementer side to make this a
	// standard plugin and to work with the gatekeeper.PluginManager type.
	Start() *shared.Error
	Stop() *shared.Error
}

// this is the pluginwrapper that individual plugins will use to create their
// instance of a go-plugin server
type PluginDispenser struct {
	// this is only used for servers (actualy plugin implementers)
	UpstreamPlugin Plugin
}

func (d PluginDispenser) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &PluginRPCServer{broker: b, impl: d.UpstreamPlugin}, nil
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

	pluginClient, ok := rawPlugin.(PluginClient)
	if !ok {
		return nil, fmt.Errorf("Unable to cast dispensed plugin into a PluginClient")
	}
	return pluginClient, nil
}
