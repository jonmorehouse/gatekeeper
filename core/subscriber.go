package core

import (
	"sync"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type Subscriber interface {
	startStopper

	AddUpstreamEventHook(gatekeeper.Event, func(*UpstreamEvent)) error
}

func NewSubscriber(broadcaster Broadcaster) Subscriber {
	return &subscriber{
		hooks:       make(map[gatekeeper.Event][]func(*UpstreamEvent)),
		eventCh:     make(EventCh),
		broadcaster: broadcaster,
	}
}

type subscriber struct {
	hooks       map[gatekeeper.Event][]func(*UpstreamEvent)
	broadcaster Broadcaster

	eventCh EventCh
	doneCh  chan error
	stopCh  chan struct{}

	sync.RWMutex
}

func (s *subscriber) Start() error {
	go s.worker()
	return nil
}

func (s *subscriber) Stop(time.Duration) error {
	s.stopCh <- struct{}{}
	return <-s.doneCh
}

func (s *subscriber) AddUpstreamEventHook(event gatekeeper.Event, hook func(*UpstreamEvent)) error {
	// make sure that the event type is of an actual upstream event ...
	if _, ok := map[gatekeeper.Event]struct{}{
		gatekeeper.UpstreamAddedEvent:   struct{}{},
		gatekeeper.UpstreamRemovedEvent: struct{}{},
		gatekeeper.BackendAddedEvent:    struct{}{},
		gatekeeper.BackendRemovedEvent:  struct{}{},
	}[event]; !ok {
		return InvalidEventErr
	}

	s.Lock()
	defer s.Unlock()
	s.hooks[event] = append(s.hooks[event], hook)
	return nil
}

func (s *subscriber) worker() {
	errs := NewMultiError()
	var wg sync.WaitGroup

	// TODO update this code when listeners for non-upstream events are added
	ch := make(EventCh, 5)
	listenerID := s.broadcaster.AddListener(ch, []gatekeeper.Event{
		gatekeeper.UpstreamAddedEvent,
		gatekeeper.UpstreamRemovedEvent,
		gatekeeper.BackendAddedEvent,
		gatekeeper.BackendRemovedEvent,
	})

	// handle an event, emitting it to all of its hooks
	handler := func(event Event) {
		if event == nil {
			return
		}

		upstreamEvent, ok := event.(*UpstreamEvent)
		if !ok {
			errs.Add(InvalidEventError)
			return
		}

		s.RLock()
		hooks, ok := s.hooks[event.Type()]
		s.RUnlock()
		if !ok {
			return
		}

		wg.Add(len(hooks))
		for _, hook := range hooks {
			go func(hook func(*UpstreamEvent)) {
				defer wg.Done()
				hook(upstreamEvent)
			}(hook)
		}
	}

	var stopped bool
	for {
		select {
		case <-s.stopCh:
			break
		case event, _ := <-s.eventCh:
			handler(event)
		}

		if stopped {
			s.broadcaster.RemoveListener(listenerID)
			close(s.eventCh)
		}
	}

	wg.Wait()
	s.doneCh <- errs.ToErr()
}
