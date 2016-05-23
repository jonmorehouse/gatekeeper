package upstream

// manager is the interface which clients use to talk back to the gatekeeper parent process
type Manager interface {
	AddUpstream(Upstream) (UpstreamID, error)
	RemoveUpstream(UpstreamID) error

	AddBackend(UpstreamID, Backend) (BackendID, error)
	RemoveBackend(BackendID) error
}
