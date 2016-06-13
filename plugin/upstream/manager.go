package upstream

import (
	"github.com/jonmorehouse/gatekeeper/shared"
)

type Manager interface {
	AddUpstream(*shared.Upstream) error
	RemoveUpstream(shared.UpstreamID) error

	AddBackend(shared.UpstreamID, *shared.Backend) error
	RemoveBackend(shared.BackendID) error
}

type ManagerRPC interface {
	AddUpstream(*shared.Upstream) *shared.Error
	RemoveUpstream(shared.UpstreamID) *shared.Error

	AddBackend(shared.UpstreamID, *shared.Backend) *shared.Error
	RemoveBackend(shared.BackendID) *shared.Error

	// NOTE this method is not accessible to the plugin implemeneter, it's
	// used internally to ensure that the plugin is still connected back to
	// the parent process via RPC.
	Heartbeat() *shared.Error
}

// ManagerClient is a client which wraps a ManagerRPC implementing type and
// exposes it in a friendly way, abstracting the concrete error types. This is
// passed along to the client itself, allowing it to make seamlessly make
// requests back into the parent process over RPC.
type ManagerClient struct {
	ManagerRPC ManagerRPC
}

func (m *ManagerClient) AddUpstream(upstream *shared.Upstream) error {
	return m.ManagerRPC.AddUpstream(upstream)
}

func (m *ManagerClient) RemoveUpstream(upstreamID shared.UpstreamID) error {
	return m.ManagerRPC.RemoveUpstream(upstreamID)
}

func (m *ManagerClient) AddBackend(upstreamID shared.UpstreamID, backend *shared.Backend) error {
	return m.ManagerRPC.AddBackend(upstreamID, backend)
}

func (m *ManagerClient) RemoveBackend(backendID shared.BackendID) error {
	return m.ManagerRPC.RemoveBackend(backendID)
}
