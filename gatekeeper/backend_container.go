package gatekeeper

import "sync"

// BackendContainer is a light weight, backend only service store.
// Specifically, its helpful as a building block for load balancer plugins as
// it relates backends to UpstreamIDs only.
type BackendContainer interface {
	AddBackend(UpstreamID, *Backend) error
	RemoveBackend(BackendID) error
	RemoveAllBackends() error
	BackendByID(BackendID) (*Backend, error)

	FetchUpstreamID(BackendID) (UpstreamID, error)
	FetchBackends(UpstreamID) ([]*Backend, error)
	FetchAllBackends() []*Backend
}

func NewBackendContainer() BackendContainer {
	return &backendContainer{
		backends:         make(map[BackendID]*Backend),
		backendUpstreams: make(map[BackendID]UpstreamID),
		upstreamBackends: make(map[UpstreamID]map[BackendID]struct{}),
	}
}

type backendContainer struct {
	backends         map[BackendID]*Backend
	backendUpstreams map[BackendID]UpstreamID
	upstreamBackends map[UpstreamID]map[BackendID]struct{}

	sync.RWMutex
}

func (b *backendContainer) AddBackend(uID UpstreamID, backend *Backend) error {
	b.Lock()
	defer b.Unlock()

	b.backends[backend.ID] = backend
	b.backendUpstreams[backend.ID] = uID

	if _, ok := b.upstreamBackends[uID]; !ok {
		b.upstreamBackends[uID] = make(map[BackendID]struct{})
	}
	b.upstreamBackends[uID][backend.ID] = struct{}{}
	return nil
}

func (b *backendContainer) RemoveBackend(bID BackendID) error {
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

func (b *backendContainer) BackendByID(bID BackendID) (*Backend, error) {
	b.RLock()
	defer b.RUnlock()
	backend, ok := b.backends[bID]
	if !ok {
		return nil, BackendNotFoundErr
	}
	return backend, nil
}

func (b *backendContainer) FetchUpstreamID(bID BackendID) (UpstreamID, error) {
	b.RLock()
	defer b.RUnlock()
	if uID, ok := b.backendUpstreams[bID]; ok {
		return uID, nil
	}
	return "", BackendNotFoundErr
}

func (b *backendContainer) FetchBackends(uID UpstreamID) ([]*Backend, error) {
	b.RLock()
	defer b.RUnlock()

	backendIDs, ok := b.upstreamBackends[uID]
	if !ok {
		return []*Backend(nil), UpstreamNotFoundErr
	}

	backends := make([]*Backend, len(backendIDs))
	idx := 0

	for backendID, _ := range backendIDs {
		backends[idx] = b.backends[backendID]
		idx += 1
	}

	return backends, nil
}

func (b *backendContainer) FetchAllBackends() []*Backend {
	b.RLock()
	defer b.RUnlock()

	backends := make([]*Backend, len(b.backends))
	ctr := 0

	for _, backend := range backends {
		backends[ctr] = backend
		ctr += 1
	}

	return backends
}
