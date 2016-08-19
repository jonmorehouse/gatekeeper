package core

import (
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	loadbalancer_plugin "github.com/jonmorehouse/gatekeeper/plugin/loadbalancer"
	metric_plugin "github.com/jonmorehouse/gatekeeper/plugin/metric"
	modifier_plugin "github.com/jonmorehouse/gatekeeper/plugin/modifier"
	router_plugin "github.com/jonmorehouse/gatekeeper/plugin/router"
	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

type PluginManager interface {
	Build() error
	Start() error
	Stop(time.Duration) error

	// Call a method on the plugin instance with additional robustness and
	// metrics emitted to the metricWriter. This supports retries and
	// timeouts, by default timing out a call based upon the configured
	// timeout
	Call(string, func(Plugin) error) error
	CallOnce(string, func(Plugin) error) error

	// Grab a plugin under the readLock, under normal circumstances, we
	// most likely wouldn't do anything substantial here. MetricWriter uses
	// it for type switching to see which metrics to pass along
	Grab(func(Plugin))

	// information about the underlying plugin
	Type() PluginType
	Name() string
}

func NewPluginManager(cmd string, args map[string]interface{}, pluginType PluginType, metricWriter MetricWriterClient) PluginManager {
	pluginName := filepath.Base(strings.SplitN(cmd, " ", 2)[0])

	return &pluginManager{
		metricWriter: metricWriter,

		pluginType: pluginType,
		pluginName: pluginName,
		pluginCmd:  cmd,
		pluginArgs: args,

		callTimeout:       time.Second * 100,
		callRetries:       3,
		heartbeatInterval: time.Millisecond * 500,

		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),

		HookManager: NewHookManager(),
	}
}

type pluginManager struct {
	metricWriter MetricWriterClient

	pluginType PluginType
	pluginName string
	pluginCmd  string
	pluginArgs map[string]interface{}

	instance Plugin

	workers           uint
	callTimeout       time.Duration
	callRetries       uint
	heartbeatInterval time.Duration

	// internal worker attributes
	stopCh chan struct{}
	doneCh chan struct{}

	RWMutex

	SyncStartStopper
	HookManager
}

func (p *pluginManager) Build() error {
	if err := p.buildInstance(); err != nil {
		return err
	}
	return nil
}

func (p *pluginManager) Start() error {
	return p.SyncStart(func() error {
		if err := p.startInstance(); err != nil {
			return err
		}
		p.HookManager.AddHook(p.heartbeatInterval, p.heartbeat)
		return p.HookManager.Start()
	})
}

func (p *pluginManager) Stop(dur time.Duration) error {
	return p.SyncStop(func() error {
		errs := NewMultiError()
		if err := p.HookManager.Stop(dur); err != nil {
			errs.Add(err)
		}

		// stop and kill the plugin and its connection
		if err := p.CallOnce("Stop", func(plugin Plugin) error {
			if plugin != nil {
				return plugin.Stop()
			}
			return nil
		}); err != nil {
			errs.Add(err)
		}

		if err := p.CallOnce("Kill", func(plugin Plugin) error {
			if plugin != nil {
				plugin.Kill()
			}
			return nil
		}); err != nil {
			errs.Add(err)
		}

		return errs.ToErr()

	})
}

func (p *pluginManager) Grab(cb func(Plugin)) {
	p.RLock()
	defer p.RUnlock()

	cb(p.instance)
}

// call will attempt to execute a method, using a timeout and will write
// metrics around it! It will retry up to N times and waits a configurable
// amount of time. It doesn't make any guarantees about error returns and
// instead relies upon the callback returning Nil or a real error before
// proceeding forward.
func (p *pluginManager) Call(method string, cb func(Plugin) error) error {
	log.Println("Plugin.Call method:", method, " type: ", p.pluginType.String())
	calls := 0
	defer func() {
		if calls > 1 {
			p.eventMetric(gatekeeper.PluginRetryEvent)
		}
	}()

	return Retry(p.callRetries, func() error {
		err, ok := CallWithTimeout(p.callTimeout, func() error {
			p.RLock()
			defer p.RUnlock()

			calls += 1

			startTS := time.Now()
			err := cb(p.instance)
			p.pluginMetric(method, time.Now().Sub(startTS), err)
			return err
		})

		if !ok {
			return PluginTimeoutErr
		}
		return err
	})
}

func (p *pluginManager) CallOnce(method string, cb func(Plugin) error) error {
	log.Println("Plugin.CallOnce method:", method, " type: ", p.pluginType.String())
	err, ok := CallWithTimeout(p.callTimeout, func() error {
		p.RLock()
		defer p.RUnlock()

		startTS := time.Now()
		err := cb(p.instance)
		p.pluginMetric(method, time.Now().Sub(startTS), err)
		return err
	})

	if !ok {
		return PluginTimeoutErr
	}

	return err
}

func (p *pluginManager) Type() PluginType {
	return p.pluginType
}

func (p *pluginManager) Name() string {
	return p.pluginName
}

// build builds a plugin instance, configuring and starting it
func (p *pluginManager) buildInstance() error {
	// fetch the plugin instance
	instance, err := func() (Plugin, error) {
		switch p.pluginType {
		case LoadBalancerPlugin:
			return loadbalancer_plugin.NewClient(p.pluginName, p.pluginCmd)
		case ModifierPlugin:
			return modifier_plugin.NewClient(p.pluginName, p.pluginCmd)
		case MetricPlugin:
			return metric_plugin.NewClient(p.pluginName, p.pluginCmd)
		case UpstreamPlugin:
			return upstream_plugin.NewClient(p.pluginName, p.pluginCmd)
		case RouterPlugin:
			return router_plugin.NewClient(p.pluginName, p.pluginCmd)
		}
		return nil, InternalPluginError
	}()

	if err != nil {
		return err
	}

	p.Lock()
	defer p.Unlock()
	p.instance = instance
	return nil
}

func (p *pluginManager) startInstance() error {
	err := p.CallOnce("Configure", func(plugin Plugin) error {
		return plugin.Configure(p.pluginArgs)
	})
	if err != nil {
		return err
	}

	return p.CallOnce("Start", func(plugin Plugin) error {
		return plugin.Start()
	})
}

// heartbeat is responsible for calling the Heartbeat method on the plugin with
// a timeout and retry. If it continually fails, then we stop and rebuild the plugin.
func (p *pluginManager) heartbeat() error {
	// Try to call the heartbeat method three times. Each call to the
	// plugin will attempt up to p.retries times, respecting the call
	// timeout configured in this plugin.
	err := Retry(3, func() error {
		return p.Call("Heartbeat", func(plugin Plugin) error {
			return plugin.Heartbeat()
		})
	})

	if err == nil {
		return nil
	}

	p.eventMetric(gatekeeper.PluginRestartedEvent)
	p.buildInstance()
	p.startInstance()
	return err
}

func (p *pluginManager) eventMetric(event gatekeeper.Event) {
	p.metricWriter.EventMetric(&gatekeeper.EventMetric{
		Timestamp: time.Now(),
		Event:     event,
		Extra: map[string]string{
			"plugin-name": p.pluginName,
			"plugin-type": p.pluginType.String(),
			"plugin-cmd":  p.pluginCmd,
		},
	})
}

func (p *pluginManager) pluginMetric(method string, latency time.Duration, err error) {
	p.metricWriter.PluginMetric(&gatekeeper.PluginMetric{
		Timestamp: time.Now(),
		Latency:   latency,

		PluginType: p.pluginType.String(),
		PluginName: p.pluginName,
		MethodName: method,

		Err: err,
	})
}
