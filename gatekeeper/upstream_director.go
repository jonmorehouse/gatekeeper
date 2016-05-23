package gatekeeper

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

var UpstreamIDLock sync.RWMutex
var UpstreamIDCounter uint = 0

// build a unique upstreamID, ensuring that no more than a single upstream is
// created at once to ensure no conflicts occur between
func NewUpstreamID() upstream.UpstreamID {
	UpstreamIDLock.Lock()
	defer UpstreamIDLock.Unlock()

	UpstreamIDCounter += 1
	return upstream.UpstreamID(fmt.Sprintf("%d", UpstreamIDCounter))
}

var BackendIDLock sync.RWMutex
var BackendIDCounter uint = 0

func NewBackendID() upstream.BackendID {
	BackendIDLock.Lock()
	defer BackendIDLock.Unlock()

	BackendIDCounter += 1
	return upstream.BackendID(fmt.Sprintf("%d", BackendIDCounter))
}

// this manages multiple plugins and ensures that all listeners all receive a copy of
type UpstreamDirector struct {
	managers []InternalUpstreamManager
}

func (u *UpstreamDirector) AddManager(m InternalUpstreamManager) error {
	u.managers = append(u.managers, m)

	return nil
}

func (u *UpstreamDirector) UpstreamForRequest(req *http.Request) (upstream.Upstream, error) {
	// find the upstream for a request ...
	return upstream.Upstream{}, nil
}
