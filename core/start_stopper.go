package core

import (
	"errors"
	"sync"
	"time"
)

var AlreadyStartedErr = errors.New("already started error")

// startStopper represents the base start stopping
type startStopper interface {
	Start() error
	Stop(time.Duration) error
}

type baseStartStopper struct {
	started bool

	sync.RWMutex
}

func (b *baseStartStopper) syncStart(cb func() error) error {
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

func (b *baseStartStopper) syncStop(cb func() error) error {
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
