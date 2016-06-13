package gatekeeper

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	loadbalancer_plugin "github.com/jonmorehouse/gatekeeper/plugin/loadbalancer"
	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

type PluginManager interface {
	// start all plugins, running them in the background...
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
	pluginName string
	pluginCmd  string
	opts       map[string]interface{}

	// count is the number of instances that this manager should manage for this plugin
	count     uint
	instances []Plugin
	killers   []func()
}

func NewPluginManager(pluginCmd string, opts map[string]interface{}, count uint, pluginType PluginType) PluginManager {
	return &pluginManager{
		pluginType: pluginType,
		pluginCmd:  pluginCmd,
		pluginName: filepath.Base(pluginCmd),
		opts:       opts,
		count:      count,
		instances:  make([]Plugin, 0, count),
		killers:    make([]func(), 0, count),
	}
}

func (p *pluginManager) Start() error {
	// starts and configures the plugins so that they work correctly
	errs := NewAsyncMultiError()
	var wg sync.WaitGroup

	for i := uint(0); i < p.count; i++ {
		wg.Add(1)
		instance, killer, err := p.buildPlugin()
		if err != nil {
			errs.Add(err)

			// most of the time, when an error occurs, it is
			// because we specified an invalid plugin of this type.
			// However, if there is a problem starting a plugin,
			// then we need kill off the plugin before exiting.
			if instance != nil {
				killer()
			}
			wg.Done()
			continue
		}
		p.instances = append(p.instances, instance)
		p.killers = append(p.killers, killer)

		go func(plugin Plugin) {
			defer wg.Done()
			if err := plugin.Configure(p.opts); err != nil {
				errs.Add(err)
				return
			}

			if err := plugin.Start(); err != nil {
				errs.Add(err)
			}
		}(instance)
	}

	wg.Wait()
	return errs.ToErr()
}

func (m pluginManager) buildPlugin() (Plugin, func(), error) {
	if m.pluginType == UpstreamPlugin {
		return upstream_plugin.NewClient(m.pluginName, m.pluginCmd)
	}
	if m.pluginType == LoadBalancerPlugin {
		return loadbalancer_plugin.NewClient(m.pluginName, m.pluginCmd)
	}
	if m.pluginType == ModifierPlugin {
		return modifier_plugin.NewClient(m.pluginName, m.pluginCmd)
	}
	if m.pluginType == EventPlugin {
		return event_plugin.NewClient(m.pluginName, m.pluginCmd)
	}

	return nil, nil, fmt.Errorf("INVALID_PLUGIN_TYPE")
}

func (p *pluginManager) Stop(duration time.Duration) error {
	errs := NewAsyncMultiError()

	// stop each plugin in a goroutine
	var wg sync.WaitGroup
	for _, instance := range p.instances {
		wg.Add(1)
		go func(p Plugin) {
			if err := p.Stop(); err != nil {
				errs.Add(err)
			}

			// stop the plugin client

			wg.Done()
		}(instance)
	}

	// wait for the plugins to all finish, or otherwise timeout
	doneCh := make(chan interface{})
	go func() {
		wg.Wait()
		doneCh <- struct{}{}
	}()

	// wait for the waitGroup to finish and signal or exit with a timeout
	for {
		select {
		case <-doneCh:
			goto cleanup
		case <-time.After(duration):
			errs.Add(fmt.Errorf("Timed out waiting for plugins to stop..."))
			goto cleanup
		}
	}

cleanup:
	// call all kill functions for all plugin clients, ensuring that we
	// shut down any loose rpc connections
	for _, killer := range p.killers {
		killer()
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
