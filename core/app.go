package core

import (
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

type App struct {
	servers         []Server
	plugins         PluginManagerContainer
	components      []interface{}
	metricWriter    MetricWriter
	upstreamManager UpstreamManager
}

func New(options Options) (*App, error) {
	// build out global metrics based components
	metricWriter := NewMetricWriter(int(options.MetricBufferSize), options.MetricFlushInterval)
	metricWriter.EventMetric(&gatekeeper.EventMetric{
		Timestamp: time.Now(),
		Event:     gatekeeper.AppStartedEvent,
	})
	profiler := NewProfiler(metricWriter, options.ProfilerInterval)

	// build out plugins from the given configuration
	plugins := buildPlugins(&options, metricWriter)
	if plugins[UpstreamPlugin] == nil {
		return nil, ConfigurationError
	}

	// build out upstream manager and event pipeline
	broadcaster := NewBroadcaster()
	upstreamManager := NewUpstreamManager(broadcaster, metricWriter)

	// build out router
	var router Router
	if plugins[RouterPlugin] == nil {
		router = NewLocalRouter(broadcaster, metricWriter)
	} else {
		router = NewPluginRouter(broadcaster, plugins[RouterPlugin][0])
	}

	// build out loadbalancer
	var loadBalancer LoadBalancer
	if plugins[LoadBalancerPlugin] == nil {
		loadBalancer = NewLocalLoadBalancer(broadcaster)
	} else {
		loadBalancer = NewPluginLoadBalancer(broadcaster, plugins[LoadBalancerPlugin][0])
	}

	// build out the modifier, pivoting between a local modifier or a
	// plugin based one by inspecting options
	var modifier Modifier
	if plugins[ModifierPlugin] == nil {
		modifier = NewLocalModifier()
	} else {
		modifier = NewPluginModifier(plugins[ModifierPlugin])
	}

	proxier := NewProxier(modifier, metricWriter)
	servers := buildServers(options, router, loadBalancer, modifier, proxier, metricWriter)

	return &App{
		components: []interface{}{
			profiler,
			modifier,
			router,
			loadBalancer,
			upstreamManager,
		},
		plugins:         plugins,
		servers:         servers,
		metricWriter:    metricWriter,
		upstreamManager: upstreamManager,
	}, nil
}

func (a *App) Start() error {
	a.eventMetric(gatekeeper.AppStartedEvent)

	// write out a method that will filter out starters
	if err := filterStarters(a.components, func(i starter) error {
		return i.start()
	}); err != nil {
		return err
	}

	// start each of the plugins; handling the case of an upstream plugin particularly carefully.
	types := []PluginType{MetricPlugin, ModifierPlugin, LoadBalancerPlugin, RouterPlugin, UpstreamPlugin}
	err = syncPluginManagerFilter(a.plugins, types, func(manager PluginManager) error {
		if err := manager.Build(); err != nil {
			return err
		}

		// if this is an upstreamPlugin, then the Manager needs to be set accordingly
		if UpstreamPlugin == manager.Type() {
			if err := manager.CallOnce("SetManager", func(plugin Plugin) error {
				upstreamPlugin, ok := plugin.(upstream_plugin.PluginClient)
				if !ok {
					return InvalidPluginErr
				}
				return upstreamPlugin.SetManager(a.upstreamManager)
			}); err != nil {
				return err
			}
		}

		a.metricWriter.AddPlugin(manager)
		return nil
	})
	if err != nil {
		return err
	}

	// start the metricWriter, so it can begin flushing metrics
	if err := a.metricWriter.Start(); err != nil {
		return err
	}

	return filterServers(a.servers, func(i starter) error {
		return i.start()
	})
}

func (a *App) Stop(duration time.Duration) error {
	a.eventMetric(gatekeeper.AppStoppedEvent)
	errs := NewMultiError()

	// stop servers
	errs.Add(filterServers(i.servers, func(server Server) error {
		return server.Stop(duration)
	}))

	// stop the plugins, apart from the metricWriter
	typs := []PluginType{UpstreamPlugin, RouterPlugin, LoadBalancerPlugin, ModifierPlugin}
	errs.Add(asyncFilterPlugins(a.plugins, typs, func(pluginManager PluginManager) error {
		return pluginManager.Stop(duration)
	}))

	// stop all other components
	errs.Add(filterGracefulStopper(i.components, func(i gracefulStopper) error {
		return i.Stop(duration)
	}))
	errs.Add(filterStopper(i.components, func(i stopper) error {
		return i.Stop()
	}))

	// stop the metricWriter plugins
	errs.Add(asyncFilterPlugins(a.plugins, []PluginType{MetricWriter}, func(pluginManager PluginManager) error {
		return pluginManager.Stop(duration)
	}))
	return errs.ToErr()
}

func (a *App) eventMetric(event gatekeeper.Event) {
	a.metricWriter.EventMetric(&gatekeeper.EventMetric{
		Timestamp: time.Now(),
		Event:     event,
	})
}
