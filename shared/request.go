package shared

import (
	"net/http"
	"strings"
)

type RequestError struct {
	Code uint
	Body string
}

func (e *RequestError) Error() string {
	return e.Body
}

func ReqPrefix(req *http.Request) string {
	pieces := strings.Split(req.URL.Path, "/")
	if len(pieces) == 0 || len(pieces) == 1 {
		return ""
	}

	return pieces[1]
}

// An internal representation of an *http.Request object which is RPC safe and
// understandable for request based routing.
type Request struct {
	Protocol Protocol
	Upstream *Upstream
	// the mechanism with which the upstream was matched
	UpstreamMatchType UpstreamMatchType

	// remote address of the caller
	RemoteAddr string
	Method     string

	// request.Host or url.Host depending upon which is set
	Host string

	// the first component of the URL, specifically the prefix
	Prefix         string
	PrefixlessPath string

	// url.URL.Path
	Path string
	// url.URL.RawPath
	RawPath string
	// url.URL.RawQuery
	RawQuery string
	// url.URL.Fragment
	Fragment string
	Header   map[string][]string

	// at any point of the request lifecycle, a RequestError will result in
	// an error response being sent back to the client.
	Err *RequestError
}

func NewRequest(req *http.Request, protocol Protocol) *Request {
	return &Request{
		Protocol:          protocol,
		Upstream:          nil,
		UpstreamMatchType: NilMatch,

		RemoteAddr: req.RemoteAddr,
		Method:     req.Method,

		Host: req.Host,

		// build out prefix and prefixless path
		Prefix:         ReqPrefix(req),
		PrefixlessPath: strings.TrimPrefix(req.URL.Path, "/"+ReqPrefix(req)),

		// copy over path components
		Path:     req.URL.Path,
		RawPath:  req.URL.RawPath,
		RawQuery: req.URL.RawQuery,
		Fragment: req.URL.Fragment,

		Header: req.Header,
		Err:    nil,
	}
}
