package main

import (
	"errors"
	"log"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	"github.com/jonmorehouse/gatekeeper/gatekeeper/utils"
	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

var (
	InvalidConfigErr          = errors.New("Invalid config")
	UnparseableConfigErr      = errors.New("unable to parse config")
	UpstreamNameRequiredError = errors.New("Upstream name required")
)

type config struct {
	ConfigPath string `field:"upstream-config" required:"true"`
}

func newUpstreamPlugin() upstream_plugin.Plugin {
	return &upstreamPlugin{}
}

type upstreamPlugin struct {
	rpcManager upstream_plugin.Manager

	serviceContainer utils.ServiceContainer
	config           *config
}

func (u *upstreamPlugin) Configure(opts map[string]interface{}) error {
	var cfg config
	if err := utils.ParseConfig(opts, &cfg); err != nil {
		return err
	}
	u.config = &cfg

	// parse services out of the config file
	serviceDef, err := parseConfig(u.config.ConfigPath)
	if err != nil {
		return err
	}

	// sync services into the ServiceContainer
	u.serviceContainer = utils.NewSyncedServiceContainer(u.rpcManager)
	return syncServices(serviceDef, u.serviceContainer)
}

func (u *upstreamPlugin) SetManager(manager upstream_plugin.Manager) error {
	u.rpcManager = manager
	return nil
}

func (u *upstreamPlugin) Start() error                                    { return nil }
func (u *upstreamPlugin) Stop() error                                     { return nil }
func (u *upstreamPlugin) Heartbeat() error                                { return nil }
func (u *upstreamPlugin) UpstreamMetric(*gatekeeper.UpstreamMetric) error { return nil }

func main() {
	upstreamPlugin := newUpstreamPlugin()
	if err := upstream_plugin.RunPlugin("static-upstreams", upstreamPlugin); err != nil {
		log.Fatal(err)
	}
}
