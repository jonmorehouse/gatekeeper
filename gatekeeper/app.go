package gatekeeper

import (
	"sync"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
)

type startStop interface {
	Start() error
	Stop(time.Duration) error
}

type App struct {
	// the server type adheres to the startStop interface, by convenience.
	servers []Server

	metricWriter      MetricWriter
	broadcaster       EventBroadcaster
	upstreamPublisher *UpstreamPublisher
	upstreamMatcher   UpstreamMatcher
	modifier          Modifier
	loadBalancer      LoadBalancer
}

func New(options Options) (*App, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	// MetricWriter is a wrapper around a series of Event plugins which is
	// used by various processes around teh application to emit metrics and
	// errors to the Event plugin
	metricPlugins := make([]PluginManager, len(options.MetricPlugins), len(options.MetricPlugins))
	metricWriter := NewMetricWriter(metricPlugins)
	for idx, pluginCmd := range options.MetricPlugins {
		plugin := NewPluginManager(pluginCmd, options.MetricPluginOpts, options.MetricPluginsCount, MetricPlugin, metricWriter)
		metricPlugins[idx] = plugin
	}

	metricWriter.EventMetric(&shared.EventMetric{
		Timestamp: time.Now(),
		Event:     shared.AppStartedEvent,
	})

	// the broadcaster is what glues everything together. It is responsible
	// for dispensing events throughout the server so that plugins can
	// update themselves in accordance with systems going online and
	// offline.
	broadcaster := NewUpstreamEventBroadcaster()

	// each UpstreamPlugin is special because it is responsible for calling
	// asynchronously back into the parent process. Specifically it
	// requires an UpstreamPublisher which is cast as an
	// upstream_plugin.Manager to be accessible for calling back into the
	// parent program.
	upstreamPlugins := make([]PluginManager, 0, len(options.UpstreamPlugins))
	for _, pluginCmd := range options.UpstreamPlugins {
		plugin := NewPluginManager(pluginCmd, options.UpstreamPluginOpts, options.UpstreamPluginsCount, UpstreamPlugin, metricWriter)
		upstreamPlugins = append(upstreamPlugins, plugin)
	}

	// the upstreamPublisher needs to know about each pluginManager and in
	// return, each upstreamPlugin needs to use the UpstreamPublisher
	// because it implements the Manager interface and is what the
	// RPCServer that is launched inside of each RPCClient uses to emit
	// messages too
	upstreamPublisher := NewUpstreamPublisher(upstreamPlugins, broadcaster, metricWriter)
	// when the upstream plugins are configured, the publisher gets passed
	// to them and used as the manager type. This allows the upstreamPlugin
	// to talk back into this parent process.
	options.UpstreamPluginOpts["manager"] = upstreamPublisher

	// build an upstreamMatcher for each server to communicate to the
	// upstream store. This is used to find the correct upstream for each
	// request.
	upstreamMatcher := NewUpstreamMatcher(broadcaster)

	// only one loadbalancer plugin is permitted, this is to ensure that we
	// actually have sane load balancing! Otherwise, we run the risk of
	// having multiple different load balancing algorithms at once.
	loadBalancerPlugin := NewPluginManager(options.LoadBalancerPlugin, options.LoadBalancerPluginOpts, options.LoadBalancerPluginsCount, LoadBalancerPlugin, metricWriter)
	loadBalancer := NewLoadBalancer(broadcaster, loadBalancerPlugin)

	// for each specified Modifier plugin, we create a PluginManager which
	// manages the lifecycle of the plugin.
	modifierPlugins := make([]PluginManager, 0, len(options.ModifierPlugins))
	for _, pluginCmd := range options.ModifierPlugins {
		plugin := NewPluginManager(pluginCmd, options.ModifierPluginOpts, options.ModifierPluginsCount, ModifierPlugin, metricWriter)
		modifierPlugins = append(modifierPlugins, plugin)
	}

	// modifier is a type that wraps a series of modifier plugins and is
	// used by the Server and Proxier to actually modify requests and
	// responses
	modifier := NewModifier(modifierPlugins, metricWriter)

	// Proxier is the naive type which _actually_ handles proxying of
	// requests out to the backend address.
	proxier := NewProxier(modifier, metricWriter)

	// build out each server type
	servers := make([]Server, 0, 4)
	if options.HTTPPublicPort != 0 {
		servers = append(servers, &ProxyServer{
			port:            options.HTTPPublicPort,
			protocol:        shared.HTTPPublic,
			proxier:         proxier,
			upstreamMatcher: upstreamMatcher,
			loadBalancer:    loadBalancer,
			modifier:        modifier,
			metricWriter:    metricWriter,
		})
	}

	if len(servers) == 0 {
		return nil, ConfigurationError
	}

	return &App{
		metricWriter:      metricWriter,
		broadcaster:       broadcaster,
		upstreamMatcher:   upstreamMatcher,
		upstreamPublisher: upstreamPublisher,
		loadBalancer:      loadBalancer,
		modifier:          modifier,
		servers:           servers,
	}, nil
}

func (a *App) Start() error {
	// start the upstreamRequester and loadBalancer first because they
	// receive notifications from the broadcaster immediately and we'd like
	// to make sure that any plugin that emits upstreams/backends to the
	// server at any time is supported. eg: if a plugin emits
	// upstreams/backends at start time and never again.
	syncStart := []startStop{
		a.metricWriter,
		a.upstreamMatcher,
		a.loadBalancer,
		a.upstreamPublisher,
		a.modifier,
	}
	for _, job := range syncStart {
		if job == nil {
			return ConfigurationError
		}

		if err := job.Start(); err != nil {
			return err
		}
	}

	// start all servers asynchronously
	var wg sync.WaitGroup
	errs := NewMultiError()
	for _, server := range a.servers {
		wg.Add(1)
		go func(s startStop) {
			defer wg.Done()
			if err := s.Start(); err != nil {
				errs.Add(err)
			}
		}(server)
	}

	wg.Wait()
	return errs.ToErr()
}

func (a *App) Stop(duration time.Duration) error {
	errs := NewMultiError()
	var wg sync.WaitGroup

	// stop accepting connections on each server first, and then start the
	// shutdown process. Its expected that the shutdown process takes
	// longer and as such, it is fired off in a goroutine at the same time
	// that other services throughout the app are shutdown.
	for _, server := range a.servers {
		if err := server.StopAccepting(); err != nil {
			errs.Add(err)
			continue
		}
		wg.Add(1)
		go func(s startStop) {
			defer wg.Done()
			if err := s.Stop(duration); err != nil {
				errs.Add(err)
			}
		}(server)
	}

	// shutdown all other plugins and internal subscribers
	jobs := []startStop{
		a.upstreamMatcher,
		a.loadBalancer,
		a.upstreamPublisher,
		a.modifier,
	}
	for _, job := range jobs {
		wg.Add(1)
		go func(j startStop) {
			defer wg.Done()
			if err := j.Stop(duration); err != nil {
				errs.Add(err)
			}
		}(job)
	}
	wg.Wait()

	// emit a metric denoting that the app is stopping
	a.metricWriter.EventMetric(&shared.EventMetric{
		Timestamp: time.Now(),
		Event:     shared.AppStoppedEvent,
	})
	// NOTE: we stop the `metricWriter` plugin last, so as to allow for
	// other plugins to emit metrics to it in their stop methods
	if err := a.metricWriter.Stop(duration); err != nil {
		errs.Add(err)
	}
	return errs.ToErr()
}
