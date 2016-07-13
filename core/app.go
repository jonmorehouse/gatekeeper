package core

import (
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type startStop interface {
	Start() error
	Stop(time.Duration) error
}

type App struct {
	// the server type adheres to the startStop interface, by convenience.
	servers      []startStopper
	plugins      map[PluginType][]PluginManager
	components   []startStopper
	metricWriter MetricWriter
}

func buildPlugins(options *Options, metricWriter MetricWriter) (map[PluginType][]PluginManager, error) {
	return map[PluginType][]PluginManager(nil), nil

}

func New(options Options) (*App, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	metricWriter := NewMetricWriter(int(options.MetricBufferSize), options.MetricFlushInterval)
	metricWriter.EventMetric(&gatekeeper.EventMetric{
		Timestamp: time.Now(),
		Event:     gatekeeper.AppStartedEvent,
	})

	plugins, err := buildPlugins(&options, metricWriter)
	if err != nil {
		return nil, err
	}

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

	// build out modifier
	var modifier Modifier
	if plugins[ModifierPlugin] == nil {
		modifier = NewLocalModifier()
	} else {
		modifier = NewPluginModifier(plugins[ModifierPlugin])
	}

	proxier := NewProxier(modifier, metricWriter)

	// configure MetricWriter
	for _, instances := range plugins {
		for _, plugin := range instances {
			metricWriter.AddPlugin(plugin)
		}
	}

	// build out servers
	servers := make([]startStopper, 0, 0)
	if options.HTTPPublic {
		servers = append(servers, NewHTTPServer(
			gatekeeper.HTTPPublic,
			options.HTTPPublicPort,
			router,
			loadBalancer,
			modifier,
			proxier,
			metricWriter,
		))
	}

	if options.HTTPInternal {
		servers = append(servers, NewHTTPServer(
			gatekeeper.HTTPInternal,
			options.HTTPInternalPort,
			router,
			loadBalancer,
			modifier,
			proxier,
			metricWriter,
		))
	}

	if options.HTTPSPublic {
		servers = append(servers, NewHTTPSServer(
			gatekeeper.HTTPSPublic,
			options.HTTPSPublicPort,
			router,
			loadBalancer,
			modifier,
			proxier,
			metricWriter,
		))
	}

	if options.HTTPSInternal {
		servers = append(servers, NewHTTPSServer(
			gatekeeper.HTTPSInternal,
			options.HTTPSInternalPort,
			router,
			loadBalancer,
			modifier,
			proxier,
			metricWriter,
		))
	}

	// build out servers
	return &App{
		components: []startStopper{
			modifier,
			upstreamManager,
			loadBalancer,
			router,
		},
		plugins:      plugins,
		servers:      servers,
		metricWriter: metricWriter,
	}, nil
}

func (a *App) Start() error {
	a.eventMetric(gatekeeper.AppStartedEvent)

	// start internal components
	if err := CallWith(startStoppersToInterfaces(a.components), func(i interface{}) error {
		return i.(startStopper).Start()
	}); err != nil {
		return err
	}

	// start all plugins
	for _, pluginType := range []PluginType{MetricPlugin, UpstreamPlugin, ModifierPlugin, LoadBalancerPlugin, RouterPlugin} {
		if err := CallWith(pluginManagersToInterfaces(a.plugins[pluginType]), func(i interface{}) error {
			return i.(startStopper).Start()
		}); err != nil {
			return err
		}
	}

	// start the metricWriter, so it can begin flushing metrics
	if err := a.metricWriter.Start(); err != nil {
		return err
	}

	// start all servers
	if err := CallWith(startStoppersToInterfaces(a.servers), func(i interface{}) error {
		return i.(startStopper).Start()
	}); err != nil {
		return err
	}

	return nil
}

func (a *App) Stop(duration time.Duration) error {
	a.eventMetric(gatekeeper.AppStoppedEvent)
	errs := NewMultiError()

	// stop servers
	if err := CallWith(startStoppersToInterfaces(a.servers), func(i interface{}) error {
		return i.(startStopper).Stop(duration)
	}); err != nil {
		errs.Add(err)
	}

	// stop all plugins, but the metric writer ...
	for _, pluginType := range []PluginType{UpstreamPlugin, ModifierPlugin, LoadBalancerPlugin, RouterPlugin} {
		if err := CallWith(pluginManagersToInterfaces(a.plugins[pluginType]), func(i interface{}) error {
			return i.(startStopper).Stop(duration)
		}); err != nil {
			errs.Add(err)
		}
	}

	// stop the metricWriter
	if err := a.metricWriter.Stop(duration); err != nil {
		errs.Add(err)
	}

	// stop the metricWriter plugins
	if err := CallWith(pluginManagersToInterfaces(a.plugins[MetricPlugin]), func(i interface{}) error {
		return i.(startStopper).Stop(duration)
	}); err != nil {
		errs.Add(err)
	}

	return errs.ToErr()
}

func (a *App) eventMetric(event gatekeeper.Event) {
	a.metricWriter.EventMetric(&gatekeeper.EventMetric{
		Timestamp: time.Now(),
		Event:     event,
	})
}
