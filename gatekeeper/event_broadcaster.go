package gatekeeper

import (
	"fmt"
	"sync"

	"github.com/jonmorehouse/gatekeeper/shared"
)

type EventType uint

const (
	UpstreamAdded EventType = iota + 1
	UpstreamRemoved
	BackendAdded
	BackendRemoved
)

type Event interface {
	Type() EventType
}

type EventCh chan Event

type EventListenerID uint

var NilEventListenerID EventListenerID = 0

type EventBroadcaster interface {
	// AddListener accepts a channel and ensures that it receives all
	// messages of the types passed in.
	AddListener(EventCh, []EventType) (EventListenerID, error)

	// RemoveListener accepts a ListenerID and will remove it from
	// receiving messages. This shouldn't handle closing the channel,
	// however.
	RemoveListener(EventListenerID) error

	// Publish is used to emit a message to any and all listeners for the
	// given type.
	Publish(Event) error
}

type UpstreamEvent struct {
	EventType  EventType
	Upstream   *shared.Upstream
	UpstreamID shared.UpstreamID
	Backend    *shared.Backend
	BackendID  shared.BackendID
}

func (u UpstreamEvent) Type() EventType {
	return u.EventType
}

// this is internal to the UpstreamEventBroadcaster
type upstreamEventBroadcasterChID struct {
	Ch EventCh
	ID EventListenerID
}

type UpstreamEventBroadcaster struct {
	// map the id to which eventtypes its listening on, this is for
	// deletion purposes so we don't have to iterate over the entirety of
	// the internal store.
	listenersToEventTypes map[EventListenerID][]EventType
	eventTypesToListeners map[EventType][]*upstreamEventBroadcasterChID

	// counter for the current eventListenerID
	eventListenerID EventListenerID
	sync.Mutex
}

func NewUpstreamEventBroadcaster() EventBroadcaster {
	return &UpstreamEventBroadcaster{
		listenersToEventTypes: make(map[EventListenerID][]EventType),
		eventTypesToListeners: make(map[EventType][]*upstreamEventBroadcasterChID),
		eventListenerID:       NilEventListenerID,
	}
}

func (b *UpstreamEventBroadcaster) AddListener(ch EventCh, types []EventType) (EventListenerID, error) {
	if len(types) == 0 {
		return NilEventListenerID, fmt.Errorf("Must specify at least one EventType to listen to")
	}

	// fetch the next eventListenerID
	b.Lock()
	defer b.Unlock()

	// update the internal eventListenerID counter and fetch the latest,
	// which happens to be the ID we're returning here
	b.eventListenerID += 1
	listenerID := b.eventListenerID

	// store the eventTypes that this listener is listening on, for
	// deletion purposes later on.
	b.listenersToEventTypes[listenerID] = types

	// update the mapping of each event type specified.
	chID := &upstreamEventBroadcasterChID{
		Ch: ch,
		ID: listenerID,
	}
	for _, eventType := range types {
		if _, ok := b.eventTypesToListeners[eventType]; !ok {
			b.eventTypesToListeners[eventType] = make([]*upstreamEventBroadcasterChID, 0)
		}
		b.eventTypesToListeners[eventType] = append(b.eventTypesToListeners[eventType], chID)
	}

	return listenerID, nil
}

func (b *UpstreamEventBroadcaster) RemoveListener(id EventListenerID) error {
	if _, ok := b.listenersToEventTypes[id]; !ok {
		return fmt.Errorf("invalid EventListenerID")
	}
	b.Lock()
	defer b.Unlock()

	for _, eventType := range b.listenersToEventTypes[id] {
		if _, ok := b.eventTypesToListeners[eventType]; !ok {
			return fmt.Errorf("Internal error; did not store event types correctly")
		}

		listeners := b.eventTypesToListeners[eventType]
		for index, listener := range listeners {
			if listener.ID != id {
				continue
			}

			b.eventTypesToListeners[eventType] = append(listeners[:index], listeners[index+1:]...)
		}
	}
	delete(b.listenersToEventTypes, id)
	return nil
}

func (b *UpstreamEventBroadcaster) Publish(event Event) error {
	receivers, ok := b.eventTypesToListeners[event.Type()]
	if !ok || len(receivers) == 0 {
		return fmt.Errorf("No receivers for published message ...")
	}

	// emit messages to all receivers asynchronously
	go func() {
		for _, receiver := range receivers {
			receiver.Ch <- event
		}
	}()
	return nil
}
