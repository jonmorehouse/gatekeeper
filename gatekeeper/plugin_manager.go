package gatekeeper

import (
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type PluginManager interface {
	Start() error
	Stop(time.Duration) error

	// Call a method on the plugin instance with additional robustness and
	// metrics emitted to the metricWriter. This supports retries and
	// timeouts, by default timing out a call based upon the configured
	// timeout
	Call(string, func(Plugin) error) error

	// Grab a plugin under the readLock, under normal circumstances, we
	// most likely wouldn't do anything substantial here. MetricWriter uses
	// it for type switching to see which metrics to pass along
	Grab(func(Plugin)) Plugin
}

func NewPluginManager(cmd string, args map[string]interface{}, pluginType PluginType, broadcaster EventBroadcaster, metricWriter MetricWriterClient) PluginManager {
	pluginName := filepath.Base(strings.SplitN(cmd, " ", 2)[0])

	return &pluginManager{
		metricWriter: metricWriter,

		pluginType: pluginType,
		pluginName: pluginName,
		pluginCmd:  pluginCmd,
		pluginArgs: pluginArgs,

		instance: nil,
		workers:  0,

		callTimeout: time.Millisecond * 5,
		callRetries: 3,

		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

type pluginManager struct {
	metricWriter MetricWriterClient
	broadcaster  EventBroadcaster

	pluginType PluginType
	pluginName string
	pluginCmd  string
	pluginArgs map[string]interface{}

	instance Plugin
	workers  uint

	callTimeout       time.Duration
	callRetries       uint
	heartbeatInterval time.Duration

	// internal worker attributes
	stopCh chan struct{}
	doneCh chan struct{}
}

func (p *pluginManager) Start() error {
	if err := p.buildInstance(); err != nil {
		p.Stop()
		return err
	}

	// start the heartbeat worker
	p.Lock()
	p.workers += 1
	p.Unlock()
	go p.worker()
	return nil
}

func (p *pluginManager) Stop(time.Duration) error {
	errs := NewMultiError()
	var wg sync.WaitGroup

	p.RLock()
	workers := p.workers
	p.RUnlock()

	// for each worker, emit a stop message and wait for the corresponding
	// done message to be passed back
	_, ok := CallWithTimeout(dur, func() error {
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			p.stopCh <- struct{}{}
		}

		go func() {
			for _, _ := range p.doneCh {
				wg.Done()
			}
		}()

		wg.Wait()
		close(p.doneCh)
		return nil
	})
	if !ok {
		errs.Add(InternalPluginError)
	}

	// stop and kill the plugin and its connection
	err := p.Call("Stop", func(plugin Plugin) error {
		return plugin.Stop()
	})
	if err != nil {
		errs.Add(err)
	}

	err = p.Call("Kill", func(plugin Plugin) error {
		plugin.Kill()
		return nil
	})
	if err != nil {
		errs.Add(err)
	}

	// clean up internal state
	p.Lock()
	p.workers = 0
	p.instance = nil
	p.Unlock()

	return err
}

func (p *pluginManager) Grab(cb func(Plugin)) {
	p.RLock()
	defer p.RULock()

	cb(p.instance)
}

// call will attempt to execute a method, using a timeout and will write
// metrics around it! It will retry up to N times and waits a configurable
// amount of time. It doesn't make any guarantees about error returns and
// instead relies upon the callback returning Nil or a real error before
// proceeding forward.
func (p *pluginManager) Call(method string, cb func(Plugin) error) error {
	calls := 0
	defer func() {
		if calls > 1 {
			p.eventMetric(shared.PluginRetryEvent)
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
			return PluginTimeoutError
		}
		return err
	})
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
			return router_plugin.NewClient(p.pluginName, p.pluinCmd)
		}
		return nil, InternalPluginError
	}

	if err != nil {
		return err
	}

	startTS := time.Now()
	err = instance.Configure(p.args)
	p.pluginMetric("Configure", time.Now().Sub(startTS), err)
	if err != nil {
		return err
	}

	startTS = time.Now()
	err = instance.Start()
	p.pluginMetric("Start", time.Now().Sub(startTS), err)
	if err != nil {
		return err
	}

	p.Lock()
	defer p.Unlock()
	p.instance = instance
	return nil
}

// worker is responsible for sending heartbeats off to the plugin after a configurable amount of time
func (p *pluginManager) worker() {
	ticker := time.NewTicker(p.heartbeatInterval)

	for {
		select {
		case <-ticker.C:
			p.heartbeat()
		case <-p.stopCh:
			p.doneCh <- struct{}{}
			return
		}
	}
}

// heartbeat is responsible for calling the Heartbeat method on the plugin with
// a timeout and retry. If it continually fails, then we stop and rebuild the plugin.
func (p *pluginManager) heartbeat() {
	// Try to call the heartbeat method three times. Each call to the
	// plugin will attempt up to p.retries times, respecting the call
	// timeout configured in this plugin.
	err := Retry(3, func() error {
		return p.Call("Heartbeat", func(plugin Plugin) {
			return plugin.Heartbeat()
		})
	})

	if err == nil {
		return
	}

	p.eventMetric(shared.PluginRestartedEvent)
	p.buildInstance()
}

func (p *pluginManager) eventMetric(event shared.MetricEvent) {
	p.metricWriter.EventMetric(&shared.EventMetric{
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
	p.metricWriter.PluginMetric(&shared.PluginMetric{
		Timestamp: time.Now(),
		Latency:   latency,

		PluginType: p.pluginType.String(),
		PluginName: p.pluginName,
		MethodName: method,

		Err: err,
	})
}
