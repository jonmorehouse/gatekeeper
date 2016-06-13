package gatekeeper

import (
	"fmt"
	"log"
	"sync"
	"time"

	event_plugin "github.com/jonmorehouse/gatekeeper/plugin/event"
	"github.com/jonmorehouse/gatekeeper/shared"
)

type MetricWriter interface {
	Start() error
	Stop(time.Duration) error

	GeneralMetric(*shared.GeneralMetric)
	RequestMetric(*shared.RequestMetric)
	Error(error)
}

// metricWriter is a type that implements the MetricWriter interface. It
// buffers metrics locally and flushes them
type metricWriter struct {
	// generate channels to pass metrics to our worker on.
	generalCh chan *shared.GeneralMetric
	requestCh chan *shared.RequestMetric
	errorCh   chan error
	stopCh    chan struct{}

	generalBuffer    []*shared.GeneralMetric
	maxGeneralBuffer int

	requestBuffer    []*shared.RequestMetric
	maxRequestBuffer int

	errorBuffer    []error
	maxErrorBuffer int

	maxFlushInterval time.Duration
	pluginManagers   []PluginManager
}

func NewMetricWriter(pluginManagers []PluginManager) MetricWriter {
	maxGeneralBuffer := 100
	maxRequestBuffer := 100
	maxErrorBuffer := 100

	return &metricWriter{
		generalCh: make(chan *shared.GeneralMetric, maxGeneralBuffer/2),
		requestCh: make(chan *shared.RequestMetric, maxRequestBuffer/2),
		errorCh:   make(chan error, maxErrorBuffer/2),
		stopCh:    make(chan struct{}),

		generalBuffer:    make([]*shared.GeneralMetric, 0, maxGeneralBuffer),
		maxGeneralBuffer: maxGeneralBuffer,

		requestBuffer:    make([]*shared.RequestMetric, 0, maxRequestBuffer),
		maxRequestBuffer: maxRequestBuffer,

		errorBuffer:    make([]error, 0, maxErrorBuffer),
		maxErrorBuffer: maxErrorBuffer,

		maxFlushInterval: time.Millisecond * 100,
		pluginManagers:   pluginManagers,
	}
}

func (m *metricWriter) Start() error {
	errs := NewAsyncMultiError()
	var wg sync.WaitGroup

	for _, pluginManager := range m.pluginManagers {
		wg.Add(1)
		go func(p PluginManager) {
			if err := p.Start(); err != nil {
				errs.Add(err)
			}
			wg.Done()
		}(pluginManager)
	}

	wg.Wait()
	go m.worker()
	return errs.ToErr()
}

func (m *metricWriter) Stop(duration time.Duration) error {
	// make sure that we wait for all messages to flush to the plugins,
	// regardless of how long it takes
	m.stopCh <- struct{}{}
	<-m.stopCh

	errs := NewAsyncMultiError()
	var wg sync.WaitGroup

	for _, pluginManager := range m.pluginManagers {
		wg.Add(1)
		go func(p PluginManager) {
			if err := p.Stop(duration); err != nil {
				log.Println(err)
				errs.Add(err)
			}
			wg.Done()
		}(pluginManager)
	}

	wg.Wait()
	return errs.ToErr()
}

func (m *metricWriter) GeneralMetric(metric *shared.GeneralMetric) {
	m.generalCh <- metric
}

func (m *metricWriter) RequestMetric(metric *shared.RequestMetric) {
	m.requestCh <- metric
}

func (m *metricWriter) Error(err error) {
	m.errorCh <- err
}

func (m *metricWriter) worker() {
	generalFlushCh := time.After(m.maxFlushInterval)
	requestFlushCh := time.After(m.maxFlushInterval)
	errorFlushCh := time.After(m.maxFlushInterval)

	// for each of the unique types of messages we are receiving, we
	// capture messages from each individual channel. Furthermore, we set
	// up channels that will emit a message after timeouts to set up a
	// "periodic" flush.
	for {
		select {
		case metric := <-m.generalCh:
			m.generalBuffer = append(m.generalBuffer, metric)
			if len(m.generalBuffer) >= m.maxGeneralBuffer {
				m.flushGeneral()
			}
			generalFlushCh = time.After(m.maxFlushInterval)
		case <-generalFlushCh:
			m.flushGeneral()
			generalFlushCh = time.After(m.maxFlushInterval)
		case metric := <-m.requestCh:
			m.requestBuffer = append(m.requestBuffer, metric)
			if len(m.requestBuffer) >= m.maxRequestBuffer {
				m.flushRequest()
			}
			requestFlushCh = time.After(m.maxFlushInterval)
		case <-requestFlushCh:
			m.flushRequest()
			requestFlushCh = time.After(m.maxFlushInterval)
		case err := <-m.errorCh:
			m.errorBuffer = append(m.errorBuffer, err)
			if len(m.errorBuffer) >= m.maxErrorBuffer {
				m.flushError()
			}
			errorFlushCh = time.After(m.maxFlushInterval)
		case <-errorFlushCh:
			m.flushError()
			errorFlushCh = time.After(m.maxFlushInterval)
		case <-m.stopCh:
			goto finished
		}
	}

finished:
	m.flushGeneral()
	m.flushRequest()
	m.flushError()
	m.stopCh <- struct{}{}
}

func (m *metricWriter) getPlugins() ([]event_plugin.PluginClient, error) {
	plugins := make([]event_plugin.PluginClient, 0, len(m.pluginManagers))

	for _, pluginManager := range m.pluginManagers {
		rawPlugin, err := pluginManager.Get()
		if err != nil {
			return []event_plugin.PluginClient(nil), err
		}

		plugin, ok := rawPlugin.(event_plugin.PluginClient)
		if !ok {
			return []event_plugin.PluginClient(nil), fmt.Errorf("INVALID_EVENT_PLUGIN")
		}

		plugins = append(plugins, plugin)
	}

	return plugins, nil

}

func (m *metricWriter) flushGeneral() {
	if len(m.generalBuffer) == 0 {
		return
	}

	plugins, err := m.getPlugins()
	if err != nil {
		log.Println("Unable to fetch event plugins.")
		return
	}

	var wg sync.WaitGroup
	for _, plugin := range plugins {
		wg.Add(1)
		go func(plugin event_plugin.PluginClient) {
			if err := plugin.WriteGeneralMetrics(m.generalBuffer); err != nil {
				log.Println("FAILED_TO_WRITE_GENERAL_METRICS")
			}
			wg.Done()
		}(plugin)
	}
	wg.Wait()
	m.generalBuffer = m.generalBuffer[:0]
}

func (m *metricWriter) flushRequest() {
	if len(m.requestBuffer) == 0 {
		return
	}

	plugins, err := m.getPlugins()
	if err != nil {
		log.Println("Unable to fetch event plugins.")
		return
	}

	var wg sync.WaitGroup
	for _, plugin := range plugins {
		wg.Add(1)
		go func(plugin event_plugin.PluginClient) {
			if err := plugin.WriteRequestMetrics(m.requestBuffer); err != nil {
				log.Println("FAILED_TO_WRITE_REQUEST_METRICS")
			}
			wg.Done()
		}(plugin)
	}
	wg.Wait()
	m.requestBuffer = m.requestBuffer[:0]
}

func (m *metricWriter) flushError() {
	if len(m.errorBuffer) == 0 {
		return
	}

	plugins, err := m.getPlugins()
	if err != nil {
		log.Println("Unable to fetch event plugins.")
		return
	}

	var wg sync.WaitGroup
	for _, plugin := range plugins {
		wg.Add(1)
		go func(plugin event_plugin.PluginClient) {
			if err := plugin.WriteErrors(m.errorBuffer); err != nil {
				log.Println("FAILED_TO_WRITE_ERRORS_TO_BACKEND")
			}
			wg.Done()
		}(plugin)
	}
	wg.Wait()
	m.errorBuffer = m.errorBuffer[:0]
}
