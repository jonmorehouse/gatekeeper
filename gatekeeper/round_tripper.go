package gatekeeper

import (
	"net/http"
	"time"
)

// RoundTripper is a timeout based http.RoundTripper client which passes the
// response, duration and any raised errors to the responseHook.
type roundTripper struct {
	responseHook func(*http.Response, time.Duration, error) (*http.Response, error)
	timeout      time.Duration
}

func NewRoundTripper(timeout time.Duration, responseHook func(*http.Response, time.Duration, error) (*http.Response, error)) http.RoundTripper {
	return &roundTripper{
		timeout:      timeout,
		responseHook: responseHook,
	}
}

func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	doneCh := make(chan struct{})
	cancelCh := make(chan struct{})

	var err error
	var resp *http.Response

	startTS := time.Now()
	go func() {
		resp, err = http.DefaultTransport.RoundTrip(req)
		doneCh <- struct{}{}
	}()

	select {
	case <-doneCh:
		break
		close(cancelCh)
	case <-time.After(r.timeout):
		err = ProxyTimeoutError
		close(cancelCh)
	}

	latency := time.Now().Sub(startTS)
	return r.responseHook(resp, latency, err)
}
