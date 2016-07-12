package upstream

import (
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type Manager interface {
	AddUpstream(*gatekeeper.Upstream) error
	RemoveUpstream(gatekeeper.UpstreamID) error

	AddBackend(gatekeeper.UpstreamID, *gatekeeper.Backend) error
	RemoveBackend(gatekeeper.BackendID) error
}

type ManagerRPC interface {
	AddUpstream(*gatekeeper.Upstream) *gatekeeper.Error
	RemoveUpstream(gatekeeper.UpstreamID) *gatekeeper.Error

	AddBackend(gatekeeper.UpstreamID, *gatekeeper.Backend) *gatekeeper.Error
	RemoveBackend(gatekeeper.BackendID) *gatekeeper.Error

	// NOTE this method is not accessible to the plugin implemeneter, it's
	// used internally to ensure that the plugin is still connected back to
	// the parent process via RPC.
	Heartbeat() *gatekeeper.Error
}

// ManagerClient is a client which wraps a ManagerRPC implementing type and
// exposes it in a friendly way, abstracting the concrete error types. This is
// passed along to the client itself, allowing it to make seamlessly make
// requests back into the parent process over RPC.
type ManagerClient struct {
	ManagerRPC ManagerRPC
}

func (m *ManagerClient) AddUpstream(upstream *gatekeeper.Upstream) error {
	return gatekeeper.ErrorToError(m.ManagerRPC.AddUpstream(upstream))
}

func (m *ManagerClient) RemoveUpstream(upstreamID gatekeeper.UpstreamID) error {
	return gatekeeper.ErrorToError(m.ManagerRPC.RemoveUpstream(upstreamID))
}

func (m *ManagerClient) AddBackend(upstreamID gatekeeper.UpstreamID, backend *gatekeeper.Backend) error {
	return gatekeeper.ErrorToError(m.ManagerRPC.AddBackend(upstreamID, backend))
}

func (m *ManagerClient) RemoveBackend(backendID gatekeeper.BackendID) error {
	return gatekeeper.ErrorToError(m.ManagerRPC.RemoveBackend(backendID))
}
