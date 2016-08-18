package utils

import (
	"sync"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type UpstreamContainer interface {
	AddUpstream(*gatekeeper.Upstream) error
	RemoveUpstream(gatekeeper.UpstreamID) error
	RemoveAllUpstreams() error

	// query methods
	UpstreamByHostname(string) (*gatekeeper.Upstream, error)
	UpstreamByPrefix(string) (*gatekeeper.Upstream, error)
	UpstreamByName(string) (*gatekeeper.Upstream, error)
	UpstreamByID(gatekeeper.UpstreamID) (*gatekeeper.Upstream, error)

	FetchAllUpstreams() []*gatekeeper.Upstream
}

func NewUpstreamContainer() UpstreamContainer {
	return &upstreamContainer{
		upstreams: make(map[gatekeeper.UpstreamID]*gatekeeper.Upstream),
	}
}

type upstreamContainer struct {
	upstreams map[gatekeeper.UpstreamID]*gatekeeper.Upstream

	sync.RWMutex
}

func (u *upstreamContainer) AddUpstream(upstream *gatekeeper.Upstream) error {
	u.Lock()
	defer u.Unlock()
	u.upstreams[upstream.ID] = upstream
	return nil
}

func (u *upstreamContainer) RemoveUpstream(uID gatekeeper.UpstreamID) error {
	_, err := u.UpstreamByID(uID)
	if err != nil {
		return err
	}

	u.Lock()
	defer u.Unlock()
	delete(u.upstreams, uID)
	return nil
}

func (u *upstreamContainer) RemoveAllUpstreams() error {
	var err error

	for _, upstream := range u.FetchAllUpstreams() {
		if e := u.RemoveUpstream(upstream.ID); e != nil {
			err = e
		}
	}
	return err
}

func (u *upstreamContainer) UpstreamByID(id gatekeeper.UpstreamID) (*gatekeeper.Upstream, error) {
	u.RLock()
	defer u.RUnlock()

	upstream, ok := u.upstreams[id]
	if !ok {
		return nil, gatekeeper.UpstreamNotFoundErr
	}

	return upstream, nil
}

func (u *upstreamContainer) UpstreamByHostname(hostname string) (*gatekeeper.Upstream, error) {
	u.RLock()
	defer u.RUnlock()

	for _, upstream := range u.upstreams {
		for _, hn := range upstream.Hostnames {
			if hn == hostname {
				return upstream, nil
			}
		}
	}

	return nil, gatekeeper.UpstreamNotFoundErr
}

func (u *upstreamContainer) UpstreamByPrefix(prefix string) (*gatekeeper.Upstream, error) {
	u.RLock()
	defer u.RUnlock()

	for _, upstream := range u.upstreams {
		for _, pre := range upstream.Prefixes {
			if prefix == pre {
				return upstream, nil
			}
		}
	}

	return nil, gatekeeper.UpstreamNotFoundErr
}

func (u *upstreamContainer) UpstreamByName(name string) (*gatekeeper.Upstream, error) {
	u.RLock()
	defer u.RUnlock()

	for _, upstream := range u.upstreams {
		if name == upstream.Name {
			return upstream, nil
		}
	}

	return nil, gatekeeper.UpstreamNotFoundErr
}

func (u *upstreamContainer) FetchAllUpstreams() []*gatekeeper.Upstream {
	u.RLock()
	defer u.RUnlock()

	upstreams := make([]*gatekeeper.Upstream, len(u.upstreams))
	ctr := 0
	for _, upstream := range u.upstreams {
		upstreams[ctr] = upstream
		ctr += 1
	}

	return upstreams
}
