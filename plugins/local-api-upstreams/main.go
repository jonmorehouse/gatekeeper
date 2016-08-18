package main

import (
	"fmt"
	"log"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	"github.com/jonmorehouse/gatekeeper/gatekeeper/utils"
	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

// config exposes configuration that can be controlled via the command line for this plugin
type config struct {
	// upstream configuration
	UpstreamID        string   `flag:"upstream-id" default:"local-api-upstreams"`
	UpstreamName      string   `flag:"upstream-api-name" default:"local-api-upstreams"`
	UpstreamHostnames []string `flag:"upstream-api-hostnames"`
	UpstreamPrefixes  []string `flag:"upstream-api-prefixes" default:"_upstreams"`
	UpstreamProtocols []string `flag:"upstream-protocols" default:"http-public,http-internal"`

	// backend configuration
	BackendID string `flag:"upstream-backend-id" default:"local-api-upstreams:backend"`
	Port      int    `flag:"port"`
}

func upstreamFromConfig(c *config) (*gatekeeper.Upstream, error) {
	protocols, err := gatekeeper.ParseProtocols(c.UpstreamProtocols)
	if err != nil {
		return nil, err
	}

	return &gatekeeper.Upstream{
		ID:        gatekeeper.UpstreamID(c.UpstreamID),
		Name:      c.UpstreamName,
		Hostnames: c.UpstreamHostnames,
		Prefixes:  c.UpstreamPrefixes,
		Protocols: protocols,
	}, nil
}

func backendFromConfig(c *config) *gatekeeper.Backend {
	return &gatekeeper.Backend{
		ID:      gatekeeper.BackendID(c.BackendID),
		Address: fmt.Sprintf("http://localhost:%d", c.Port),
	}
}

// newPluign returns an instance of *plugin that implements the upstream_plugin.Plugin interface
func newPlugin() upstream_plugin.Plugin {
	return &plugin{}
}

// plugin implements the upstream_plugin.Plugin interface
type plugin struct {
	rpcManager upstream_plugin.Manager
	config     *config

	// internal state
	serviceContainer utils.ServiceContainer
	server           utils.Server
	upstreamID       gatekeeper.UpstreamID
	started          bool
}

func (p *plugin) Heartbeat() error {
	if !p.started {
		return gatekeeper.NotStartedErr
	}
	return nil
}

func (p *plugin) Start() error {
	if p.rpcManager == nil {
		return gatekeeper.NoManagerErr
	}
	if p.config == nil {
		return gatekeeper.NoConfigErr
	}

	// build out the internal state and server
	p.serviceContainer = utils.NewSyncedServiceContainer(p.rpcManager)
	p.server = utils.NewDefaultServer(NewAPI(p.serviceContainer))

	// start the server, returning an error if
	if p.config.Port > 0 {
		if err := p.server.StartOnPort(p.config.Port); err != nil {
			return err
		}
	} else {
		port, err := p.server.StartAnywhere()
		if err != nil {
			return err
		}
		p.config.Port = port
	}

	// create an upstream and backend and synchronize them to the container
	upstream, err := upstreamFromConfig(p.config)
	if err != nil {
		return err
	}
	backend := backendFromConfig(p.config)
	if err := p.serviceContainer.AddUpstream(upstream); err != nil {
		return err
	}
	if err := p.serviceContainer.AddBackend(upstream.ID, backend); err != nil {
		return err
	}

	p.started = true
	return nil
}

func (p *plugin) Stop() error {
	if !p.started {
		return nil
	}
	return p.server.Stop()
}

func (p *plugin) Configure(opts map[string]interface{}) error {
	var cfg config
	err := utils.ParseConfig(opts, &cfg)
	if err != nil {
		return err
	}

	p.config = &cfg
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
