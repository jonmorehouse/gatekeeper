package gatekeeper

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/jonmorehouse/gatekeeper/shared"
)

func Retry(f func() error, retries uint) error {
	err := f()
	if err == nil {
		return nil
	}

	if retries == 0 {
		return err
	}
	return Retry(f, retries-1)
}

func ReqPrefix(req *http.Request) string {
	pieces := strings.Split(req.URL.Path, "/")
	if len(pieces) == 0 || len(pieces) == 1 {
		return ""
	}

	return pieces[1]
}

func PrepareBackendRequest(req *http.Request, upstream *shared.Upstream, backend shared.Backend) {
	prefix := ReqPrefix(req)
	for _, upstreamPrefix := range upstream.Prefixes {
		if prefix != upstreamPrefix {
			continue
		}

		req.URL.Path = "/" + strings.TrimPrefix(req.URL.Path, "/"+prefix+"/")
		break
	}

	backendURL, _ := url.Parse(backend.Address)
	req.URL.Host = backendURL.Host
	req.URL.Scheme = backendURL.Scheme
}
