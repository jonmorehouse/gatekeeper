package gatekeeper

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
)

type Proxier interface {
	Proxy(http.ResponseWriter, *http.Request, *shared.Request, *shared.Backend, *shared.RequestMetric) error
}

type proxier struct {
	defaultTimeout time.Duration

	modifier     Modifier
	metricWriter MetricWriterClient
	sync.RWMutex

	requests map[*http.Request]*shared.Request
	metrics  map[*http.Request]*shared.RequestMetric
}

func NewProxier(modifier Modifier, metricWriter MetricWriterClient) Proxier {
	return &proxier{
		modifier:     modifier,
		requests:     make(map[*http.Request]*shared.Request),
		metrics:      make(map[*http.Request]*shared.RequestMetric),
		metricWriter: metricWriter,
	}
}

func (p *proxier) Proxy(rw http.ResponseWriter, httpReq *http.Request, req *shared.Request, backend *shared.Backend, metric *shared.RequestMetric) error {
	// first, we build out a new request to be proxied
	p.modifyProxyRequest(httpReq, req)

	// build out the backend proxier
	backendURL, err := url.Parse(backend.Address)
	if err != nil {
		return err
	}

	// NOTE, we could probably cache these proxies by backend, but
	// underneath the hood, the transport is going to reuse connections
	// where possible
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
		p.requests[proxyReq] = req
		p.metrics[proxyReq] = metric
		p.Unlock()
	}

	// the proxier type, this local struct acts as the actual Proxier,
	// simply relying upon the default round trip var to make the requests
	proxy.Transport = p

	// NOTE: move this latency timing down into the director / proxy level to remove more overhead
	startTS := time.Now()
	proxy.ServeHTTP(rw, httpReq)
	metric.ProxyLatency = time.Now().Sub(startTS)

	return nil
}

func (p *proxier) modifyProxyRequest(httpReq *http.Request, req *shared.Request) {
	if req.UpstreamMatchType == shared.PrefixUpstreamMatch {
		httpReq.URL.Path = req.PrefixlessPath
	} else {
		httpReq.URL.Path = req.Path
	}

	httpReq.Header = req.Header
}

func (p *proxier) RoundTrip(httpRequest *http.Request) (*http.Response, error) {
	p.Lock()
	metric, found := p.metrics[httpRequest]
	if !found {
		p.Unlock()
		response := shared.NewErrorResponse(500, InternalProxierError)
		metric.Response = response
		metric.Error = InternalProxierError
		return p.errorResponse(InternalProxierError, &shared.Request{}, response)
	}
	delete(p.metrics, httpRequest)

	request, found := p.requests[httpRequest]
	if !found {
		p.Unlock()
		response := shared.NewErrorResponse(500, InternalProxierError)
		metric.Error = InternalProxierError
		metric.Response = response
		return p.errorResponse(InternalProxierError, &shared.Request{}, response)
	}
	delete(p.requests, httpRequest)
	p.Unlock()

	var wg sync.WaitGroup
	var err error
	var httpResp *http.Response

	// set the cancelCh on the request so we can properly clean it up, even
	// after we've caught the timeout on our transport
	cancelCh := make(chan struct{})
	httpRequest.Cancel = cancelCh

	// proxy the request, respecting the specified _total_ timeout. For
	// now, this timeout doesn't differentiate between dial and response
	// lifecycle timeouts and is a "total" timeout.
	wg.Add(1)
	go func() {
		// perform the request using the default RoundTrip mechanism, creating
		// an RPC compatbile response with the httpResp once finished
		startTS := time.Now()
		httpResp, err = http.DefaultTransport.RoundTrip(httpRequest)
		metric.ProxyLatency = time.Now().Sub(startTS)
		wg.Done()
	}()

	// in a goroutine, wait for the request to finish.
	doneCh := make(chan struct{})
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
					close(cancelCh)
					err = ProxyTimeoutError
					return
				}
			}
		}()
	} else {
		wg.Wait()
	}

	if err != nil {
		response := shared.NewErrorResponse(500, ProxyTimeoutError)
		metric.Response = response
		metric.Error = ProxyTimeoutError
		return p.errorResponse(err, request, response)
	}

	// the response was successfully proxied, build a local response object
	// and modify it before returning to the caller
	resp := shared.NewResponse(httpResp)

	// if any err occurs modifying the request it most likely means that
	// either a.) something extenuating is happening and we can't
	// communicate over RPC or b.) the modifier emitted an error. If we
	// can't communicate over RPC, then we drop the response from the proxy
	// and write an internal error. If the modifier returned an error, then
	// we assume that they also would like to drop the response and
	// consider it an internal error.
	startTS := time.Now()
	resp, err = p.modifier.ModifyResponse(request, resp)
	metric.ResponseModifierLatency = time.Now().Sub(startTS)
	if err != nil {
		response := shared.NewErrorResponse(500, ModifierPluginError)
		metric.Response = response
		metric.Error = ModifierPluginError
		return p.errorResponse(err, request, response)
	}

	metric.Response = resp

	// by design, we never return an error here because we explicitly build our own error responses
	return httpResp, nil
}

func (p *proxier) responseToHTTPResponse(response *shared.Response, httpResp *http.Response) {
	// transform the local response back to an http.Response
	httpResp.Status = response.Status
	httpResp.StatusCode = response.StatusCode
	httpResp.Proto = response.Proto
	httpResp.ProtoMajor = response.ProtoMajor
	httpResp.ProtoMinor = response.ProtoMinor
	httpResp.Header = response.Header
	httpResp.ContentLength = response.ContentLength
	httpResp.TransferEncoding = response.TransferEncoding
	httpResp.Close = response.Close
	httpResp.Trailer = response.Trailer

	// if the response plugin specified a response body, then we go ahead
	// and override with it in the actual response
	if response.Body != nil {
		httpResp.Body = ioutil.NopCloser(bytes.NewReader(response.Body))
	}
}

func (p *proxier) errorResponse(err error, request *shared.Request, response *shared.Response) (*http.Response, error) {
	_, err = p.modifier.ModifyErrorResponse(err, request, response)
	if err != nil {
		log.Println("ModifierPlugin.ErrorResponse method failed: ", err)
	}

	httpResp := &http.Response{}
	p.responseToHTTPResponse(response, httpResp)
	return httpResp, nil
}
