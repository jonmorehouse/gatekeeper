package core

import (
	"sync"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
)

type MetricWriter interface {
	StartStopper

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

// The MetricsWriter doesn't mind which metrics a plugin is interested in. For
// each unique plugin type, we attempt to flush the buffered metrics to any
// plugins that desire them. This makes it easier to pass metrics between new
// plugin types dynamically.
type eventMetricsReceiver interface {
	WriteEventMetrics([]*shared.EventMetric) []error
}

type profilingMetricsReceiver interface {
	WriteProfilingMetrics([]*shared.ProfilingMetric) []error
}

type pluginMetricsReceiver interface {
	WritePluginMetrics([]*shared.PluginMetric) []error
}

type requestMetricsReceiver interface {
	WriteRequestMetrics([]*shared.RequestMetric) []error
}

type upstreamMetricsReceiver interface {
	WriteUpstreamMetrics([]*shared.UpstreamMetric) []error
}

func NewBufferedMetricsWriter(bufferSize int, flushInterval time.Duration, pluginManagers []PluginManager) MetricWriter {
	return &metricWriter{
		pluginManagers: pluginManagers,

		bufferSize:    bufferSize,
		flushInterval: flushInterval,
		buffer:        make([]shared.Metric, 0, bufferSize),

		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		bufferCh: make(chan shared.Metric, 10000),
	}
}

type metricWriter struct {
	pluginManagers []PluginManager

	buffer        []shared.Metric
	bufferSize    int
	flushInterval time.Duration

	stopCh   chan struct{}
	doneCh   chan struct{}
	bufferCh chan shared.Metric
}

func (m *metricWriter) EventMetric(event *shared.EventMetric) {
	m.bufferCh <- event
}

func (m *metricWriter) ProfilingMetric(event *shared.ProfilingMetric) {
	m.bufferCh <- event
}

func (m *metricWriter) PluginMetric(event *shared.PluginMetric) {
	m.bufferCh <- event
}

func (m *metricWriter) RequestMetric(event *shared.RequestMetric) {
	m.bufferCh <- event
}

func (m *metricWriter) UpstreamMetric(*shared.UpstreamMetric) {
	m.bufferCh <- event

}

func (m *metricWriter) worker() {
	timer := time.NewTimer(m.flushInterval)

	flush := func() {
		go m.flush(m.buffer)
		m.buffer = make([]shared.Metric, 0, m.bufferSize)
		timer.Reset()
	}

	defer func() {
		timer.Stop()
		m.flush(m.buffer)
		m.buffer = []shared.Metric(nil)
		m.doneCh <- struct{}{}
	}()

	for {
		select {
		case metric := <-m.metricCh:
			m.buffer = append(m.buffer, metric)
			if len(m.buffer) == m.bufferSize {
				flush()
			}
		case <-timer.C:
			flush()
		case <-stopCh:
			return
		}
	}
}

// flush metrics to their correct methods on the correct plugins. This is a big
// method, but its rather simple at a high level. We start by type-asserting
// each buffered metric. This gives us a slice of each metric by type.	for
// each of our plugins, we then go through them and see which metric interfaces
// they correspond too. For each one, we send the right metrics their way.
func (m *metricWriter) flush(buffer []shared.Metric) {
	// create buffers for each unique kind of metric
	eventMetrics := make([]*shared.EventMetric, 0, m.bufferSize)
	profilingMetrics := make([]*shared.ProfilingMetrics, 0, m.bufferSize)
	pluginMetrics := make([]*shared.PluginMetrics, 0, m.bufferSize)
	requestMetrics := make([]*shared.RequestMetric, 0, m.bufferSize)
	upstreamMetrics := make([]*shared.UpstreamMetric, 0, m.bufferSize)

	// bucket metrics by their type
	for _, metric := range buffer {
		switch v := metric.(type) {
		case *shared.EventMetric:
			eventMetrics = append(eventMetrics, event.(*shared.EventMetric))
		case *shared.ProfilingMetric:
			profilingMetrics = append(profilingMetrics, event.(*shared.ProfilingMetric))
		case *shared.PluginMetric:
			pluginMetrics = append(pluginMetrics, event.(*shared.PluginMetric))
		case *shared.RequestMetric:
			requestMetrics = append(requestMetrics, event.(*shared.RequestMetric))
		case *shared.UpstreamMetric:
			upstreamMetric = append(upstreamMetrics, event.(*shared.UpstreamMetric))
		default:
			shared.ProgrammingError("unknown buffered metric")
		}
	}

	var wg sync.WaitGroup

	// now for each plugin, try to write the metrics to each interface that
	// it happens to implement. Any plugin that wants to receive metrics
	// can simply implement any of the interfaces and they will be batched
	// to them!
	for _, pluginManager := range m.pluginManagers {
		wg.Add(1)

		go func(pluginManager PluginManager) {
			defer wg.Done()

			// Grab the plugin to do type switching, to decide which metrics to write to it.
			pluginManager.Grab(func(plugin Plugin) {
				// write event metrics
				if _, ok := plugin.(eventMetricsReceiver); ok {
					plugin.Call("WriteEventMetrics", func(plugin Plugin) error {
						return plugin.(eventMetricsReceiver).WriteEventMetrics(eventMetrics)
					})
				}

				// write plugin metrics
				if _, ok := plugin.(pluginMetricsReceiver); ok {
					plugin.Call("WritePluginMetrics", func(plugin Plugin) error {
						return plugin.(pluginMetricsReceiver).WritePluginMetrics(pluginMetrics)
					})
				}

				// write profiling metrics
				if _, ok := plugin.(profilingMetricsReceiver); ok {
					plugin.Call("WriteProfilingMetrics", func(plugin Plugin) error {
						return plugin.(profilingMetricsReceiver).WriteProfilingMetrics(profilingMetrics)
					})
				}

				// write request metrics
				if _, ok := plugin.(requestMetricsReceiver); ok {
					plugin.Call("WriteRequestMetrics", func(plugin Plugin) error {
						return plugin.(requestMetricsReceiver).WriteRequestMetrics(requestMetrics)
					})
				}

				// write upstream metrics
				if _, ok := plugin.(upstreamMetricsReceiver); ok {
					plugin.Call("WriteUpstreamMetrics", func(plugin Plugin) error {
						return plugin.(upstreamMetricsReceiver).WriteUpstreamMetrics(upstreamMetrics)
					})
				}
			})
		}()
	}

	wg.Wait()
}
