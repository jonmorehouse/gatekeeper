package core

import (
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

type UpstreamManager interface {
	startStopper
	upstream_plugin.Manager
}

func NewUpstreamManager(broadcaster Broadcaster, metricWriter MetricWriterClient) UpstreamManager {
	return &upstreamManager{
		broadcaster: broadcaster,

		upstreams:        make(map[gatekeeper.UpstreamID]*gatekeeper.Upstream),
		backends:         make(map[gatekeeper.BackendID]*gatekeeper.Backend),
		backendUpstreams: make(map[gatekeeper.BackendID]gatekeeper.UpstreamID),

		metricWriter: metricWriter,
	}
}

type upstreamManager struct {
	broadcaster Broadcaster

	upstreams        map[gatekeeper.UpstreamID]*gatekeeper.Upstream
	backends         map[gatekeeper.BackendID]*gatekeeper.Backend
	backendUpstreams map[gatekeeper.BackendID]gatekeeper.UpstreamID

	metricWriter MetricWriterClient

	sync.Mutex
}

func (m *upstreamManager) Start() error             { return nil }
func (m *upstreamManager) Stop(time.Duration) error { return nil }

func (m *upstreamManager) AddUpstream(upstream *gatekeeper.Upstream) error {
	m.Lock()
	defer m.Unlock()

	existing, ok := m.upstreams[upstream.ID]
	if ok && existing.Name != upstream.Name {
		return DuplicateUpstreamErr
	}

	m.upstreams[upstream.ID] = upstream

	// emit events to the internal and metric-writer pipelines
	m.eventMetric(gatekeeper.UpstreamAddedEvent)
	m.upstreamMetric(gatekeeper.UpstreamAddedEvent, upstream, nil)
	m.broadcaster.Publish(&UpstreamEvent{
		Event:      gatekeeper.UpstreamAddedEvent,
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

	delete(m.upstreams, upstreamID)

	m.eventMetric(gatekeeper.UpstreamRemovedEvent)
	m.upstreamMetric(gatekeeper.UpstreamRemovedEvent, upstream, nil)
	log.Println("upstream broadcasted...")
	m.broadcaster.Publish(&UpstreamEvent{
		Event:      gatekeeper.UpstreamRemovedEvent,
		Upstream:   upstream,
		UpstreamID: upstreamID,
	})
	return nil
}

func (m *upstreamManager) AddBackend(upstreamID gatekeeper.UpstreamID, backend *gatekeeper.Backend) error {
	m.Lock()
	defer m.Unlock()

	// validate the backend before adding it
	upstream, ok := m.upstreams[upstreamID]
	if !ok {
		return UpstreamNotFoundError
	}

	existing, ok := m.backends[backend.ID]
	if ok && (backend.Address != existing.Address) {
		return DuplicateBackendErr
	}

	if _, err := url.Parse(backend.Address); err != nil {
		return BackendAddressErr
	}

	m.backends[backend.ID] = backend
	m.backendUpstreams[backend.ID] = upstreamID

	m.eventMetric(gatekeeper.BackendAddedEvent)
	m.upstreamMetric(gatekeeper.BackendAddedEvent, upstream, backend)

	log.Println("backend broadcasted...")
	m.broadcaster.Publish(&UpstreamEvent{
		Event:      gatekeeper.BackendAddedEvent,
		Upstream:   upstream,
		UpstreamID: upstreamID,
		Backend:    backend,
		BackendID:  backend.ID,
	})

	return nil
}

func (m *upstreamManager) RemoveBackend(backendID gatekeeper.BackendID) error {
	m.Lock()
	defer m.Unlock()

	backend, ok := m.backends[backendID]
	if !ok {
		return BackendNotFoundError
	}

	upstream, ok := m.upstreams[m.backendUpstreams[backendID]]
	if !ok {
		return OrphanedBackendError
	}

	delete(m.backends, backendID)

	m.eventMetric(gatekeeper.BackendRemovedEvent)
	m.upstreamMetric(gatekeeper.BackendRemovedEvent, upstream, backend)
	m.broadcaster.Publish(&UpstreamEvent{
		Event:      gatekeeper.BackendRemovedEvent,
		Upstream:   upstream,
		UpstreamID: upstream.ID,
		Backend:    backend,
		BackendID:  backendID,
	})

	return nil
}

func (p *upstreamManager) eventMetric(event gatekeeper.Event) {
	p.metricWriter.EventMetric(&gatekeeper.EventMetric{
		Timestamp: time.Now(),
		Event:     event,
		Extra: map[string]string{
			"process": "upstream-manager",
		},
	})
}

func (p *upstreamManager) upstreamMetric(event gatekeeper.Event, upstream *gatekeeper.Upstream, backend *gatekeeper.Backend) {
	p.metricWriter.UpstreamMetric(&gatekeeper.UpstreamMetric{
		Event:     event,
		Timestamp: time.Now(),
		Upstream:  upstream,
		Backend:   backend,
	})
}
