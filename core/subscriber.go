package subscriber

import (
	"sync"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type Subscriber interface {
	StartStopper

	AddUpstreamEventHook(gatekeeper.Event, func(*UpstreamEvent))
}

func NewSubscriber(broadcaster Broadcaster) Subscriber {
	return &subscriber{
		hooks:       make(map[gatekeeper.Event][]func(*UpstreamEvent)),
		eventCh:     make(EventCh),
		listenerIDs: make(map[gatekeeper.Event]ListenerID),
		broadcaster: broadcaster,
	}
}

type subscriber struct {
	hooks map[gatekeeper.Event][]func(*UpstreamEvent)

	eventCh EventCh
	doneCh  chan error
	stopCh  chan struct{}

	sync.RWMutex
}

func (s *subscriber) Start() error {
	go s.worker()
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

	r.Lock()
	defer r.Unlock()
	s.hooks[event] = append(s.hooks[event], hook)
	return nil
}

func (s *subscriber) worker() {
	errs := NewMultiError()
	var wg sync.WaitGroup

	// TODO update this code when listeners for non-upstream events are added
	ch := make(EventCh, 5)
	listenerID := s.broadcaster.AddListener(ch, []gatekeeper.EventType{
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

	for {
		select {
		case <-stopCh:
			break
		case event, _ := <-eventCh:
			handler(event)
		}

		if stopped {
			if err := s.broadcaster.RemoveListener(listenerID); err != nil {
				errs.Add(err)
			}
			close(eventCh)
		}
	}

	wg.Wait()
	s.doneCh <- errs.ToErr()
}
