package gatekeeper

import (
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	loadbalancer_plugin "github.com/jonmorehouse/gatekeeper/plugin/loadbalancer"
	metric_plugin "github.com/jonmorehouse/gatekeeper/plugin/metric"
	modifier_plugin "github.com/jonmorehouse/gatekeeper/plugin/modifier"
	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
	"github.com/jonmorehouse/gatekeeper/shared"
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

	// emit a PluginMetric to the the metricWriter
	WriteMetric(string, time.Duration, error)
}

type pluginManager struct {
	metricWriter MetricWriterClient
	pluginType   PluginType
	pluginName   string
	pluginCmd    string
	opts         map[string]interface{}

	// count is the number of instances that this manager should manage for this plugin
	count     uint
	instances []Plugin
}

func NewPluginManager(pluginCmd string, opts map[string]interface{}, count uint, pluginType PluginType, metricWriter MetricWriterClient) PluginManager {
	return &pluginManager{
		metricWriter: metricWriter,
		pluginType:   pluginType,
		pluginCmd:    pluginCmd,
		pluginName:   filepath.Base(pluginCmd),
		opts:         opts,
		count:        count,
		instances:    make([]Plugin, 0, count),
	}
}

func (p *pluginManager) Start() error {
	// starts and configures the plugins so that they work correctly
	errs := NewMultiError()
	var wg sync.WaitGroup

	for i := uint(0); i < p.count; i++ {
		wg.Add(1)
		instance, err := p.buildPlugin()
		if err != nil {
			errs.Add(err)

			// most of the time, when an error occurs, it is
			// because we specified an invalid plugin of this type.
			// However, if there is a problem starting a plugin,
			// then we need kill off the plugin before exiting.
			if instance != nil {
				p.eventMetric(shared.PluginFailedEvent)
				instance.Kill()
			}
			wg.Done()
			continue
		}
		p.instances = append(p.instances, instance)

		go func(plugin Plugin) {
			defer wg.Done()

			// configure plugin
			startTS := time.Now()
			err := plugin.Configure(p.opts)
			p.WriteMetric("configure", time.Now().Sub(startTS), err)
			if err != nil {
				p.eventMetric(shared.PluginFailedEvent)
				errs.Add(err)
				plugin.Kill()
				return
			}

			// start plugin
			startTS = time.Now()
			err = plugin.Start()
			p.WriteMetric("start", time.Now().Sub(startTS), err)
			if err != nil {
				p.eventMetric(shared.PluginFailedEvent)
				errs.Add(err)
				plugin.Kill()
				return
			}

			p.eventMetric(shared.PluginStartedEvent)
		}(instance)
	}

	wg.Wait()
	return errs.ToErr()
}

func (p *pluginManager) Stop(duration time.Duration) error {
	errs := NewMultiError()

	// stop each plugin in a goroutine
	var wg sync.WaitGroup
	for _, instance := range p.instances {
		wg.Add(1)
		go func(instance Plugin) {
			// stop the instance
			startTS := time.Now()
			err := instance.Stop()
			p.WriteMetric("stop", time.Now().Sub(startTS), err)
			if err != nil {
				p.eventMetric(shared.PluginFailedEvent)
				errs.Add(err)
			}
			p.eventMetric(shared.PluginStoppedEvent)

			// kill the plugin, regardless of whether it stopped successfully or not
			startTS = time.Now()
			instance.Kill()
			p.WriteMetric("kill", time.Now().Sub(startTS), nil)

			wg.Done()
		}(instance)
	}

	// wait for the plugins to all finish, or otherwise timeout
	doneCh := make(chan struct{})
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
			errs.Add(InternalTimeoutError)
			goto cleanup
		}
	}

cleanup:
	return errs.ToErr()
}

func (p *pluginManager) Get() (Plugin, error) {
	if len(p.instances) == 0 {
		return nil, InternalPluginError
	}

	idx := rand.Intn(len(p.instances))
	return p.instances[idx], nil
}

func (p *pluginManager) All() ([]Plugin, error) {
	if len(p.instances) == 0 {
		return []Plugin{}, InternalPluginError
	}
	return p.instances, nil
}

func (p *pluginManager) WriteMetric(method string, latency time.Duration, err error) {
	pluginResponse := shared.PluginResponseOk
	if err != nil {
		pluginResponse = shared.PluginResponseNotOk
	}

	metric := &shared.PluginMetric{
		Timestamp:    time.Now(),
		PluginType:   p.pluginType.String(),
		PluginName:   p.pluginName,
		MethodName:   method,
		Latency:      latency,
		ResponseType: pluginResponse,
		Error:        err,
	}
	p.metricWriter.PluginMetric(metric)
}

func (m pluginManager) buildPlugin() (Plugin, error) {
	if m.pluginType == UpstreamPlugin {
		return upstream_plugin.NewClient(m.pluginName, m.pluginCmd)
	}
	if m.pluginType == LoadBalancerPlugin {
		return loadbalancer_plugin.NewClient(m.pluginName, m.pluginCmd)
	}
	if m.pluginType == ModifierPlugin {
		return modifier_plugin.NewClient(m.pluginName, m.pluginCmd)
	}
	if m.pluginType == MetricPlugin {
		return metric_plugin.NewClient(m.pluginName, m.pluginCmd)
	}

	shared.ProgrammingError("Invalid plugin type")
	return nil, InternalPluginError
}

// emits a new event to the metricsWriter, adding additional information into
// the `Extra` field denoting the plugin name and path.
func (p *pluginManager) eventMetric(event shared.MetricEvent) {
	metric := &shared.EventMetric{
		Event:     event,
		Timestamp: time.Now(),
		Extra: map[string]string{
			"plugin-type": p.pluginType.String(),
			"plugin-name": p.pluginName,
			"plugin-cmd":  p.pluginCmd,
		},
	}
	p.metricWriter.EventMetric(metric)
}
