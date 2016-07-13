package core

import (
	"sync"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

// MetricWriterClient is an interface which is passed around simply to write
// metrics too. Most writers of metrics will be from metrics client, where as
// some sort of higher level "manager-like" type will be responsible for the
// full lifecycle of the MetricWriter
type MetricWriterClient interface {
	EventMetric(*gatekeeper.EventMetric)
	ProfilingMetric(*gatekeeper.ProfilingMetric)
	PluginMetric(*gatekeeper.PluginMetric)
	RequestMetric(*gatekeeper.RequestMetric)
	UpstreamMetric(*gatekeeper.UpstreamMetric)
}

type MetricWriter interface {
	startStopper

	AddPlugin(PluginManager)
	MetricWriterClient
}

// The MetricsWriter doesn't mind which metrics a plugin is interested in. For
// each unique plugin type, we attempt to flush the buffered metrics to any
// plugins that desire them. This makes it easier to pass metrics between new
// plugin types dynamically.
type eventMetricsReceiver interface {
	WriteEventMetrics([]*gatekeeper.EventMetric) []error
}

type profilingMetricsReceiver interface {
	WriteProfilingMetrics([]*gatekeeper.ProfilingMetric) []error
}

type pluginMetricsReceiver interface {
	WritePluginMetrics([]*gatekeeper.PluginMetric) []error
}

type requestMetricsReceiver interface {
	WriteRequestMetrics([]*gatekeeper.RequestMetric) []error
}

type upstreamMetricsReceiver interface {
	WriteUpstreamMetrics([]*gatekeeper.UpstreamMetric) []error
}

func NewMetricWriter(bufferSize int, flushInterval time.Duration) MetricWriter {
	return &metricWriter{
		bufferSize:    bufferSize,
		flushInterval: flushInterval,
		buffer:        make([]gatekeeper.Metric, 0, int(bufferSize)),

		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		bufferCh: make(chan gatekeeper.Metric, 10000),
	}
}

type metricWriter struct {
	pluginManagers []PluginManager

	buffer        []gatekeeper.Metric
	bufferSize    int
	flushInterval time.Duration

	stopCh   chan struct{}
	doneCh   chan struct{}
	bufferCh chan gatekeeper.Metric

	sync.RWMutex
}

func (m *metricWriter) Start() error                      { return nil }
func (m *metricWriter) Stop(duration time.Duration) error { return nil }

func (m *metricWriter) AddPlugin(plugin PluginManager) {
	m.Lock()
	defer m.Unlock()
	m.pluginManagers = append(m.pluginManagers, plugin)
}

func (m *metricWriter) EventMetric(event *gatekeeper.EventMetric) {
	m.bufferCh <- event
}

func (m *metricWriter) ProfilingMetric(event *gatekeeper.ProfilingMetric) {
	m.bufferCh <- event
}

func (m *metricWriter) PluginMetric(event *gatekeeper.PluginMetric) {
	m.bufferCh <- event
}

func (m *metricWriter) RequestMetric(event *gatekeeper.RequestMetric) {
	m.bufferCh <- event
}

func (m *metricWriter) UpstreamMetric(event *gatekeeper.UpstreamMetric) {
	m.bufferCh <- event

}

func (m *metricWriter) worker() {
	timer := time.NewTimer(m.flushInterval)

	flush := func() {
		go m.flush(m.buffer)
		m.buffer = make([]gatekeeper.Metric, 0, m.bufferSize)
		timer.Reset(m.flushInterval)
	}

	defer func() {
		timer.Stop()
		m.flush(m.buffer)
		m.buffer = []gatekeeper.Metric(nil)
		m.doneCh <- struct{}{}
	}()

	for {
		select {
		case metric := <-m.bufferCh:
			m.buffer = append(m.buffer, metric)
			if len(m.buffer) == m.bufferSize {
				flush()
			}
		case <-timer.C:
			flush()
		case <-m.stopCh:
			return
		}
	}
}

// flush metrics to their correct methods on the correct plugins. This is a big
// method, but its rather simple at a high level. We start by type-asserting
// each buffered metric. This gives us a slice of each metric by type.	for
// each of our plugins, we then go through them and see which metric interfaces
// they correspond too. For each one, we send the right metrics their way.
func (m *metricWriter) flush(buffer []gatekeeper.Metric) {
	// create buffers for each unique kind of metric
	eventMetrics := make([]*gatekeeper.EventMetric, 0, m.bufferSize)
	profilingMetrics := make([]*gatekeeper.ProfilingMetric, 0, m.bufferSize)
	pluginMetrics := make([]*gatekeeper.PluginMetric, 0, m.bufferSize)
	requestMetrics := make([]*gatekeeper.RequestMetric, 0, m.bufferSize)
	upstreamMetrics := make([]*gatekeeper.UpstreamMetric, 0, m.bufferSize)

	// bucket metrics by their type
	for _, metric := range buffer {
		switch metric.(type) {
		case *gatekeeper.EventMetric:
			eventMetrics = append(eventMetrics, metric.(*gatekeeper.EventMetric))
		case *gatekeeper.ProfilingMetric:
			profilingMetrics = append(profilingMetrics, metric.(*gatekeeper.ProfilingMetric))
		case *gatekeeper.PluginMetric:
			pluginMetrics = append(pluginMetrics, metric.(*gatekeeper.PluginMetric))
		case *gatekeeper.RequestMetric:
			requestMetrics = append(requestMetrics, metric.(*gatekeeper.RequestMetric))
		case *gatekeeper.UpstreamMetric:
			upstreamMetrics = append(upstreamMetrics, metric.(*gatekeeper.UpstreamMetric))
		default:
			gatekeeper.ProgrammingError("unknown buffered metric")
		}
	}

	m.RLock()
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
					pluginManager.Call("WriteEventMetrics", func(plugin Plugin) error {
						errs := plugin.(eventMetricsReceiver).WriteEventMetrics(eventMetrics)
						return (&MultiError{errs: errs}).ToErr()
					})
				}

				// write plugin metrics
				if _, ok := plugin.(pluginMetricsReceiver); ok {
					pluginManager.Call("WritePluginMetrics", func(plugin Plugin) error {
						errs := plugin.(pluginMetricsReceiver).WritePluginMetrics(pluginMetrics)
						return (&MultiError{errs: errs}).ToErr()
					})
				}

				// write profiling metrics
				if _, ok := plugin.(profilingMetricsReceiver); ok {
					pluginManager.Call("WriteProfilingMetrics", func(plugin Plugin) error {
						errs := plugin.(profilingMetricsReceiver).WriteProfilingMetrics(profilingMetrics)
						return (&MultiError{errs: errs}).ToErr()
					})
				}

				// write request metrics
				if _, ok := plugin.(requestMetricsReceiver); ok {
					pluginManager.Call("WriteRequestMetrics", func(plugin Plugin) error {
						errs := plugin.(requestMetricsReceiver).WriteRequestMetrics(requestMetrics)
						return (&MultiError{errs: errs}).ToErr()
					})
				}

				// write upstream metrics
				if _, ok := pluginManager.(upstreamMetricsReceiver); ok {
					pluginManager.Call("WriteUpstreamMetrics", func(plugin Plugin) error {
						errs := plugin.(upstreamMetricsReceiver).WriteUpstreamMetrics(upstreamMetrics)
						return (&MultiError{errs: errs}).ToErr()
					})
				}
			})
		}(pluginManager)
	}

	m.RUnlock()
	wg.Wait()
}
