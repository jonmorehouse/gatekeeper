package gatekeeper

// this is a local representation of the upstream.Manager (plugin) interface.
// This is copied here so we don't have to re-import the plugin package,
// causing a circular dependency
type Manager interface {
	AddUpstream(*Upstream) error
	RemoveUpstream(UpstreamID) error

	AddBackend(UpstreamID, *Backend) error
	RemoveBackend(BackendID) error
}

// ServiceContainer is a container which persists both upstreams and backends.
// Specifically, this container is useful for building upstream plugins. It is
// used internally by the core.UpstreamManager type to persist backends and
// upstreams.
type ServiceContainer interface {
	UpstreamContainer
	BackendContainer
}

func NewServiceContainer() ServiceContainer {
	return &serviceContainer{
		UpstreamContainer: NewUpstreamContainer(),
		BackendContainer:  NewBackendContainer(),
	}
}

type serviceContainer struct {
	UpstreamContainer
	BackendContainer
}

// SyncedServiceContainer is a container which syncs upstreams and backends
// locally and with a Manager interface, which in most cases will correspond to
// the actual parent, gatekeeper-core process
func NewSyncedServiceContainer(manager Manager) ServiceContainer {
	return &syncedServiceContainer{
		manager:           manager,
		UpstreamContainer: NewUpstreamContainer(),
		BackendContainer:  NewBackendContainer(),
	}
}

type syncedServiceContainer struct {
	manager Manager
	UpstreamContainer
	BackendContainer
}

func (s *syncedServiceContainer) AddUpstream(upstream *Upstream) error {
	if err := s.manager.AddUpstream(upstream); err != nil {
		return err
	}

	return s.UpstreamContainer.AddUpstream(upstream)
}

func (s *syncedServiceContainer) RemoveUpstream(upstreamID UpstreamID) error {
	if err := s.manager.RemoveUpstream(upstreamID); err != nil {
		return err
	}

	return s.UpstreamContainer.RemoveUpstream(upstreamID)
}

func (s *syncedServiceContainer) AddBackend(upstreamID UpstreamID, backend *Backend) error {
	if err := s.manager.AddBackend(upstreamID, backend); err != nil {
		return err
	}

	return s.BackendContainer.AddBackend(upstreamID, backend)
}

func (s *syncedServiceContainer) RemoveBackend(backendID BackendID) error {
	if err := s.manager.RemoveBackend(backendID); err != nil {
		return err
	}

	return s.BackendContainer.RemoveBackend(backendID)
}
