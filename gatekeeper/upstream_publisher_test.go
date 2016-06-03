package gatekeeper

import (
	"testing"

	"github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

type testEventBroadcaster struct {
	t *testing.T
	EventBroadcaster

	publishCalls      []Event
	publishSideEffect func()
	publishReturn     error
}

func (eb *testEventBroadcaster) Publish(event Event) error {
	eb.publishCalls = append(eb.publishCalls, event)
	if eb.publishSideEffect != nil {
		eb.publishSideEffect()
	}
	return eb.publishReturn
}

func NewTestEventBroadcaster() EventBroadcaster {
	return &testEventBroadcaster{
		publishCalls: make([]Event, 0),
	}
}

func TestUpstreamManager__AddUpstream__Ok(t *testing.T) {
	b := NewTestEventBroadcaster()
	m := &UpstreamPluginPublisher{
		broadcaster:    b,
		knownUpstreams: make(map[upstream.UpstreamID]interface{}),
		knownBackends:  make(map[upstream.BackendID]interface{}),
	}

	upstr := upstream.Upstream{}
	id, err := m.AddUpstream(upstr)
	if err != nil {
		t.Fatalf("Invalid response for addUpstream")
	}
	if b.(*testEventBroadcaster).publishCalls[0].(UpstreamEvent).UpstreamID != UpstreamID(id) {
		t.Fatalf("Invalid call to publish")
	}
	if b.(*testEventBroadcaster).publishCalls[0].(UpstreamEvent).Upstream == nil {
		t.Fatalf("Nil Upstream in UpstreamEvent")
	}
}
