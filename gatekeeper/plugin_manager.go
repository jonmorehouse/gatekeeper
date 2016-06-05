package gatekeeper

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

type PluginManager interface {
	// start all plugins
	Start() error
	// stop all instances of the plugin
	Stop(time.Duration) error

	// fetches a reference to a plugin
	Get() (Plugin, error)
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
	//
}

func (p pluginManager) buildPlugin() (Plugin, error) {
	switch p.pluginType {
	case UpstreamPlugin:
		plugin, err := upstream.NewClient(p.opts.Name, p.opts.Cmd)
		if err != nil {
			return nil, err
		}
	default:
	}

	return nil, fmt.Errorf("Unsupported plugin type...")
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
			break // finished
		case time.Now().After(timeout):
			errs.Add(fmt.Errorf("Timed out waiting for plugins to stop..."))
			break
		default:
		}
	}

	return errs
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
