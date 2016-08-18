package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	"github.com/jonmorehouse/gatekeeper/gatekeeper/utils"
	"github.com/julienschmidt/httprouter"
)

type httpError struct {
	msg  string
	code int
}

var (
	InternalErr = httpError{"INTERNAL_ERROR", 500}

	InvalidUpstreamParamsErr = httpError{"INVALID_UPSTREAM_PARAMS", 400}
	InvalidBackendParamsErr  = httpError{"INVALID_BACKEND_PARAMS", 400}

	UpstreamNotFoundErr   = httpError{"UPSTREAM_NOT_FOUND", 404}
	BackendNotFoundErr    = httpError{"BACKEND_NOT_FOUND", 404}
	UpstreamIDRequiredErr = httpError{"UPSTREAM_ID_REQUIRED", 400}
	BackendIDRequiredErr  = httpError{"BACKEND_ID_REQUIRED", 400}

	errMapping = map[error]httpError{
		gatekeeper.UpstreamNotFoundErr: UpstreamNotFoundErr,
		gatekeeper.BackendNotFoundErr:  BackendNotFoundErr,
	}
)

func (e httpError) Error() string {
	return e.msg
}

type errorResponse struct {
	Msg string `json:"message"`
}

// errorStatusCode attempts to find the best statusCode possible relating to
// the error. It looks for either a concrete type with a code attribute or an
// interface method which returns the code.
func errorStatusCode(err error) int {
	// if the error is unknown, default to 500
	code := 500

	// attempt to extract a response code out of the error
	if httpError, ok := err.(httpError); ok {
		code = httpError.code
	}

	return code
}

// writeJSONErrorResponse attempts to write out a JSON response from a given
// error. First, it attempts to match any known errors to errors that are known
// locally. Secondly, it will pass the error and its code into a json encoder
// actually writing the response to the response writer.
func writeJSONErrorResponse(rw http.ResponseWriter, err error) {
	mappedErr, ok := errMapping[err]
	if ok {
		err = mappedErr
	}

	writeJSONResponse(rw, errorStatusCode(err), &errorResponse{
		Msg: err.Error(),
	})
}

// writeJSONResponse attempts to write out a JSON response and status code to
// the given ResponseWriter by encoding the type as JSON directly to the
// writer.
func writeJSONResponse(rw http.ResponseWriter, code int, val interface{}) {
	rw.WriteHeader(code)
	json.NewEncoder(rw).Encode(val)
}

// responseWriter is a local type that allows for logging of a request by
// capturing information sent to the responseWriter along the way.
type ResponseWriter interface {
	http.ResponseWriter
	Code() int
}

func newResponseWriter(rw http.ResponseWriter) ResponseWriter {
	return &responseWriter{code: 200, ResponseWriter: rw}
}

type responseWriter struct {
	code int
	http.ResponseWriter
}

func (rw *responseWriter) Code() int {
	if rw.code == 0 {
		return 200
	}
	return rw.code
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.code = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(handler httprouter.Handle) httprouter.Handle {
	return func(hrw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		startTS := time.Now()
		rw := newResponseWriter(hrw)

		handler(hrw, req, params)

		msg := fmt.Sprintf("method=%s uri=%s duration=%v code=%d", req.Method, req.URL.RawPath, time.Now().Sub(startTS), rw.Code())
		log.Println(msg)
	}
}

// upstream is a JSON formatted representation of the gatekeeper.Upstream type
type upstream struct {
	ID        string                 `json:"upstream_id"`
	Name      string                 `json:"name"`
	Protocols []string               `json:"protocols"`
	Hostnames []string               `json:"hostnames"`
	Prefixes  []string               `json:"prefixes"`
	Timeout   time.Duration          `json:"timeout"`
	Extra     map[string]interface{} `json:"extra"`

	// backends
	Backends []*backend `json:"backends"`
}

// take a gatekeeper Upstream and return a serialized local Upstream that can be written out as JSON
func toUpstream(u *gatekeeper.Upstream) *upstream {
	protocols := make([]string, len(u.Protocols))
	for idx, protocol := range u.Protocols {
		protocols[idx] = protocol.String()
	}

	return &upstream{
		ID:        string(u.ID),
		Name:      u.Name,
		Protocols: protocols,
		Hostnames: u.Hostnames,
		Prefixes:  u.Prefixes,
		Timeout:   u.Timeout,
		Extra:     u.Extra,
	}
}

// parse a local *Upstream type from a post request and return the resolved gatekeeper types or an error
func parseUpstream(u *upstream) (*gatekeeper.Upstream, []*gatekeeper.Backend, error) {
	protocols, err := gatekeeper.NewProtocols(u.Protocols)
	if err != nil {
		return nil, nil, err
	}

	upstream := &gatekeeper.Upstream{
		ID:        gatekeeper.UpstreamID(u.ID),
		Name:      u.Name,
		Protocols: protocols,
		Hostnames: u.Hostnames,
		Prefixes:  u.Prefixes,
		Timeout:   u.Timeout,
		Extra:     u.Extra,
	}

	backends := make([]*gatekeeper.Backend, len(u.Backends))
	for idx, backend := range u.Backends {
		backends[idx] = parseBackend(backend)
	}

	return upstream, backends, nil
}

// serialize a gatekeeper backend into a local Backend
func toBackend(b *gatekeeper.Backend) *backend {
	return &backend{
		ID:      string(b.ID),
		Address: b.Address,
		Extra:   b.Extra,
	}
}

// parse a local backend into a gatekeeper.Backend
func parseBackend(b *backend) *gatekeeper.Backend {
	return &gatekeeper.Backend{
		ID:      gatekeeper.BackendID(b.ID),
		Address: b.Address,
		Extra:   b.Extra,
	}
}

// backend is a JSON formatted representation of the gatekeeper.Backend type
type backend struct {
	ID      string                 `json:"backend_id"`
	Address string                 `json:"address"`
	Extra   map[string]interface{} `json:"extra"`
}

func NewAPI(serviceContainer utils.ServiceContainer) utils.Service {
	return &api{
		serviceContainer: serviceContainer,
	}
}

type api struct {
	serviceContainer utils.ServiceContainer
}

// Router returns an http.Handler that can handle all traffic; routing it to the proper handlers
func (a *api) Router() http.Handler {
	router := httprouter.New()

	router.GET("/", loggingMiddleware(a.indexHandler))

	// manage upstreams collection
	router.GET("/upstreams", loggingMiddleware(a.fetchUpstreamsHandler))
	router.POST("/upstreams", loggingMiddleware(a.addUpstreamHandler))
	router.DELETE("/upstreams/:upstream_id", loggingMiddleware(a.removeUpstreamHandler))

	// manage upstream backend collections
	router.GET("/upstreams/:upstream_id", loggingMiddleware(a.fetchUpstreamHandler))
	router.POST("/upstreams/:upstream_id/backend", loggingMiddleware(a.addBackendHandler))
	// NOTE: an upstream_id is _not_ required to delete a backend; expose both urls for api-friendliness though!
	router.DELETE("/upstreams/:upstream_id/backend/:backend_id", loggingMiddleware(a.removeBackendHandler))
	router.DELETE("/backends/:backend_id", loggingMiddleware(a.removeBackendHandler))

	return router
}

func (a *api) indexHandler(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	writeJSONResponse(rw, 200, "OK")
}

// fetch all upstreams currently registered
func (a *api) fetchUpstreamsHandler(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	upstreams := a.serviceContainer.FetchAllUpstreams()

	formattedUpstreams := make([]*upstream, len(upstreams))
	for idx, u := range upstreams {
		formattedUpstreams[idx] = toUpstream(u)

		backends, err := a.serviceContainer.FetchBackends(u.ID)
		if err != nil {
			writeJSONErrorResponse(rw, InternalErr)
			return
		}

		formattedUpstreams[idx].Backends = make([]*backend, len(backends))
		for bidx, backend := range backends {
			formattedUpstreams[idx].Backends[bidx] = toBackend(backend)
		}
	}

	writeJSONResponse(rw, 200, formattedUpstreams)
}

// add an upstream
func (a *api) addUpstreamHandler(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	// parse out upstream from the json request body
	decoder := json.NewDecoder(req.Body)
	var rawUpstream upstream
	if err := decoder.Decode(&rawUpstream); err != nil {
		writeJSONErrorResponse(rw, InvalidUpstreamParamsErr)
		return
	}

	// parse the upstream and its backends
	upstream, backends, err := parseUpstream(&rawUpstream)
	if err != nil {
		writeJSONErrorResponse(rw, InvalidUpstreamParamsErr)
		return
	}

	// persist the upstream/backends to the service container
	if err := a.serviceContainer.AddUpstream(upstream); err != nil {
		writeJSONErrorResponse(rw, err)
		return
	}
	for _, backend := range backends {
		if err := a.serviceContainer.AddBackend(upstream.ID, backend); err != nil {
			writeJSONErrorResponse(rw, err)
			return
		}
	}

	writeJSONResponse(rw, 201, "OK")
}

// remove an upstream and all of its backends
func (a *api) removeUpstreamHandler(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
	upstreamID := params.ByName("upstream_id")
	if upstreamID == "" {
		writeJSONErrorResponse(rw, UpstreamIDRequiredErr)
		return
	}

	if err := a.serviceContainer.RemoveUpstream(gatekeeper.UpstreamID(upstreamID)); err != nil {
		writeJSONErrorResponse(rw, err)
		return
	}

	writeJSONResponse(rw, 204, "DELETED")
}

// fetch an upstream and its backends.
func (a *api) fetchUpstreamHandler(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
	upstreamID := params.ByName("upstream_id")
	if upstreamID == "" {
		writeJSONErrorResponse(rw, UpstreamIDRequiredErr)
		return
	}

	log.Println(upstreamID)
	// fetch the upstream and its backends out of the container
	upstream, err := a.serviceContainer.UpstreamByID(gatekeeper.UpstreamID(upstreamID))
	if err != nil {
		writeJSONErrorResponse(rw, err)
		return
	}

	backends, err := a.serviceContainer.FetchBackends(gatekeeper.UpstreamID(upstreamID))
	if err != nil {
		writeJSONErrorResponse(rw, err)
		return
	}

	// build out the response objects
	respUpstream := toUpstream(upstream)
	respUpstream.Backends = make([]*backend, len(backends))
	for idx, backend := range backends {
		respUpstream.Backends[idx] = toBackend(backend)
	}

	writeJSONResponse(rw, 200, respUpstream)
}

// add a backend for an upstream
func (a *api) addBackendHandler(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
	upstreamID := params.ByName("upstream_id")
	if upstreamID == "" {
		writeJSONErrorResponse(rw, UpstreamIDRequiredErr)
		return
	}

	// parse the backend from the request
	var backend backend
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&backend); err != nil {
		writeJSONErrorResponse(rw, InvalidBackendParamsErr)
		return
	}

	// save the backend to the service container
	if err := a.serviceContainer.AddBackend(gatekeeper.UpstreamID(upstreamID), parseBackend(&backend)); err != nil {
		writeJSONErrorResponse(rw, err)
		return
	}

	writeJSONResponse(rw, 201, "OK")
}

// removeBackendHandler removes a backend from an individual upstream
func (a *api) removeBackendHandler(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
	backendID := params.ByName("backend_id")
	if backendID == "" {
		writeJSONErrorResponse(rw, BackendIDRequiredErr)
		return
	}

	if err := a.serviceContainer.RemoveBackend(gatekeeper.BackendID(backendID)); err != nil {
		writeJSONErrorResponse(rw, err)
	}

	writeJSONResponse(rw, 204, "DELETED")
}
