package gatekeeper

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	loadbalancer_plugin "github.com/jonmorehouse/gatekeeper/plugin/loadbalancer"
	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

type PluginManager interface {
	// start all plugins
	Start() error
	// stop all instances of the plugin
	Stop(time.Duration) error

	// fetches a reference to a plugin
	Get() (Plugin, error)

	// fetch all references to all plugins
	All() ([]Plugin, error)
}

type pluginManager struct {
	pluginType PluginType
	opts       PluginOpts

	// count is the number of instances that this manager should manage for this plugin
	count     uint
	instances []Plugin
}

func NewPluginManager(pluginType PluginType, opts PluginOpts, count uint) PluginManager {
	if count != 1 {
		log.Fatal("PluginManager only supports a single plugin for now")
		return nil
	}

	return &pluginManager{
		pluginType: pluginType,
		opts:       opts,
		count:      1,
		instances:  make([]Plugin, 0, 1),
	}
}

func (p *pluginManager) Start() error {
	// starts and configures the plugins so that they work correctly
	errs := NewAsyncMultiError()
	var wg sync.WaitGroup
	defer wg.Wait()

	for i := uint(0); i < p.count; i++ {
		wg.Add(1)
		instance, err := p.buildPlugin()
		if err != nil {
			errs.Add(err)
			wg.Done()
			continue
		}

		go func(plugin Plugin) {
			defer wg.Done()

			if err := plugin.Configure(p.opts.Opts); err != nil {
				errs.Add(err)
				return
			}

			if err := plugin.Start(); err != nil {
				errs.Add(err)
			}
		}(instance)
	}

	return errs.ToErr()
}

func (m pluginManager) buildPlugin() (Plugin, error) {
	if m.pluginType == UpstreamPlugin {
		return upstream_plugin.NewClient(m.opts.Name, m.opts.Cmd)
	}
	if m.pluginType == LoadBalancerPlugin {
		return loadbalancer_plugin.NewClient(m.opts.Name, m.opts.Cmd)
	}

	return nil, fmt.Errorf("INVALID_PLUGIN_TYPE")
}

func (p *pluginManager) Stop(duration time.Duration) error {
	timeout := time.Now().Add(duration)
	errs := NewAsyncMultiError()

	// stop each plugin in a goroutine
	var wg sync.WaitGroup
	for _, instance := range p.instances {
		wg.Add(1)
		go func(p Plugin) {
			if err := p.Stop(); err != nil {
				errs.Add(err)
			}
			wg.Done()
		}(instance)
	}

	// wait for the plugins to all finish, or otherwise timeout
	doneCh := make(chan interface{})
	for {
		select {
		case <-doneCh:
			return errs.ToErr()
		default:
			if time.Now().After(timeout) {
				errs.Add(fmt.Errorf("Timed out waiting for plugins to stop..."))
				return errs.ToErr()
			}
		}
	}

	return errs.ToErr()
}

func (p *pluginManager) Get() (Plugin, error) {
	if len(p.instances) == 0 {
		return nil, fmt.Errorf("No plugin instances...")
	}

	idx := rand.Intn(len(p.instances))
	return p.instances[idx], nil
}

func (p *pluginManager) All() ([]Plugin, error) {
	if len(p.instances) == 0 {
		return []Plugin{}, fmt.Errorf("No plugin instances")
	}
	return p.instances, nil
}
