package gatekeeper

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
)

type UpstreamRequester interface {
	Start() error
	Stop() error

	UpstreamForRequest(*http.Request) (*shared.Upstream, error)
}

type AsyncUpstreamRequester struct {
	broadcaster EventBroadcaster
	listenID    EventListenerID
	listenCh    EventCh
	stopCh      chan interface{}

	knownUpstreams      map[shared.UpstreamID]shared.Upstream
	upstreamsByHostname map[string]*shared.Upstream
	upstreamsByPrefix   map[string]*shared.Upstream
	sync.RWMutex
}

func NewAsyncUpstreamRequester(broadcaster EventBroadcaster) UpstreamRequester {
	return &AsyncUpstreamRequester{
		broadcaster: broadcaster,
		listenCh:    make(chan Event),
		stopCh:      make(chan interface{}),

		knownUpstreams:      make(map[shared.UpstreamID]shared.Upstream),
		upstreamsByHostname: make(map[string]*shared.Upstream),
		upstreamsByPrefix:   make(map[string]*shared.Upstream),
	}
}

func (r *AsyncUpstreamRequester) Start() error {
	id, err := r.broadcaster.AddListener(r.listenCh, []EventType{UpstreamAdded, UpstreamRemoved})
	if err != nil {
		return err
	}

	r.listenID = id
	go r.listener()

	return nil
}

func (r *AsyncUpstreamRequester) listener() {
	for {
		select {
		case rawEvent := <-r.listenCh:
			upstreamEvent, ok := rawEvent.(UpstreamEvent)
			if !ok {
				log.Fatal("Received an invalid UpstreamEvent...")
				continue
			}
			if upstreamEvent.Type() == UpstreamAdded {
				r.addUpstream(upstreamEvent)
			} else if upstreamEvent.Type() == UpstreamRemoved {
				r.addUpstream(upstreamEvent)
			} else {
				log.Fatal("Received an invalid upstream event...")
			}
		case <-r.stopCh:
			r.stopCh <- struct{}{}
			return
		default:
		}
	}
}

func (r *AsyncUpstreamRequester) addUpstream(event UpstreamEvent) {
	if event.UpstreamID == shared.NilUpstreamID {
		log.Fatal("Received an invalid upstream event...")
		return
	}
	r.Lock()
	defer r.Unlock()
	r.knownUpstreams[event.UpstreamID] = event.Upstream
}

func (r *AsyncUpstreamRequester) removeUpstream(event UpstreamEvent) {
	r.RLock()
	upstr, ok := r.knownUpstreams[event.UpstreamID]
	r.RUnlock()

	if !ok {
		log.Fatal("Attempted to remove upstream that was not present")
		return
	}

	r.Lock()
	defer r.Unlock()

	for _, hostname := range upstr.Hostnames {
		if _, ok := r.upstreamsByHostname[hostname]; ok {
			delete(r.upstreamsByHostname, hostname)
		}
	}

	for _, prefix := range upstr.Prefixes {
		if _, ok := r.upstreamsByPrefix[prefix]; ok {
			delete(r.upstreamsByPrefix, prefix)
		}
	}
	delete(r.knownUpstreams, event.UpstreamID)
}

func (r *AsyncUpstreamRequester) Stop() error {
	r.broadcaster.RemoveListener(r.listenID)
	r.listenID = NilEventListenerID
	r.stopCh <- struct{}{}

	timeout := time.Now().Add(time.Second)
	for {
		select {
		case <-r.stopCh:
			goto finish
		default:
			if time.Now().After(timeout) {
				return fmt.Errorf("Timed out waiting for worker goroutine to stop")
			}
		}
	}

finish:
	close(r.listenCh)
	close(r.stopCh)
	return nil
}

func (r *AsyncUpstreamRequester) UpstreamForRequest(req *http.Request) (*shared.Upstream, error) {
	r.Lock()
	defer r.Unlock()
	hostname := req.Host
	prefix := ReqPrefix(req)

	// check hostname cache
	if upstream, ok := r.upstreamsByHostname[hostname]; hostname != "" && ok {
		return upstream, nil
	}

	// check prefix cache
	if upstream, ok := r.upstreamsByPrefix[prefix]; prefix != "" && ok {
		return upstream, nil
	}

	// check all knownUpstreams, returning the first match
	for _, upstream := range r.knownUpstreams {
		if upstream.HasHostname(hostname) {
			r.upstreamsByHostname[hostname] = &upstream
			return &upstream, nil
		}
		if upstream.HasPrefix(prefix) {
			r.upstreamsByPrefix[prefix] = &upstream
			return &upstream, nil
		}
	}

	return nil, fmt.Errorf("No matching upstream for request...")
}
