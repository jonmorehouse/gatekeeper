package gatekeeper

import "sync"

type UpstreamContainer interface {
	AddUpstream(*Upstream) error
	RemoveUpstream(UpstreamID) error
	RemoveAllUpstreams() error

	// query methods
	UpstreamByHostname(string) (*Upstream, error)
	UpstreamByPrefix(string) (*Upstream, error)
	UpstreamByName(string) (*Upstream, error)
	UpstreamByID(UpstreamID) (*Upstream, error)

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

func (u *upstreamContainer) UpstreamByID(id UpstreamID) (*Upstream, error) {
	u.RLock()
	defer u.RUnlock()

	upstream, ok := u.upstreams[id]
	if !ok {
		return nil, UpstreamNotFoundErr
	}

	return upstream, nil
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
