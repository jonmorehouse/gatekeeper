package utils

import (
	"sync"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

// BackendContainer is a light weight, backend only service store.
// Specifically, its helpful as a building block for load balancer plugins as
// it relates backends to UpstreamIDs only.
type BackendContainer interface {
	AddBackend(gatekeeper.UpstreamID, *gatekeeper.Backend) error
	RemoveBackend(gatekeeper.BackendID) error
	RemoveAllBackends() error
	BackendByID(gatekeeper.BackendID) (*gatekeeper.Backend, error)

	FetchUpstreamID(gatekeeper.BackendID) (gatekeeper.UpstreamID, error)
	FetchBackends(gatekeeper.UpstreamID) ([]*gatekeeper.Backend, error)
	FetchAllBackends() []*gatekeeper.Backend
}

func NewBackendContainer() BackendContainer {
	return &backendContainer{
		backends:         make(map[gatekeeper.BackendID]*gatekeeper.Backend),
		backendUpstreams: make(map[gatekeeper.BackendID]gatekeeper.UpstreamID),
		upstreamBackends: make(map[gatekeeper.UpstreamID]map[gatekeeper.BackendID]struct{}),
	}
}

type backendContainer struct {
	backends         map[gatekeeper.BackendID]*gatekeeper.Backend
	backendUpstreams map[gatekeeper.BackendID]gatekeeper.UpstreamID
	upstreamBackends map[gatekeeper.UpstreamID]map[gatekeeper.BackendID]struct{}

	sync.RWMutex
}

func (b *backendContainer) AddBackend(uID gatekeeper.UpstreamID, backend *gatekeeper.Backend) error {
	b.Lock()
	defer b.Unlock()

	b.backends[backend.ID] = backend
	b.backendUpstreams[backend.ID] = uID

	if _, ok := b.upstreamBackends[uID]; !ok {
		b.upstreamBackends[uID] = make(map[gatekeeper.BackendID]struct{})
	}
	b.upstreamBackends[uID][backend.ID] = struct{}{}
	return nil
}

func (b *backendContainer) RemoveBackend(bID gatekeeper.BackendID) error {
	if _, err := b.BackendByID(bID); err != nil {
		return err
	}

	b.Lock()
	defer b.Unlock()

	delete(b.backends, bID)
	upstreamID, ok := b.backendUpstreams[bID]
	if ok {
		if upstreamBackends, ok := b.upstreamBackends[upstreamID]; ok {
			delete(upstreamBackends, bID)
		}
	}

	delete(b.backendUpstreams, bID)
	return nil
}

func (b *backendContainer) RemoveAllBackends() error {
	var err error

	for _, backend := range b.FetchAllBackends() {
		if e := b.RemoveBackend(backend.ID); e != nil {
			err = e
		}
	}

	return err
}

func (b *backendContainer) BackendByID(bID gatekeeper.BackendID) (*gatekeeper.Backend, error) {
	b.RLock()
	defer b.RUnlock()
	backend, ok := b.backends[bID]
	if !ok {
		return nil, gatekeeper.BackendNotFoundErr
	}
	return backend, nil
}

func (b *backendContainer) FetchUpstreamID(bID gatekeeper.BackendID) (gatekeeper.UpstreamID, error) {
	b.RLock()
	defer b.RUnlock()
	if uID, ok := b.backendUpstreams[bID]; ok {
		return uID, nil
	}
	return "", gatekeeper.BackendNotFoundErr
}

func (b *backendContainer) FetchBackends(uID gatekeeper.UpstreamID) ([]*gatekeeper.Backend, error) {
	b.RLock()
	defer b.RUnlock()

	backendIDs, ok := b.upstreamBackends[uID]
	if !ok {
		return []*gatekeeper.Backend(nil), gatekeeper.UpstreamNotFoundErr
	}

	backends := make([]*gatekeeper.Backend, len(backendIDs))
	idx := 0

	for backendID, _ := range backendIDs {
		backends[idx] = b.backends[backendID]
		idx += 1
	}

	return backends, nil
}

func (b *backendContainer) FetchAllBackends() []*gatekeeper.Backend {
	b.RLock()
	defer b.RUnlock()

	backends := make([]*gatekeeper.Backend, len(b.backends))
	ctr := 0

	for _, backend := range backends {
		backends[ctr] = backend
		ctr += 1
	}

	return backends
}
