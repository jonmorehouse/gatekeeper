package main

import (
	"log"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

func newPlugin() upstream_plugin.Plugin {
	return &plugin{}
}

// plugin implements the upstream_plugin.Plugin interface
type plugin struct {
	server     Server
	rpcManager upstream_plugin.Manager
}

func (p *plugin) Start() error                           { return nil }
func (p *plugin) Stop() error                            { return nil }
func (p *plugin) Heartbeat() error                       { return nil }
func (p *plugin) Configure(map[string]interface{}) error { return nil }

// UpstreamMetric accepts upstream specific metrics
func (p *plugin) UpstreamMetric(*gatekeeper.UpstreamMetric) error {
	return nil
}

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
