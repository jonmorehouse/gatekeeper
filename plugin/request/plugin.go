package request

import (
	"net/rpc"
	"os/exec"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/shared"
)

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "gatekeeper|plugin-type",
	MagicCookieValue: "request",
}

// this is the type that gatekeeper and plugins interact with. We abstract the
// error handling behind the scenes to ensure that we pass only *shared.Error
// types over the wire.
type Plugin interface {
	Configure(map[string]interface{}) error
	Heartbeat() error
	Start() error
	Stop() error

	ModifyRequest(*shared.Request) (*shared.Request, error)
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

func NewClient(name string, cmd string) (PluginRPC, func(), error) {
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
		return nil, func() {}, err
	}

	rawPlugin, err := rpcClient.Dispense(name)
	if err != nil {
		client.Kill()
		return nil, func() {}, err
	}

	// TODO change this to return Plugin and not PluginRPC once all other
	// plugins are compatbile and gatekeeper.plugin_manager.
	return rawPlugin.(PluginRPC), func() { client.Kill() }, nil

}
