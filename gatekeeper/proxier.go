package gatekeeper

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/jonmorehouse/gatekeeper/shared"
)

type Proxier interface {
	Proxy(http.ResponseWriter, *http.Request, *shared.Request, *shared.Backend) error
}

type proxier struct{}

func NewProxier() Proxier {
	return proxier{}
}

func (p proxier) Proxy(rw http.ResponseWriter, rawReq *http.Request, req *shared.Request, backend *shared.Backend) error {
	// first, we build out a new request to be proxied
	p.modifyRawRequest(rawReq, req)

	// build out the backend proxier
	backendURL, err := url.Parse(backend.Address)
	if err != nil {
		return err
	}
	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	proxy.ServeHTTP(rw, rawReq)
	return nil
}

func (p proxier) modifyRawRequest(rawReq *http.Request, req *shared.Request) {
	if req.UpstreamMatchType == shared.PrefixMatch {
		rawReq.URL.Path = req.PrefixlessPath
	} else {
		rawReq.URL.Path = req.Path
	}

	rawReq.Header = req.Header
}
