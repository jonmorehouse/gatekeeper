package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	"github.com/tylerb/graceful"
)

type API interface {
	Start() error
	Stop() error
}

type api struct {
	config  *Config
	manager ManagerClient

	server *graceful.Server

	errCh   chan error
	stopped bool
}

// formatted response for upstream and backend types
type formattedUpstream struct {
	ID        gatekeeper.UpstreamID `json:"id"`
	Name      string            `json:"name"`
	Protocols []string          `json:"protocols"`
	Hostnames []string          `json:"hostnames"`
	Prefixes  []string          `json:"prefixes"`
	Timeout   string            `json:"request_time"`
}

type formattedBackend struct {
	ID          gatekeeper.BackendID `json:"id"`
	Address     string           `json:"address"`
	Healthcheck string           `json:"healthcheck"`
}

// standard error response
type errorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type statusResponse struct {
	Status string `json:"status"`
}

// responses for GET only endpoints
type rootResponse struct {
	Status    string            `json:"status"`
	Endpoints map[string]string `json:"endpoints"`
}

type healthResponse struct {
	Status string `json:"status"`
}

// request/responses for POST/DELETE endpoints
type addUpstreamRequest struct {
	Name      string   `json:"name"`
	Protocols []string `json:"protocols"`
	Hostnames []string `json:"hostnames"`
	Prefixes  []string `json:"prefixes"`
	Timeout   string   `json:"request_timeout"`
}

type addUpstreamResponse struct {
	Status     string             `json:"status"`
	Upstream   *formattedUpstream `json:"upstream"`
	UpstreamID gatekeeper.UpstreamID  `json:"upstream_id"`
}

type addBackendRequest struct {
	UpstreamID  gatekeeper.UpstreamID `json:"upstream_id"`
	Address     string            `json:"address"`
	Healthcheck string            `json:"healthcheck"`
}

type addBackendResponse struct {
	Status     string             `json:"status"`
	BackendID  gatekeeper.BackendID   `json:"backend_id"`
	UpstreamID gatekeeper.UpstreamID  `json:"upstream_id"`
	Backend    *formattedBackend  `json:"upstream"`
	Upstream   *formattedUpstream `json:"upstream"`
}

type removeUpstreamRequest struct {
	UpstreamID gatekeeper.UpstreamID `json:"upstream_id"`
}

type removeBackendRequest struct {
	BackendID gatekeeper.BackendID `json:"backend_id"`
}

// helper methods for casting backend / upstreams between request/response and in-memory types
func newFormattedUpstream(upstream *gatekeeper.Upstream) *formattedUpstream {
	protocols := make([]string, 0, len(upstream.Protocols))
	for _, protocol := range upstream.Protocols {
		protocols = append(protocols, protocol.String())
	}

	return &formattedUpstream{
		ID:        upstream.ID,
		Name:      upstream.Name,
		Protocols: protocols,
		Prefixes:  upstream.Prefixes,
		Timeout:   upstream.Timeout.String(),
	}
}

func newUpstream(rawUpstream *addUpstreamRequest) (*gatekeeper.Upstream, error) {
	timeout, err := time.ParseDuration(rawUpstream.Timeout)
	if err != nil {
		return nil, err
	}

	protocolMappings := map[string]gatekeeper.Protocol{
		gatekeeper.HTTPPublic.String():   gatekeeper.HTTPPublic,
		gatekeeper.HTTPInternal.String(): gatekeeper.HTTPInternal,
	}
	protocols := make([]gatekeeper.Protocol, 0, len(rawUpstream.Protocols))
	for _, protocolString := range rawUpstream.Protocols {
		protocol, ok := protocolMappings[protocolString]
		if !ok {
			return nil, fmt.Errorf("Invalid protocol ", protocolString)
		}

		protocols = append(protocols, protocol)
	}

	return &gatekeeper.Upstream{
		ID:        gatekeeper.NilUpstreamID,
		Name:      rawUpstream.Name,
		Protocols: protocols,
		Hostnames: rawUpstream.Hostnames,
		Prefixes:  rawUpstream.Prefixes,
		Timeout:   timeout,
	}, nil
}

func newFormattedBackend(backend *gatekeeper.Backend) *formattedBackend {
	return &formattedBackend{
		ID:          backend.ID,
		Address:     backend.Address,
		Healthcheck: backend.Healthcheck,
	}
}

func newBackend(rawBackend *addBackendRequest) (*gatekeeper.Backend, error) {
	_, err := url.Parse(rawBackend.Address)
	if err != nil {
		return nil, err
	}

	return &gatekeeper.Backend{
		ID:          gatekeeper.NilBackendID,
		Address:     rawBackend.Address,
		Healthcheck: rawBackend.Healthcheck,
	}, nil
}

func NewAPI(config *Config, manager Manager) API {
	return &api{
		errCh:   make(chan error, 1),
		stopped: false,
		config:  config,
		manager: manager,
	}
}

func (a *api) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.root)
	mux.HandleFunc("/stats", a.stats)
	mux.HandleFunc("/health", a.health)
	mux.HandleFunc("/add_upstream", a.addUpstream)
	mux.HandleFunc("/add_backend", a.addBackend)
	mux.HandleFunc("/remove_upstream", a.removeUpstream)
	mux.HandleFunc("/remove_backend", a.removeBackend)

	a.server = &graceful.Server{
		Server: &http.Server{
			Addr:    fmt.Sprintf(":%d", a.config.Port),
			Handler: mux,
		},
		NoSignalHandling: true,
	}

	go func() {
		log.Println("starting plugin server on port: ", a.config.Port)
		a.errCh <- a.server.ListenAndServe()
	}()

	// wait a short amount of time to make sure that the server doesn't
	// immediately fail. For instance, if the port is taken.
	select {
	case <-time.After(100 * time.Millisecond):
		break
	case err := <-a.errCh:
		return err
	}

	return nil
}

func (a *api) Stop() error {
	a.server.Stop(time.Second)
	// wait a maximum of one second if requests are outstanding and hanging for some reason.
	select {
	case <-time.After(time.Second * 2):
		return fmt.Errorf("timeout waiting for server to stop")
	case err := <-a.errCh:
		return err
	}
	return nil
}

func (a *api) writeResponse(rw http.ResponseWriter, code int, response interface{}) {
	rw.WriteHeader(code)
	rw.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(rw).Encode(response)
	if err != nil {
		log.Println(err)
	}
}

func (a *api) root(rw http.ResponseWriter, req *http.Request) {
	log.Println("/ handler ...")
	resp := rootResponse{
		Status: "ok",
		Endpoints: map[string]string{
			"/health":          "healthcheck",
			"/info":            "information about known upstreams and backends",
			"/add_upstream":    "add a new upstream",
			"/add_backend":     "add a new backend",
			"/remove_upstream": "remove an upstream and all its backends",
			"/remove_backend":  "remove a backend",
		},
	}
	a.writeResponse(rw, 200, &resp)
}

func (a *api) health(rw http.ResponseWriter, req *http.Request) {
	log.Println("/health handler ...")
	a.writeResponse(rw, 200, statusResponse{Status: "ok"})
}

func (a *api) stats(rw http.ResponseWriter, req *http.Request) {
	log.Println("/stats handler ...")
	a.writeResponse(rw, 400, &errorResponse{Status: "not_implemented", Message: "/stats endpoint is not yet implemented"})
}

func (a *api) addUpstream(rw http.ResponseWriter, req *http.Request) {
	log.Println("/add_upstream handler called ...")
	if req.Method != http.MethodPost {
		a.writeResponse(rw, 400, &errorResponse{
			Status:  "invalid_method",
			Message: "/add_upstream only supports HTTP POST requests"})
		return
	}

	rawUpstream := &addUpstreamRequest{}
	if err := json.NewDecoder(req.Body).Decode(rawUpstream); err != nil {
		a.writeResponse(rw, 400, &errorResponse{
			Status:  "invalid_request",
			Message: "/add_upstream expects a json request body with the following keys: `name,protocols,hostnames,prefixes,request_timeout`"})
		return
	}

	upstream, err := newUpstream(rawUpstream)
	if err != nil {
		a.writeResponse(rw, 400, &errorResponse{Status: "invalid_request", Message: err.Error()})
		return
	}

	upstreamID, err := a.manager.AddUpstream(upstream)

	if err != nil {
		a.writeResponse(rw, 400, &errorResponse{
			Status:  "failure",
			Message: err.Error()})
		return
	}

	a.writeResponse(rw, 201, &addUpstreamResponse{
		Status:     "success",
		Upstream:   newFormattedUpstream(upstream),
		UpstreamID: upstreamID,
	})
}

func (a *api) removeUpstream(rw http.ResponseWriter, req *http.Request) {
	log.Println("/remove_upstream handler called ...")
	if req.Method != http.MethodDelete {
		a.writeResponse(rw, 400, &errorResponse{
			Status:  "invalid_method",
			Message: "/remove_upstream only supports HTTP DELETE requests"})
		return
	}

	data := &removeUpstreamRequest{}
	if err := json.NewDecoder(req.Body).Decode(data); err != nil {
		a.writeResponse(rw, 400, &errorResponse{
			Status:  "invalid_request",
			Message: "/remove_upstream expects a json request body with the following keys: `upstream_id`"})
		return
	}

	if err := a.manager.RemoveUpstream(data.UpstreamID); err != nil {
		a.writeResponse(rw, 400, &errorResponse{
			Status:  "failure",
			Message: err.Error(),
		})
		return
	}

	a.writeResponse(rw, 202, &statusResponse{"successfully_deleted"})
}

func (a *api) addBackend(rw http.ResponseWriter, req *http.Request) {
	log.Println("/add_backend handler called ...")
	if req.Method != http.MethodPost {
		a.writeResponse(rw, 400, &errorResponse{
			Status:  "invalid_method",
			Message: "/add_backend only supports HTTP POST requests"})
		return
	}

	rawBackend := &addBackendRequest{}
	if err := json.NewDecoder(req.Body).Decode(rawBackend); err != nil {
		a.writeResponse(rw, 400, &errorResponse{
			Status:  "invalid_request",
			Message: "/add_backend expects a json request body with the following keys: `upstream_id,address,healthcheck`"})
		return
	}

	backend, err := newBackend(rawBackend)
	if err != nil {
		a.writeResponse(rw, 400, &errorResponse{Status: "invalid_request", Message: err.Error()})
		return
	}

	upstreamID := gatekeeper.UpstreamID(rawBackend.UpstreamID)
	backendID, err := a.manager.AddBackend(upstreamID, backend)
	backend.ID = backendID
	if err != nil {
		a.writeResponse(rw, 400, &errorResponse{Status: "failure", Message: err.Error()})
		return
	}

	a.writeResponse(rw, 201, &addBackendResponse{
		Status:     "success",
		Backend:    newFormattedBackend(backend),
		UpstreamID: upstreamID,
		BackendID:  backendID,
	})
}

func (a *api) removeBackend(rw http.ResponseWriter, req *http.Request) {
	log.Println("/remove_backend handler called ...")
	if req.Method != http.MethodDelete {
		a.writeResponse(rw, 400, &errorResponse{
			Status:  "invalid_method",
			Message: "/remove_backend only supports HTTP DELETE requests"})
		return
	}

	data := &removeBackendRequest{}
	if err := json.NewDecoder(req.Body).Decode(data); err != nil {
		a.writeResponse(rw, 400, &errorResponse{
			Status:  "invalid_request",
			Message: "/remove_backend expects a json request body with the following keys: `backend_id`"})
		return
	}

	if err := a.manager.RemoveBackend(data.BackendID); err != nil {
		a.writeResponse(rw, 400, &errorResponse{
			Status:  "failure",
			Message: err.Error(),
		})
		return
	}

	a.writeResponse(rw, 202, &statusResponse{"successfully_deleted"})
}
