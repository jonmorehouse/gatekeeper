package metric

import (
	"fmt"
	"os/exec"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/internal"
)

var Handshake = internal.NewHandshakeConfig("metric")

func NewClient(name string, cmd string) (PluginClient, error) {
	dispenser := &Dispenser{}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			name: dispenser,
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

	pluginRPC, ok := rawPlugin.(*RPCClient)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("Unable to cast plugin to the correct type")
	}

	return NewPluginClient(pluginRPC, client), nil

}

func RunPlugin(name string, impl Plugin) error {
	dispenser := &Dispenser{impl}
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			name: dispenser,
		},
	})
	return nil
}
