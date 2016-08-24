package core

import (
	"sync"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type Subscriber interface {
	starter
	stopper

	AddUpstreamEventHook(gatekeeper.Event, func(*UpstreamEvent)) error
}

func NewSubscriber(broadcaster Broadcaster) Subscriber {
	return &subscriber{
		hooks:	     make(map[gatekeeper.Event][]func(*UpstreamEvent)),
		broadcaster: broadcaster,
		stopCh:      make(chan struct{}, 1),
		doneCh:      make(chan error, 1),
	}
}

type subscriber struct {
	hooks	    map[gatekeeper.Event][]func(*UpstreamEvent)
	broadcaster Broadcaster

	doneCh chan error
	stopCh chan struct{}

	RWMutex
}

func (s *subscriber) Start() error {
	s.worker()
	return nil
}

func (s *subscriber) Stop() error {
	s.stopCh <- struct{}{}
	return <-s.doneCh
}

func (s *subscriber) AddUpstreamEventHook(event gatekeeper.Event, hook func(*UpstreamEvent)) error {
	// make sure that the event type is of an actual upstream event ...
	if _, ok := map[gatekeeper.Event]struct{}{
		gatekeeper.UpstreamAddedEvent:	 struct{}{},
		gatekeeper.UpstreamRemovedEvent: struct{}{},
		gatekeeper.BackendAddedEvent:	 struct{}{},
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
	eventCh := make(EventCh, 5)
	listenerID := s.broadcaster.AddListener(eventCh, []gatekeeper.Event{
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

	go func() {
		for {
			select {
			case <-s.stopCh:
				goto stop
			case event, _ := <-eventCh:
				handler(event)
			default:
			}
		}

	stop:
		s.broadcaster.RemoveListener(listenerID)
		close(eventCh)
		// finish any outstanding hooks
		wg.Wait()
		s.doneCh <- errs.ToErr()
	}()
}
