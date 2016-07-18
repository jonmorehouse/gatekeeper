package gatekeeper

import "sync"

type UpstreamContainer interface {
	AddUpstream(*Upstream) error
	RemoveUpstream(UpstreamID) error

	// query methods
	UpstreamByHostname(string) (*Upstream, error)
	UpstreamByPrefix(string) (*Upstream, error)
	UpstreamByName(string) (*Upstream, error)

	FetchAllUpstreams() []*Upstream
}

func NewUpstreamContainer() UpstreamContainer {
	return &upstreamContainer{
		upstreams: make(map[UpstreamID]*Upstream),
	}
}

type upstreamContainer struct {
	upstreams map[UpstreamID]*Upstream

	sync.RWMutex
}

func (u *upstreamContainer) AddUpstream(upstream *Upstream) error {
	u.Lock()
	defer u.Unlock()
	u.upstreams[upstream.ID] = upstream
	return nil
}

func (u *upstreamContainer) RemoveUpstream(uID UpstreamID) error {
	u.Lock()
	defer u.Unlock()
	delete(u.upstreams, uID)
	return nil
}

func (u *upstreamContainer) UpstreamByHostname(hostname string) (*Upstream, error) {
	u.RLock()
	defer u.RUnlock()

	for _, upstream := range u.upstreams {
		for _, hn := range upstream.Hostnames {
			if hn == hostname {
				return upstream, nil
			}
		}
	}

	return nil, UpstreamNotFoundErr
}

func (u *upstreamContainer) UpstreamByPrefix(prefix string) (*Upstream, error) {
	u.RLock()
	defer u.RUnlock()

	for _, upstream := range u.upstreams {
		for _, pre := range upstream.Prefixes {
			if prefix == pre {
				return upstream, nil
			}
		}
	}

	return nil, UpstreamNotFoundErr
}

func (u *upstreamContainer) UpstreamByName(name string) (*Upstream, error) {
	u.RLock()
	defer u.RUnlock()

	for _, upstream := range u.upstreams {
		if name == upstream.Name {
			return upstream, nil
		}
	}

	return nil, UpstreamNotFoundErr
}

func (u *upstreamContainer) FetchAllUpstreams() []*Upstream {
	u.RLock()
	defer u.RUnlock()

	upstreams := make([]*Upstream, len(u.upstreams))
	ctr := 0
	for _, upstream := range u.upstreams {
		upstreams[ctr] = upstream
		ctr += 1
	}

	return upstreams
}
