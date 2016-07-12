package core

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type Proxier interface {
	Proxy(http.ResponseWriter, *http.Request, *gatekeeper.Request, *gatekeeper.Upstream, *gatekeeper.Backend, *gatekeeper.RequestMetric) error
}

type proxier struct {
	defaultTimeout time.Duration

	modifier     Modifier
	metricWriter MetricWriterClient
}

func NewProxier(modifier Modifier, metricWriter MetricWriterClient) Proxier {
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 200
	return &proxier{
		modifier:       modifier,
		metricWriter:   metricWriter,
		defaultTimeout: 5 * time.Second,
	}
}

func (p *proxier) Proxy(rw http.ResponseWriter,
	httpReq *http.Request,
	req *gatekeeper.Request,
	upstream *gatekeeper.Upstream,
	backend *gatekeeper.Backend,
	metric *gatekeeper.RequestMetric) error {

	backendAddress, err := url.Parse(backend.Address)
	if err != nil {
		return BackendAddressError
	}

	timeout := upstream.Timeout
	if timeout == time.Millisecond*0 {
		timeout = p.defaultTimeout
	}

	// build out the request and the proxy that will be used to perform the request
	p.modifyProxyRequest(httpReq, req)
	proxy := httputil.NewSingleHostReverseProxy(backendAddress)
	proxy.Transport = NewRoundTripper(timeout, func(httpResp *http.Response, latency time.Duration, err error) (*http.Response, error) {
		var startTS time.Time
		if httpResp == nil {
			httpResp = &http.Response{
				StatusCode: 500,
			}
		}

		resp := gatekeeper.NewResponse(httpResp)

		if err != nil {
			startTS = time.Now()
			resp, err = p.modifier.ModifyErrorResponse(err, req, resp)
			metric.ErrorResponseModifierLatency = time.Now().Sub(startTS)
			goto finished
		}

		// Attempt to modify the response
		startTS = time.Now()
		resp, err = p.modifier.ModifyResponse(req, resp)
		metric.ResponseModifierLatency = time.Now().Sub(startTS)
		if err != nil {
			startTS = time.Now()
			resp, err = p.modifier.ModifyErrorResponse(err, req, resp)
			metric.ErrorResponseModifierLatency = time.Now().Sub(startTS)
		}

	finished:
		metric.ProxyLatency = latency
		metric.Response = resp
		metric.Error = err
		resp.Error = gatekeeper.NewError(err)

		// in the rare case that a plugin failed and returned a nil
		// response, create a generic ErrorResponse denoting an
		// internal error
		if resp == nil {
			resp = gatekeeper.NewErrorResponse(500, InternalError)
		}

		p.responseToHTTPResponse(resp, httpResp)
		return httpResp, err
	})

	proxy.ServeHTTP(rw, httpReq)
	return nil
}

func (p *proxier) modifyProxyRequest(httpReq *http.Request, req *gatekeeper.Request) {
	if req.UpstreamMatchType == gatekeeper.PrefixUpstreamMatch {
		httpReq.URL.Path = req.PrefixlessPath
	} else {
		httpReq.URL.Path = req.Path
	}

	httpReq.Header = req.Header
}

func (p *proxier) responseToHTTPResponse(resp *gatekeeper.Response, httpResp *http.Response) {
	// this should basically never happen, but if it does, then assume that
	// that parent has handled it properly
	if resp == nil || httpResp == nil {
		return
	}

	// transform the local response back to an http.Response
	httpResp.Status = resp.Status
	httpResp.StatusCode = resp.StatusCode
	httpResp.Proto = resp.Proto
	httpResp.ProtoMajor = resp.ProtoMajor
	httpResp.ProtoMinor = resp.ProtoMinor
	httpResp.Header = resp.Header
	httpResp.ContentLength = resp.ContentLength
	httpResp.TransferEncoding = resp.TransferEncoding
	httpResp.Close = resp.Close
	httpResp.Trailer = resp.Trailer

	// if the plugin returns an override response, then go ahead and
	// consume the entirety of the httpResponse's reader, and close it to
	// prevent backing up the connection queue on the default transport.
	if resp.Body != nil {
		ioutil.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		httpResp.Body = ioutil.NopCloser(bytes.NewReader(resp.Body))
	}
}
