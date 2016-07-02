package gatekeeper

import "github.com/jonmorehouse/gatekeeper/shared"

type EventCh chan Event

// Event is an interface, that wraps a shared.Event with additional information
// for internal purposes. Specifically only _some_ types of data are accessible
// through an event and we tightly control that via the interface.
type Event interface {
	Type() shared.Event
}

type UpstreamEvent struct {
	Event shared.Event

	Upstream   *shared.Upstream
	UpstreamID shared.UpsteamID
	Backend    *shared.Backend
	BackendID  shared.BackendID
}

func (u *UpstreamEvent) UpstreamEvent() (*UpstreamEvent, error) {
	validEvents := map[shared.Event]struct{}{
		shared.UpstreamAddedEvent:   struct{}{},
		shared.UpstreamRemovedEvent: struct{}{},
		shared.BackendAddedEvent:    struct{}{},
		shared.BackendRemovedEvent:  struct{}{},
	}

	if _, ok := validEvents[u.Event]; !ok {
		return nil, InvalidEventError
	}

	return u, nil
}

type ListenerID string

type Broadcaster interface {
	// Add a listener accepting all events of this type on the input channel
	AddListener(EventCh, []shared.Event) ListenerID

	// RemoveListener accepts a ListenerID and will remove it from
	// receiving messages. This does nothing to close the channel
	RemoveListener(ListenerID)

	// Publish is used to emit a message to any and all listeners for the
	// given type.
	Publish(Event)
}

func NewBroadcaster() Broadcaster {
	return &broadcaster{
		eventListeners: make(map[shared.Event]map[ListenerID]EventCh),
	}
}

type broadcaster struct {
	eventListeners map[shared.Event]map[ListenerID]EventCh
}

func (b *broadcaster) AddListener(ch EventCh, events []shared.Event) ListenerID {
	listenerID := ListenerID(NewUUID())
	for _, event := range events {
		listeners, found := b.eventListeners[event]
		if !found {
			b.eventListeners[event] = make(map[ListenerID]EventCh, 1)
		}

		b.eventListeners[event][listenerID] = ch
	}

	return listenerID, nil
}

func (b *broadcaster) RemoveListener(id ListenerID) {
	for event, listeners := range b.eventListeners {
		delete(listeners, id)
	}
}

func (b *broadcaster) Publish(event Event) {
	listeners, ok := b.eventListeners[event.Type()]
	if !ok {
		return
	}

	for _, eventCh := range listeners {
		go func(EventCh) {
			eventCh <- event
		}(eventCh)

	}
}
