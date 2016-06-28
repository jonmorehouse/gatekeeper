package gatekeeper

import (
	"sync"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
)

type MetricWriter interface {
	Start() error
	Stop(time.Duration) error

	EventMetric(*shared.EventMetric)
	ProfilingMetric(*shared.ProfilingMetric)
	PluginMetric(*shared.PluginMetric)
	RequestMetric(*shared.RequestMetric)
	UpstreamMetric(*shared.UpstreamMetric)
}

// MetricWriterClient is an interface which is passed around simply to write
// metrics too. Most writers of metrics will be from metrics client, where as
// some sort of higher level "manager-like" type will be responsible for the
// full lifecycle of the MetricWriter
type MetricWriterClient interface {
	EventMetric(*shared.EventMetric)
	ProfilingMetric(*shared.ProfilingMetric)
	PluginMetric(*shared.PluginMetric)
	RequestMetric(*shared.RequestMetric)
	UpstreamMetric(*shared.UpstreamMetric)
}

type MetricPluginClient interface {
	WriteEventMetrics([]*shared.EventMetric) []error
	WriteProfilingMetrics([]*shared.ProfilingMetric) []error
	WritePluginMetrics([]*shared.PluginMetric) []error
	WriteRequestMetrics([]*shared.RequestMetric) []error
	WriteUpstreamMetrics([]*shared.UpstreamMetric) []error
}

type metricTuple struct {
	name   string
	metric interface{}
}

type metricWriter struct {
	pluginManagers []PluginManager // plugins that this metrics writer emits too
	flushInterval  time.Duration   // maximum flush interval
	maxBuffered    uint
	buffered       uint
	buffer         map[string][]interface{}

	metricCh chan metricTuple
	stopCh   chan struct{}
	skipCh   chan struct{}
	doneCh   chan struct{}
}

func NewMetricWriter(pluginManagers []PluginManager) MetricWriter {
	return &metricWriter{
		pluginManagers: pluginManagers,
		flushInterval:  time.Millisecond * 500,
		maxBuffered:    1000,
		buffered:       0,
		buffer: map[string][]interface{}{
			"event-metrics":     make([]interface{}, 0, 0),
			"profiling-metrics": make([]interface{}, 0, 0),
			"plugin-metrics":    make([]interface{}, 0, 0),
			"request-metrics":   make([]interface{}, 0, 0),
			"upstream-metrics":  make([]interface{}, 0, 0),
		},

		metricCh: make(chan metricTuple, 1000),
		stopCh:   make(chan struct{}),
		skipCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func (m *metricWriter) Start() error {
	err := EachPluginManager(m.pluginManagers, func(pluginManager PluginManager) error {
		return pluginManager.Start()
	})
	if err != nil {
		return err
	}

	go m.worker()
	return nil
}

func (m *metricWriter) Stop(duration time.Duration) error {
	m.stopCh <- struct{}{}
	select {
	case <-m.doneCh:
		return nil
	case <-time.After(duration):
		return InternalTimeoutError
	}

	return EachPluginManager(m.pluginManagers, func(pluginManager PluginManager) error {
		return pluginManager.Stop(duration)
	})
}

func (m *metricWriter) EventMetric(metric *shared.EventMetric) {
	m.sendMetric(metricTuple{
		metric: metric,
		name:   "event-metrics",
	})
}

func (m *metricWriter) ProfilingMetric(metric *shared.ProfilingMetric) {
	m.sendMetric(metricTuple{
		metric: metric,
		name:   "profiling-metrics",
	})
}

func (m *metricWriter) PluginMetric(metric *shared.PluginMetric) {
	m.sendMetric(metricTuple{
		metric: metric,
		name:   "plugin-metrics",
	})
}

func (m *metricWriter) RequestMetric(metric *shared.RequestMetric) {
	m.sendMetric(metricTuple{
		metric: metric,
		name:   "request-metrics",
	})
}

func (m *metricWriter) UpstreamMetric(metric *shared.UpstreamMetric) {
	m.sendMetric(metricTuple{
		metric: metric,
		name:   "upstream-metrics",
	})
}

// emit a metric to our internal worker to be buffered and flushed later on
func (m *metricWriter) sendMetric(tuple metricTuple) {
	m.metricCh <- tuple
}

func (m *metricWriter) worker() {
	// NOTE: we can't use a time.Ticker type here because we'd like to
	// "reset" the timer periodically
	flushCh := make(chan struct{})
	flushTimer := time.AfterFunc(m.flushInterval, func() {
		flushCh <- struct{}{}
	})

	for {
		select {
		case <-m.stopCh:
			goto stop
		case metric := <-m.metricCh:
			m.bufferMetric(metric.name, metric.metric)
			if m.buffered > m.maxBuffered {
				m.flush()
				flushTimer.Reset(m.flushInterval)
			}
		case <-flushCh:
			m.flush()
			flushTimer.Reset(m.flushInterval)
		}
	}

stop:
	flushTimer.Stop()
	m.flush()
	m.doneCh <- struct{}{}
}

func (m *metricWriter) bufferMetric(kind string, metric interface{}) {
	if _, ok := m.buffer[kind]; !ok {
		shared.ProgrammingError("unable to buffer metric")
		return
	}

	m.buffer[kind] = append(m.buffer[kind], metric)
	m.buffered += 1
}

func (m *metricWriter) flush() {
	// flush all metrics to the metrics plugin
	go func(buffer map[string][]interface{}) {
		var wg sync.WaitGroup

		methods := map[string]func([]interface{}){
			"event-metrics":     m.flushEventMetrics,
			"profiling-metrics": m.flushProfilingMetrics,
			"plugin-metrics":    m.flushPluginMetrics,
			"request-metrics":   m.flushRequestMetrics,
			"upstream-metrics":  m.flushUpstreamMetrics,
		}

		for kind, bufferedMetrics := range buffer {
			method, ok := methods[kind]
			if !ok {
				shared.ProgrammingError("invalid metrics type in metric-writer")
			}
			wg.Add(1)

			go func(method func([]interface{}), metrics []interface{}) {
				method(metrics)
				wg.Done()
			}(method, bufferedMetrics)
		}
	}(m.buffer)

	// allocate a new buffer after we pass a pointer to the old buffer to the go routine
	m.buffer = map[string][]interface{}{
		"event-metrics":     make([]interface{}, 0, m.maxBuffered),
		"profiling-metrics": make([]interface{}, 0, m.maxBuffered),
		"plugin-metrics":    make([]interface{}, 0, m.maxBuffered),
		"request-metrics":   make([]interface{}, 0, m.maxBuffered),
		"upstream-metrics":  make([]interface{}, 0, m.maxBuffered),
	}
	m.buffered = 0
	m.EventMetric(&shared.EventMetric{
		Timestamp: time.Now(),
		Event:     shared.MetricsFlushedEvent,
	})
}

func (m *metricWriter) localEventMetric(event shared.MetricEvent) {
	m.EventMetric(&shared.EventMetric{
		Timestamp: time.Now(),
		Event:     event,
	})
}

// for each pluginManager, call the closure passed in with a fully operable
// MetricPluginClient, emitting metrics for each  call.
func (m *metricWriter) eachPlugin(methodName string, cb func(MetricPluginClient) error) error {
	var wg sync.WaitGroup
	errs := NewMultiError()

	for _, pluginManager := range m.pluginManagers {
		wg.Add(1)

		go func(pluginManager PluginManager) {
			defer wg.Done()
			plugin, err := pluginManager.Get()
			if err != nil {
				shared.ProgrammingError("PluginManager has not instances available")
				return
			}

			metricPlugin, ok := plugin.(MetricPluginClient)
			if !ok {
				shared.ProgrammingError("PluginManager returned an instance that was not a MetricsClient")
				return
			}

			// call the method, emitting a metric with its latency
			startTS := time.Now()
			err = cb(metricPlugin)
			pluginManager.WriteMetric(methodName, time.Now().Sub(startTS), err)
			if err != nil {
				errs.Add(err)
			}
		}(pluginManager)
	}

	wg.Wait()

	return errs.ToErr()
}

func (m *metricWriter) flushEventMetrics(buffer []interface{}) {
	metrics := make([]*shared.EventMetric, len(buffer))

	for idx, metric := range buffer {
		eventMetric, ok := metric.(*shared.EventMetric)
		if !ok {
			shared.ProgrammingError("unable to cast metric to *shared.EventMetric")
			return
		}
		metrics[idx] = eventMetric
	}

	err := m.eachPlugin("WriteEventMetrics", func(plugin MetricPluginClient) error {
		errs := plugin.WriteEventMetrics(metrics)
		return (&MultiError{errors: errs}).ToErr()
	})

	if err != nil {
		m.localEventMetric(shared.MetricsFlushedSuccessEvent)
	} else {
		m.localEventMetric(shared.MetricsFlushedErrorEvent)
	}
}

func (m *metricWriter) flushProfilingMetrics(buffer []interface{}) {
	metrics := make([]*shared.ProfilingMetric, len(buffer))

	for idx, metric := range buffer {
		eventMetric, ok := metric.(*shared.ProfilingMetric)
		if !ok {
			shared.ProgrammingError("unable to cast metric to *shared.ProfilingMetric")
			return
		}
		metrics[idx] = eventMetric
	}

	err := m.eachPlugin("WriteProfilingMetrics", func(plugin MetricPluginClient) error {
		errs := plugin.WriteProfilingMetrics(metrics)
		return (&MultiError{errors: errs}).ToErr()
	})

	if err != nil {
		m.localEventMetric(shared.MetricsFlushedSuccessEvent)
	} else {
		m.localEventMetric(shared.MetricsFlushedErrorEvent)
	}
}

func (m *metricWriter) flushPluginMetrics(buffer []interface{}) {
	metrics := make([]*shared.PluginMetric, len(buffer))

	for idx, metric := range buffer {
		eventMetric, ok := metric.(*shared.PluginMetric)
		if !ok {
			shared.ProgrammingError("unable to cast metric to *shared.PluginMetric")
			return
		}
		metrics[idx] = eventMetric
	}

	err := m.eachPlugin("WritePluginMetrics", func(plugin MetricPluginClient) error {
		errs := plugin.WritePluginMetrics(metrics)
		return (&MultiError{errors: errs}).ToErr()

	})

	if err != nil {
		m.localEventMetric(shared.MetricsFlushedSuccessEvent)
	} else {
		m.localEventMetric(shared.MetricsFlushedErrorEvent)
	}
}

func (m *metricWriter) flushRequestMetrics(buffer []interface{}) {
	metrics := make([]*shared.RequestMetric, len(buffer))

	for idx, metric := range buffer {
		eventMetric, ok := metric.(*shared.RequestMetric)
		if !ok {
			shared.ProgrammingError("unable to cast metric to *shared.RequestMetric")
			return
		}
		metrics[idx] = eventMetric
	}

	err := m.eachPlugin("WriteRequestMetrics", func(plugin MetricPluginClient) error {
		errs := plugin.WriteRequestMetrics(metrics)
		return (&MultiError{errors: errs}).ToErr()
	})

	if err != nil {
		m.localEventMetric(shared.MetricsFlushedSuccessEvent)
	} else {
		m.localEventMetric(shared.MetricsFlushedErrorEvent)
	}
}

func (m *metricWriter) flushUpstreamMetrics(buffer []interface{}) {
	metrics := make([]*shared.UpstreamMetric, len(buffer))

	for idx, metric := range buffer {
		eventMetric, ok := metric.(*shared.UpstreamMetric)
		if !ok {
			shared.ProgrammingError("unable to cast metric to *shared.UpstreamMetric")
			return
		}
		metrics[idx] = eventMetric
	}

	err := m.eachPlugin("WriteUpstreamMetrics", func(plugin MetricPluginClient) error {
		errs := plugin.WriteUpstreamMetrics(metrics)
		return (&MultiError{errors: errs}).ToErr()
	})

	if err != nil {
		m.localEventMetric(shared.MetricsFlushedSuccessEvent)
	} else {
		m.localEventMetric(shared.MetricsFlushedErrorEvent)
	}
}
