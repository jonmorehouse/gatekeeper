package core

import "errors"

var AlreadyStartedErr = errors.New("already started error")

// SyncStartStopper exposes an interface for starting and stopping methods on a
// goroutine safe struct in a reliable way.
type SyncStartStopper interface {
	SyncStart(func() error) error
	SyncStop(func() error) error
	Started() bool
}

type syncStartStopper struct {
	started bool

	RWMutex
}

func (b *syncStartStopper) Started() bool {
	return b.started
}

func (b *syncStartStopper) SyncStart(cb func() error) error {
	b.RLock()
	started := b.started
	b.RUnlock()

	if started {
		return AlreadyStartedErr
	}

	if err := cb(); err != nil {
		return err
	}

	b.Lock()
	defer b.Unlock()
	b.started = true
	return nil
}

func (b *syncStartStopper) SyncStop(cb func() error) error {
	b.RLock()
	started := b.started
	b.RUnlock()

	if !started {
		return nil
	}

	b.Lock()
	defer func() {
		b.started = false
		b.Unlock()
	}()
	return cb()
}
