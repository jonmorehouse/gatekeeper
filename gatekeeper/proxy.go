package gatekeeper

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// this proxies a request to a backend
type BackendProxy struct{}

func NewBackendProxy(backend url.URL) *BackendProxy {
	return &BackendProxy{}
}

func (b *BackendProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	reverseProxy := &httputil.ReverseProxy{
		Director: func(_ *http.Request) {},
	}
	fmt.Println("started proxying...")
	reverseProxy.ServeHTTP(rw, req)
	fmt.Println("proxying completed...")
}
