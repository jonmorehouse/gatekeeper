package upstream

import (
	"github.com/jonmorehouse/gatekeeper/shared"
)

// manager is the interface which clients use to talk back to the gatekeeper parent process
type Manager interface {
	AddUpstream(shared.Upstream) *shared.Error
	RemoveUpstream(shared.UpstreamID) *shared.Error

	AddBackend(shared.UpstreamID, shared.Backend) *shared.Error
	RemoveBackend(shared.BackendID) *shared.Error
}
