package gatekeeper

import (
	"sync"

	"github.com/jonmorehouse/gatekeeper/shared"
)

type Plugin interface {
	Start() error
	Stop() error
	Configure(map[string]interface{}) error
	Heartbeat() error
	Kill()
}

type PluginType uint

const (
	UpstreamPlugin PluginType = iota + 1
	LoadBalancerPlugin
	ModifierPlugin
	MetricPlugin
)

var pluginTypeMapping = map[PluginType]string{
	UpstreamPlugin:     "upstream-plugin",
	LoadBalancerPlugin: "loadbalancer-plugin",
	ModifierPlugin:     "modifier-plugin",
	MetricPlugin:       "event-plugin",
}

func (p PluginType) String() string {
	desc, ok := pluginTypeMapping[p]
	if !ok {
		shared.ProgrammingError("PluginType string mapping not found")
	}
	return desc
}

func EachPluginManager(pluginManagers []PluginManager, method func(PluginManager) error) error {
	var wg sync.WaitGroup
	errs := NewMultiError()

	for _, pluginManager := range pluginManagers {
		wg.Add(1)

		go func(plugin PluginManager) {
			defer wg.Done()

			err := method(pluginManager)
			if err != nil {
				errs.Add(err)
			}
		}(pluginManager)
	}

	wg.Wait()
	return errs.ToErr()
}

// TODO implement this!
func EachPlugin(pluginManager PluginManager, method func(Plugin) error) error {
	return nil
}
