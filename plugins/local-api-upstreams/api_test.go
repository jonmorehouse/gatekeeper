package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

func fixtureUpstream() *gatekeeper.Upstream {
	return &gatekeeper.Upstream{
		ID:        "test",
		Name:      "test",
		Hostnames: []string{"test"},
		Protocols: []gatekeeper.Protocol{gatekeeper.HTTPPublic},
		Prefixes:  []string{"test"},
		Timeout:   time.Second,
		Extra: map[string]interface{}{
			"test": "test",
		},
	}
}

func fixtureBackend() *gatekeeper.Backend {
	return &gatekeeper.Backend{
		ID:      "test",
		Address: "http://localhost:44000",
		Extra: map[string]interface{}{
			"test": "test",
		},
	}
}

// newTestServiceContainer returns a service container with upstreams and backends declared
func newTestServiceContainer(t *testing.T) gatekeeper.ServiceContainer {
	container := gatekeeper.NewServiceContainer()

	// add an upstream and a backend for said upstream
	assertNil(t, container.AddUpstream(fixtureUpstream()))
	assertNil(t, container.AddBackend("test", fixtureBackend()))
	return container
}

// apiTest is a testHarness which builds out an api, the request, request body
// and performs the request before passing the response recorder along to the
// callback function passed in.
func apiTest(t *testing.T, method, target string, val interface{}, cb func(*httptest.ResponseRecorder, *api)) {
	api := NewAPI(newTestServiceContainer(t)).(*api)

	// build out the request
	var body io.Reader
	if val != nil {
		jsonBytes, err := json.Marshal(val)
		assertNil(t, err)
		body = bytes.NewBuffer(jsonBytes)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// perform the request and pass the response along
	api.Router().ServeHTTP(rr, req)
	cb(rr, api)
}

// test that an errorStatusCode is returned correctly parsing types where needed
func Test__errorStatusCode__OK(t *testing.T) {
	testCases := []struct {
		err  error
		code int
	}{
		{httpError{"httpError", 501}, 501},
		{errors.New("DEFAULT"), 500},
	}

	for _, testCase := range testCases {
		code := errorStatusCode(testCase.err)
		assertEqual(t, code, testCase.code)
	}
}

// test that writeJSONErrorResponse writes the correct response to the responseWriter
func Test__writeJSONErrorResponse(t *testing.T) {
	testCases := []struct {
		err  error
		resp errorResponse
		code int
	}{
		{gatekeeper.UpstreamNotFoundErr, errorResponse{"UPSTREAM_NOT_FOUND"}, 404},
		{gatekeeper.BackendNotFoundErr, errorResponse{"BACKEND_NOT_FOUND"}, 404},
	}

	for _, testCase := range testCases {
		rr := httptest.NewRecorder()
		writeJSONErrorResponse(rr, testCase.err)

		assertEqual(t, rr.Code, testCase.code)
		assertJSONBuffer(t, rr.Body, testCase.resp)
	}
}

func Test__fetchUpstreamsHandler(t *testing.T) {
	apiTest(t, "GET", "/upstreams", nil, func(rr *httptest.ResponseRecorder, _ *api) {
		assertEqual(t, rr.Code, 200)
		assertJSONBuffer(t, rr.Body, []*upstream{&upstream{
			ID:        "test",
			Name:      "test",
			Protocols: []string{"http-public"},
			Hostnames: []string{"test"},
			Prefixes:  []string{"test"},
			Timeout:   time.Second,
			Extra: map[string]interface{}{
				"test": "test",
			},
			Backends: []*backend{
				&backend{
					ID:      "test",
					Address: "http://localhost:44000",
					Extra: map[string]interface{}{
						"test": "test",
					},
				},
			},
		}})
	})
}

func Test__addUpstreamHandler(t *testing.T) {
	upstream := &upstream{
		ID:        "test-a",
		Name:      "test-a",
		Protocols: []string{"http-public"},
		Prefixes:  []string{"test-a"},
		Hostnames: []string{"test-a"},
		Extra: map[string]interface{}{
			"test": "test",
		},
		Backends: []*backend{},
	}

	apiTest(t, "POST", "/upstreams", upstream, func(rr *httptest.ResponseRecorder, api *api) {
		assertEqual(t, rr.Code, 201)
		assertJSONBuffer(t, rr.Body, "OK")

		fetched, err := api.serviceContainer.UpstreamByID("test-a")
		assertNil(t, err)
		assertNotNil(t, fetched)
		assertEqual(t, fetched.Name, upstream.Name)
		assertEqual(t, fetched.Protocols[0].String(), upstream.Protocols[0])
		assertEqual(t, fetched.Hostnames[0], upstream.Hostnames[0])
		assertEqual(t, fetched.Prefixes[0], upstream.Prefixes[0])
		assertEqual(t, fetched.Extra["test"], upstream.Extra["test"])
	})
}

func Test__addUpstreamHandler__InvalidParams(t *testing.T) {
	apiTest(t, "POST", "/upstreams", nil, func(rr *httptest.ResponseRecorder, api *api) {
		assertEqual(t, rr.Code, 400)
		assertJSONBuffer(t, rr.Body, errorResponse{Msg: "INVALID_UPSTREAM_PARAMS"})
	})
}

func Test__removeUpstreamHandler(t *testing.T) {
	apiTest(t, "DELETE", "/upstreams/test", nil, func(rr *httptest.ResponseRecorder, api *api) {
		assertEqual(t, rr.Code, 204)
		assertJSONBuffer(t, rr.Body, "DELETED")

		fetched, err := api.serviceContainer.UpstreamByID("test")
		assertNotNil(t, err)
		assertEqual(t, true, fetched == nil)
	})
}

func Test__removeUpstreamHandler__UpstreamNotFound(t *testing.T) {
	apiTest(t, "DELETE", "/upstreams/not_found", nil, func(rr *httptest.ResponseRecorder, api *api) {
		assertEqual(t, rr.Code, 404)
		assertJSONBuffer(t, rr.Body, errorResponse{Msg: "UPSTREAM_NOT_FOUND"})
	})
}

func Test__fetchUpstreamHandler(t *testing.T) {
	apiTest(t, "GET", "/upstreams/test", nil, func(rr *httptest.ResponseRecorder, api *api) {
		assertEqual(t, rr.Code, 200)

		expectedResp := toUpstream(fixtureUpstream())
		expectedResp.Backends = []*backend{toBackend(fixtureBackend())}
		assertJSONBuffer(t, rr.Body, expectedResp)
	})
}

func Test__addBackendHandler(t *testing.T) {
	backend := fixtureBackend()
	backend.ID = "test2"

	apiTest(t, "POST", "/upstreams/test/backend", toBackend(backend), func(rr *httptest.ResponseRecorder, api *api) {
		assertEqual(t, rr.Code, 201)
		assertJSONBuffer(t, rr.Body, "OK")

		backends, err := api.serviceContainer.FetchBackends("test")
		assertNil(t, err)
		assertEqual(t, 2, len(backends))
	})
}

func Test__addBackendHandler__InvalidBackendParams(t *testing.T) {
	apiTest(t, "POST", "/upstreams/test/backend", "invalid", func(rr *httptest.ResponseRecorder, api *api) {
		assertEqual(t, rr.Code, 400)
		assertJSONBuffer(t, rr.Body, errorResponse{Msg: "INVALID_BACKEND_PARAMS"})
	})
}

func Test__removeBackendHandler(t *testing.T) {
	apiTest(t, "DELETE", "/upstreams/test/backend/test", nil, func(rr *httptest.ResponseRecorder, api *api) {
		assertEqual(t, rr.Code, 204)
		assertJSONBuffer(t, rr.Body, "DELETED")
	})
}

func Test__removeBackendHandler__BackendNotFound(t *testing.T) {
	apiTest(t, "DELETE", "/upstreams/test/backend/invalid", nil, func(rr *httptest.ResponseRecorder, api *api) {
		assertEqual(t, rr.Code, 404)
	})
}
