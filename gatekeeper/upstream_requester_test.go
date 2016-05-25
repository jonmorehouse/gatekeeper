package gatekeeper

import (
	"net/http"
	"testing"

	"github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

type addCall struct {
	ch    EventCh
	types []EventType
}

type upstreamRequesterBroadcaster struct {
	addCalls      []addCall
	addSideEffect func()
	addRespID     EventListenerID
	addRespError  error

	removeCalls      []EventListenerID
	removeSideEffect func()
	removeRespError  error

	EventBroadcaster
}

func NewUpstreamRequesterBroadcaster() *upstreamRequesterBroadcaster {
	return &upstreamRequesterBroadcaster{
		addCalls:    make([]addCall, 0),
		removeCalls: make([]EventListenerID, 0),
	}
}

func (u *upstreamRequesterBroadcaster) AddListener(eventCh EventCh, eventTypes []EventType) (EventListenerID, error) {
	u.addCalls = append(u.addCalls, addCall{eventCh, eventTypes})
	if u.addSideEffect != nil {
		u.addSideEffect()
	}
	return u.addRespID, u.addRespError
}

func (u *upstreamRequesterBroadcaster) RemoveListener(id EventListenerID) error {
	u.removeCalls = append(u.removeCalls, id)
	if u.removeSideEffect != nil {
		u.removeSideEffect()
	}
	return u.removeRespError
}

func TestAsyncUpstreamRequester__listener__addsUpstreamsFromEvents(t *testing.T) {

}

func TestAsyncUpstreamRequester__listener__removesUpstreamsFromEvents(t *testing.T) {

}

func TestAsyncUpstreamRequester__Start__addsListenerToBroadcaster(t *testing.T) {
	br := NewUpstreamRequesterBroadcaster()
	upstrReq := NewAsyncUpstreamRequester(br)

	upstrReq.Start()
	if len(br.addCalls) != 1 {
		t.Fatalf("Did not call start method...")
	}
}

func TestAsyncUpstreamRequester__Stop__removesListenerFromBroadcaster(t *testing.T) {
	br := NewUpstreamRequesterBroadcaster()
	upstrReq := NewAsyncUpstreamRequester(br)
	upstrReq.Start()
	upstrReq.Stop()
}

func TestAsyncUpstreamRequester__UpstreamForRequest__findByPrefixUncached(t *testing.T) {
	// build an upstreamRequester with a "test" hostname upstream
	br := NewUpstreamRequesterBroadcaster()
	upstrReq := NewAsyncUpstreamRequester(br)
	upstrReq.Start()
	event := UpstreamEvent{
		Upstream: &Upstream{
			Hostnames: []string{"test"},
		},
		UpstreamID: upstream.UpstreamID("aaa"),
		EventType:  UpstreamAdded,
	}
	upstrReq.(*AsyncUpstreamRequester).listenCh <- event

	req, _ := http.NewRequest("GET", "http://foo/bar", nil)
	req.Host = "test"

	respUpstr, err := upstrReq.UpstreamForRequest(req)
	if err != nil {
		t.Fatalf("Did not find upstream correctly")
	}

	if respUpstr != event.Upstream {
		t.Fatalf("Did not return correct upstream...")
	}
}

func TestAsyncUpstreamRequester__UpstreamForRequest__findByPrefixCached(t *testing.T) {

}

func TestAsyncUpstreamRequester__UpstreamForRequest__findByHostnameUncached(t *testing.T) {

}

func TestAsyncUpstreamRequester__UpstreamForRequest__findByHostnameCached(t *testing.T) {

}
