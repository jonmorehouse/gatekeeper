package core

import (
	"log"
	"sync"
	"time"
)

type HookManager interface {
	startStopper
	AddHook(time.Duration, func() error)
}

func NewHookManager() HookManager {
	return &hookManager{}
}

type hook struct {
	cb       func() error
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
}

type hookManager struct {
	hooks []*hook

	sync.RWMutex
}

func (m *hookManager) Start() error { return nil }
func (m *hookManager) Stop(time.Duration) error {
	var wg sync.WaitGroup

	for _, h := range m.hooks {
		wg.Add(1)
		go func(h *hook) {
			defer wg.Done()
			h.stopCh <- struct{}{}
			<-h.doneCh
		}(h)
	}

	wg.Wait()
	return nil
}

func (m *hookManager) AddHook(interval time.Duration, cb func() error) {
	m.Lock()
	defer m.Unlock()
	h := &hook{
		cb:       cb,
		interval: interval,
		stopCh:   make(chan struct{}, 1),
		doneCh:   make(chan struct{}, 1),
	}
	m.hooks = append(m.hooks, h)

	go func(h *hook) {
		ticker := time.NewTicker(h.interval)

		for {
			stopped := false

			select {
			case <-ticker.C:
				if err := h.cb(); err != nil {
					log.Println(err)
				}
			case <-h.stopCh:
				ticker.Stop()
				close(h.stopCh)
				stopped = true
			}

			if stopped {
				break
			}
		}

		h.doneCh <- struct{}{}
	}(h)
}
