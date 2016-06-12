package gatekeeper

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
)

type Proxier interface {
	Proxy(http.ResponseWriter, *http.Request, *shared.Request, *shared.Backend) error
}

type proxier struct {
	defaultTimeout time.Duration

	responseModifier ResponseModifier
	sync.RWMutex

	requests map[*http.Request]*shared.Request
}

func NewProxier(responseModifier ResponseModifier, defaultTimeout time.Duration) Proxier {
	return &proxier{
		defaultTimeout:   defaultTimeout,
		responseModifier: responseModifier,
		requests:         make(map[*http.Request]*shared.Request),
	}
}

func (p *proxier) Proxy(rw http.ResponseWriter, rawReq *http.Request, req *shared.Request, backend *shared.Backend) error {
	// first, we build out a new request to be proxied
	p.modifyRawRequest(rawReq, req)

	// build out the backend proxier
	backendURL, err := url.Parse(backend.Address)
	if err != nil {
		return err
	}

	// NOTE, we could probably cache these proxies by backend, but
	// underneath the hood, the transport is going to reuse connections
	// where possible as well.
	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	originalDirector := proxy.Director
	// NOTE this is a copy of the default director, but expands upon it to
	// cache the gatekeeper request
	proxy.Director = func(proxyReq *http.Request) {
		originalDirector(proxyReq)

		// here, we save the request so that when the response is intercepted
		// we have access to the "gatekeeper" request which is RPC compatible
		// and can be passed along to our plugins for modification.
		p.Lock()
		defer p.Unlock()
		p.requests[proxyReq] = req
	}

	// NOTE we respect request timeouts by fetching the upstream timeout and specifying a timeout

	// the proxier type, this local struct acts as the actual Proxier,
	// simply relying upon the default round trip var to make the requests
	proxy.Transport = p

	proxy.ServeHTTP(rw, rawReq)
	return nil
}

func (p *proxier) modifyRawRequest(rawReq *http.Request, req *shared.Request) {
	if req.UpstreamMatchType == shared.PrefixMatch {
		rawReq.URL.Path = req.PrefixlessPath
	} else {
		rawReq.URL.Path = req.Path
	}

	rawReq.Header = req.Header
}

func (p *proxier) RoundTrip(rawReq *http.Request) (*http.Response, error) {
	request, found := p.requests[rawReq]
	if !found {
		return nil, fmt.Errorf("INTERNAL_ERROR_REQUEST_CACHE_PROBLEM")
	}

	var wg sync.WaitGroup
	var err error
	var rawResp *http.Response

	// proxy the request, respecting the specified _total_ timeout. For
	// now, this timeout doesn't differentiate between dial and response
	// lifecycle timeouts and is a "total" timeout.
	wg.Add(1)
	go func() {
		// perform the request using the default RoundTrip mechanism, creating
		// an RPC compatbile response with the rawResponse once finished
		rawResp, err = http.DefaultTransport.RoundTrip(rawReq)
		wg.Done()
	}()

	// in a goroutine, wait for the request to finish.
	doneCh := make(chan interface{})
	go func() {
		wg.Wait()
		doneCh <- struct{}{}
	}()

	// if there is actually a timeout set, then we respect it, otherwise
	// wait for the request to finish indefinitely. This of course, is
	// configurable.
	timeout := request.Upstream.Timeout
	if timeout == time.Duration(0) {
		timeout = p.defaultTimeout
	}

	if timeout > time.Duration(0) {
		func() {
			for {
				select {
				case <-doneCh:
					return
				case <-time.After(timeout):
					err = fmt.Errorf("TIMEOUT")
					return
				}
			}
		}()
	} else {
		wg.Wait()
	}

	if err != nil {
		return rawResp, err
	}

	resp := shared.NewResponse(rawResp)
	// modify the response using our responseModifier and then map the
	// response back to an _actual_ http.Response to be sent back to the
	// client.
	resp, err = p.responseModifier.ModifyResponse(request, resp)
	if err != nil {
		return rawResp, err
	}

	// transform the local response back to an http.Response
	rawResp.Status = resp.Status
	rawResp.StatusCode = resp.StatusCode
	rawResp.Proto = resp.Proto
	rawResp.ProtoMajor = resp.ProtoMajor
	rawResp.ProtoMinor = resp.ProtoMinor
	rawResp.Header = resp.Header
	rawResp.ContentLength = resp.ContentLength
	rawResp.TransferEncoding = resp.TransferEncoding
	rawResp.Close = resp.Close
	rawResp.Trailer = resp.Trailer

	// if the response plugin specified a response body, then we go ahead
	// and override with it in the actual response
	if resp.OverrideBody != nil {
		rawResp.Body = ioutil.NopCloser(bytes.NewReader(resp.OverrideBody))
	}
	return rawResp, err
}
