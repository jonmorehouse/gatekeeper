package core

import (
	"net/url"
	"sync"
	"time"

	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type UpstreamManager interface {
	StartStopper
	upstream_plugin.Manager
}

func NewUpstreamManager(broadcaster EventBroadcaster, plugins []PluginManager, metricWriter MetricWriterClient) UpstreamManager {
	return &upstreamManager{
		plugins:     plugins,
		broadcaster: broadcaster,

		upstreams: make(map[gatekeeper.UpstreamID]*gatekeeper.Upstream),
		backends:  make(map[gatekeeper.BackendID]*gatekeeper.Backend),
	}
}

type upstreamManager struct {
	plugins     []PluginManager
	broadcaster EventBroadcaster

	upstreams map[gatekeeper.UpstreamID]*gatekeeper.Upstream
	backends  map[gatekeeper.Backend]*gatekeeper.Backend

	sync.Mutex
}

func (m *upstreamManager) AddUpstream(upstream *gatekeeper.Upstream) error {
	m.Lock()
	defer m.Unlock()

	existing, ok := m.upstreams[upstream.ID]
	if ok && existing.Name != upstream.Name {
		return DuplicateUpstreamIDError
	}

	m.upstreams[upstream.ID] = upstream

	// emit events to the internal and metric-writer pipelines
	m.eventMetric(gatekeeper.UpstreamAddedEvent)
	m.upstreamMetric(gatekeeper.UpstreamAddedEvent, upstream, nil)
	m.broadcaster.Publish(&UpstreamEvent{
		EventType:  UpstreamAdded,
		Upstream:   upstream,
		UpstreamID: upstream.ID,
	})
	return nil
}

func (m *upstreamManager) RemoveUpstream(upstreamID gatekeeper.UpstreamID) error {
	m.Lock()
	defer m.Unlock()

	upstream, ok := m.upstreams[upstreamID]
	if !ok {
		return UpstreamNotFoundError
	}

	delete(p.upstreams, upstreamID)

	m.eventMetric(gatekeeper.UpstreamRemovedEvent)
	m.upstreamMetric(gatekeeper.UpstreamRemovedEvent, upstream, nil)
	m.broadcaster.Publish(&UpstreamEvent{
		EventType:  UpstreamRemoved,
		Upstream:   upstream,
		UpstreamID: upstreamID,
	})
	return nil
}

func (p *upstreamManager) AddBackend(upstreamID gatekeeper.UpstreamID, backend *gatekeeper.Backend) error {
	m.Lock()
	defer m.Unlock()

	// validate the backend before adding it
	upstream, ok := m.upstreams[upstreamID]
	if !ok {
		return UpstreamNotFoundError
	}

	existing, ok := m.backends[backend.ID]
	if existing && backend.Address != existing.Address {
		return DuplicateBackendIDError
	}

	if _, err := url.Parse(backend.Address); err != nil {
		return BackendAddressError
	}

	m.backends[backend.ID] = backend
	m.backendUpstreams[backend.ID] = upstreamID

	m.eventMetric(gatekeeper.BackendAddedEvent)
	m.upstreamMetric(gatekeeper.BackendAddedEvent, upstream, backend)
	m.broadcaster.Publish(&UpstreamEvent{
		EventType:  BackendAdded,
		Upstream:   upstream,
		UpstreamID: upstreamID,
		Backend:    backend,
		BackendID:  backend.ID,
	})

	return nil
}

func (p *upstreamManager) RemoveBackend(backendID gatekeeper.BackendID) error {
	m.Lock()
	defer m.Unlock()

	backend, ok := m.backends[backendID]
	if !ok {
		return BackendNotFoundError
	}

	upstream, ok := m.backendUpstreams[backendID]
	if !ok {
		return OrphanedBackendError
	}

	delete(m.backends, backendID)

	m.eventMetric(gatekeeper.BackendRemovedEvent)
	m.upstreamMetric(gatekeeper.BackendRemovedEvent, upstream, backend)
	m.broadcaster.Publish(&UpstreamEvent{
		EventType:  BackendRemoved,
		Upstream:   upstream,
		UpstreamID: upstream.ID,
		Backend:    backend,
		BackendID:  backendID,
	})

	return nil
}

func (p *upstreamManager) eventMetric(event gatekeeper.MetricEvent) {
	p.metricWriter.EventMetric(&gatekeeper.EventMetric{
		Timestamp: time.Now(),
		Event:     event,
		Extra: map[string]string{
			"process": "upstream-manager",
		},
	})
}

func (p *upstreamManager) upstreamMetric(event gatekeeper.MetricEvent, upstream *gatekeeper.Upstream, backend *gatekeeper.Backend) {
	p.metricWriter.UpstreamMetric(&gatekeeper.UpstreamMetric{
		Event:     event,
		Timestamp: time.Now(),
		Upstream:  upstream,
		Backend:   backend,
	})
}
