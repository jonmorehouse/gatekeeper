package main

import (
	"log"
	"strconv"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
	"github.com/jonmorehouse/gatekeeper/utils"
)

func newPlugin() upstream_plugin.Plugin {
	return &plugin{}
}

// plugin implements the upstream_plugin.Plugin interface
type plugin struct {
	serviceContainer utils.ServiceContainer
	server           Server
	upstreamID       gatekeeper.UpsteamID
}

func (p *plugin) Heartbeat() error { return nil }

func (p *plugin) Start() error {
	if p.rpcManager == nil {
		return gatekeeper.NoManagerErr
	}

	// create the server and start it in the background
	container := utils.NewSyncedServerContainer(p.rpcManager)

	// build out an upstream and its backend
	if err := p.rpcManager.AddUpstream(p.upstream); err != nil {
		return err
	}

	//p.backend = &gatekeeper.Backend{
	//ID: "local-api-upstreams:backend",
	//Address: fmt.Sprintf("http://localhost:%d", p.server.GetP,

	//}
	if err := p.rpcManager.AddBackend(p.backend); err != nil {
		return err
	}

	return nil
}

func (p *plugin) Stop() error {
	return p.server.Stop()
}

func (p *plugin) Configure(opts map[string]interface{}) error {
	portArg, ok := opts["upstream-api-port"]
	if ok {
		port, err := strconv.ParseInt(portArg.(string), 10, 64)
		if err != nil {
			return err
		}
		p.port = int(port)
	}

	return nil
}

// UpstreamMetric accepts upstream specific metrics
func (p *plugin) UpstreamMetric(*gatekeeper.UpstreamMetric) error { return nil }

// SetManager accepts an RPCManager which allows the plugin to write upstreams
// back into the parent process.
func (p *plugin) SetManager(rpcManager upstream_plugin.Manager) error {
	p.rpcManager = rpcManager
	return nil
}

func main() {
	plugin := newPlugin()
	if err := upstream_plugin.RunPlugin("local-api-upstreams", plugin); err != nil {
		log.Fatal(err)
	}
}
