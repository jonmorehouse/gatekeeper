package gatekeeper

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
)

type Proxier interface {
	Proxy(http.ResponseWriter, *http.Request, *shared.Request, *shared.Upstream, *shared.Backend, *shared.RequestMetric) error
}

type proxier struct {
	defaultTimeout time.Duration

	modifier     Modifier
	metricWriter MetricWriterClient
}

func NewProxier(modifier Modifier, metricWriter MetricWriterClient) Proxier {
	return &proxier{
		modifier:       modifier,
		metricWriter:   metricWriter,
		defaultTimeout: 5 * time.Second,
	}
}

func (p *proxier) Proxy(rw http.ResponseWriter,
	httpReq *http.Request,
	req *shared.Request,
	upstream *shared.Upstream,
	backend *shared.Backend,
	metric *shared.RequestMetric) error {

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
		metric.ProxyLatency = latency
		if err != nil {
			metric.Error = err
			resp := shared.NewErrorResponse(500, err)
			metric.Response = resp
			return p.errorResponse(err, req, resp)
		}

		// modify the response with the modifier plugin and then copy
		// the response back into the httpResp object
		resp := shared.NewResponse(httpResp)
		resp, err = p.modifier.ModifyResponse(req, resp)
		if err != nil {
			resp = shared.NewErrorResponse(500, err)
			return p.errorResponse(err, req, resp)
		}

		p.responseToHTTPResponse(resp, httpResp)
		return httpResp, nil
	})

	proxy.ServeHTTP(rw, httpReq)
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
