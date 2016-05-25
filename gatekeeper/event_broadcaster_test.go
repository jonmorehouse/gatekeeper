package gatekeeper

import (
	"sync"
	"testing"
	"time"
)

type testEvent struct{}

func (e *testEvent) Type() EventType {
	return EventType(1)
}

func TestUpstreamEventBroadcaster__Publish__Ok(t *testing.T) {
	// make sure that publishing to an event broadcaster ensures we get the correct message.
	br := NewUpstreamEventBroadcaster()
	ch := make(EventCh)
	br.AddListener(ch, []EventType{EventType(1)})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		timeout := time.Now().Add(time.Second)
		for {
			select {
			case <-ch:
				wg.Done()
			default:
				if time.Now().After(timeout) {
					t.Fatalf("Event never came")
				}
			}
		}
	}()

	br.Publish(&testEvent{})
	wg.Wait()
}

func TestUpstreamEventBroadcast__Publish__OkWithUnsubscribed(t *testing.T) {
	br := NewUpstreamEventBroadcaster()
	ch1 := make(EventCh)
	ch2 := make(EventCh)

	id1, err := br.AddListener(ch1, []EventType{EventType(1)})
	if err != nil {
		t.Fatalf("AddListener failed...")
	}
	_, err = br.AddListener(ch2, []EventType{EventType(1)})
	if err != nil {
		t.Fatalf("AddListener failed...")
	}
	br.RemoveListener(id1)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		timeout := time.Now().Add(time.Second)
		for {
			select {
			case <-ch1:
				t.Fatalf("ch1 received an event after unsubscribing")
			case <-ch2:
				wg.Done()
			default:
				if time.Now().After(timeout) {
					t.Fatalf("Event never came for ch2")
				}
			}
		}
	}()

	br.Publish(&testEvent{})
	wg.Wait()
}
