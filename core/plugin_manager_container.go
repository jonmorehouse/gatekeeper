package core

import "sync"

// PluginManagerContainer is a wrapper around a set of Plugins
type PluginManagerContainer map[PluginType][]PluginManager

var allPluginTypes = []PluginType{
	UpstreamPlugin,
	RouterPlugin,
	LoadBalancerPlugin,
	ModifierPlugin,
	MetricPlugin,
}

// BuildPlugins accepts an options struct and a MetricWriter and uses them to
// build out a set of plugins that corresponds to the requested plugins per the
// options.
func buildPlugins(options *Options, metricWriter MetricWriter) PluginManagerContainer {
	plugins := make(PluginManagerContainer)

	if !options.UseLocalRouter {
		plugin := NewPluginManager(options.RouterPlugin, options.RouterPluginArgs, RouterPlugin, metricWriter)
		plugins[RouterPlugin] = []PluginManager{plugin}
	}

	if !options.UseLocalLoadBalancer {
		plugin := NewPluginManager(options.LoadBalancerPlugin, options.LoadBalancerPluginArgs, LoadBalancerPlugin, metricWriter)
		plugins[LoadBalancerPlugin] = []PluginManager{plugin}
	}

	for _, plugin := range options.UpstreamPlugins {
		plugins[UpstreamPlugin] = append(plugins[UpstreamPlugin], NewPluginManager(plugin, options.UpstreamPluginArgs, UpstreamPlugin, metricWriter))
	}

	for _, plugin := range options.ModifierPlugins {
		plugins[ModifierPlugin] = append(plugins[ModifierPlugin], NewPluginManager(plugin, options.ModifierPluginArgs, ModifierPlugin, metricWriter))
	}

	for _, plugin := range options.MetricPlugins {
		plugins[MetricPlugin] = append(plugins[MetricPlugin], NewPluginManager(plugin, options.MetricPluginArgs, MetricPlugin, metricWriter))
	}
	return plugins
}

// asyncPluginFilter accepts a callback and a container and runs the callback,
// asynchronously on all of the elements of the specified typs in the
// container. It collects errors and returns the resolved multi-error.
func asyncPluginFilter(plugins PluginManagerContainer, typs []PluginType, cb func(PluginManager) error) error {
	if typs == nil {
		typs = allPluginTypes
	}

	errs := NewMultiError()

	// for each type of plugin, call the callback for each one as specified.
	var wg sync.WaitGroup
	for _, typ := range typs {
		managers, ok := plugins[typ]
		if !ok {
			continue
		}

		for _, manager := range managers {
			wg.Add(1)
			go func(manager PluginManager) {
				defer wg.Done()
				errs.Add(cb(manager))
			}(manager)
		}
	}

	wg.Wait()
	return errs.ToErr()
}

func syncPluginFilter(plugins PluginManagerContainer, typs []PluginType, cb func(PluginManager) error) error {
	if typs == nil {
		typs = allPluginTypes
	}

	errs := NewMultiError()

	// for each type of plugin, call the callback and track the specified error for it.
	for _, typ := range typs {
		managers, ok := plugins[typ]
		if !ok {
			continue
		}

		for _, manager := range managers {
			errs.Add(cb(manager))
		}
	}

	return errs.ToErr()
}
