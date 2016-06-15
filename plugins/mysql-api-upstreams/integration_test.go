// +build integration
package main

import (
	"testing"

	"github.com/jonmorehouse/gatekeeper/shared"
)

// integration tests that assert that when multiple instances of the
// mysql-api-upstreams are running (ie: on different hosts/gatekeepr instances)
// that we synchronize upstreams correctly between the instances.

func makeUpstreams() []*shared.Upstream {
	return []*shared.Upstream(nil)
}

func makeBackends() []*shared.Backend {
	return []*shared.Backend(nil)
}

// return a new client to the testDatabase to be used for persisting data
func newTestDatabase() Database {
	//dsn := "gatekeeper:gatekeeper@tcp(localhost:3306)/gatekeeper"
	return nil

}

func TestIntegration__FetchUpstreamsAtStart(t *testing.T) {
	// make sure that upstreams that are written to the database are synced at startup!

}

func TestIntegration__AddUpstream(t *testing.T) {
	// make sure that adding an upstream is eventually synced across startups
}

func TestIntegration__AddBackend(t *testing.T) {
	// make sure that adding a backend
}
