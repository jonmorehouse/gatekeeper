package utils

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/jonmorehouse/gatekeeper/gatekeeper/test"
)

// newTestService returns a service that can be used to mock out the Server type
func newTestService(t *testing.T) *mockTestService {
	return &mockTestService{test.NewMock(t)}
}

type mockTestService struct {
	test.Mock
}

func (m *mockTestService) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		io.WriteString(rw, "OK")
	})

	return mux
}

// assertServer accepts a mock and a port and asserts that the mock is being
// served traffic on that port
func assertServer(t *testing.T, mock *mockTestService, port int) {
	// make a request to the listening server
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d", port))
	test.AssertNil(t, err)
	test.AssertNotNil(t, resp)
	test.AssertEqual(t, resp.StatusCode, 200)
}

// serverTest builds a test context for testing out the server functionality
func serverTest(t *testing.T, cb func(*mockTestService, Server)) {
	service := newTestService(t)
	server := NewDefaultServer(service)
	cb(service, server)
	server.Stop()
}

// test out various components
func TestServer__StartOnPort(t *testing.T) {
	serverTest(t, func(mock *mockTestService, server Server) {
		test.AssertNil(t, server.StartOnPort(9999))
		assertServer(t, mock, 9999)
	})
}

//
func TestServer__StartOnPort__DuplicatePort(t *testing.T) {
	serverTest(t, func(mock *mockTestService, server Server) {
		test.AssertNil(t, server.StartOnPort(9998))
		assertServer(t, mock, 9998)

		serverTest(t, func(mock *mockTestService, server Server) {
			test.AssertNotNil(t, server.StartOnPort(9998))
		})
	})
}

func TestServer__StartAnywhere(t *testing.T) {
	serverTest(t, func(mock *mockTestService, server Server) {
		port, err := server.StartAnywhere()
		test.AssertNil(t, err)
		assertServer(t, mock, port)
	})
}

func TestServer__StartListener(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	test.AssertNil(t, err)

	port, err := GetListenerPort(listener)
	test.AssertNil(t, err)

	serverTest(t, func(mock *mockTestService, server Server) {
		test.AssertNil(t, server.StartListener(listener))
		assertServer(t, mock, port)
	})
}

func TestServer__GetPort(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	test.AssertNil(t, err)

	serverTest(t, func(mock *mockTestService, server Server) {
		test.AssertNil(t, server.StartListener(listener))
		port, err := server.GetPort()
		test.AssertNil(t, err)
		assertServer(t, mock, port)
	})
}

func TestServer__Stop(t *testing.T) {
	serverTest(t, func(mock *mockTestService, server Server) {
		port, err := server.StartAnywhere()
		test.AssertNil(t, err)
		assertServer(t, mock, port)

		server.Stop()
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d", port))
		test.AssertNotNil(t, err)
		test.AssertEqual(t, true, resp == nil)
	})
}
