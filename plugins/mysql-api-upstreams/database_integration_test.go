// +build integration

// integration tests that write state to a MySQL datastore and tests that
// operationally, read/writes work correctly. This test suite expects a few
// things; notably that a mysql instance is running locally and has been
// migrated to the correct data store version. Furthermore, it expects that it
// is accessible via the test dataSourceName specified below.
package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
)

var testDataSourceName = "gatekeeper:gatekeeper@tcp(127.0.0.1:3306)/gatekeeper"

// build a new database, ensuring that its empty and connected
func setup(t *testing.T) Database {
	db := NewDatabase()
	if err := db.Connect(testDataSourceName); err != nil {
		Fail(t, err)
	}

	// clear all tables, immediately failing if any errors happen
	tables := []string{"upstream", "upstream_mapping", "backend"}
	for _, table := range tables {
		if _, err := db.(*database).db.Exec(fmt.Sprintf("DELETE FROM %s;", table)); err != nil {
			Fail(t, err)
		}
	}

	return db
}

func TestDatabase__Connect(t *testing.T) {
	db := setup(t)
	if err := db.(*database).db.Ping(); err != nil {
		Fail(t, err)
	}
}

func TestDatabase__Connect__AlreadyConnected(t *testing.T) {
	db := setup(t)
	if err := db.Connect(testDataSourceName); err == nil {
		Fail(t, DidNotError)
	}
}

func TestDatabase__Connect__InvalidDSN(t *testing.T) {
	db := NewDatabase()
	if err := db.Connect("invalid_dsn"); err == nil {
		Fail(t, DidNotError)
	}
}

func TestDatabase__Disconnect(t *testing.T) {
	db := setup(t)
	if err := db.Disconnect(); err != nil {
		Fail(t, err)
	}
}

func TestDatabase__Disconnect__NotConnected(t *testing.T) {
	db := NewDatabase()
	if err := db.Disconnect(); err == nil {
		Fail(t, DidNotError)
	}
}

func TestDatabase__AddUpstream(t *testing.T) {
	upstream := &shared.Upstream{
		Name:      "test",
		Timeout:   time.Second * 2,
		Protocols: []shared.Protocol{shared.HTTPPublic, shared.HTTPInternal},
		Prefixes:  []string{"/test", "/testa"},
		Hostnames: []string{"test", "test.com"},
	}

	db := setup(t)
	upstreamID, err := db.AddUpstream(upstream)
	if err != nil {
		Fail(t, err)
	}

	rows, err := db.(*database).db.Query("SELECT * FROM `upstream` WHERE `name`=? AND `id`=?", "test", upstreamID.String())
	if err != nil {
		Fail(t, err)
	}

	// if there are more than one row or less than one row, error out
	if !rows.Next() || rows.Next() {
		Fail(t, fmt.Errorf("did not return correct number of rows"))
	}

	if rows.Err() != nil {
		Fail(t, rows.Err())
	}
}

func TestDatabase__AddUpstream__Duplicate(t *testing.T) {
	upstream := &shared.Upstream{
		Name:      "test",
		Timeout:   time.Second * 2,
		Protocols: []shared.Protocol{shared.HTTPPublic, shared.HTTPInternal},
		Prefixes:  []string{"/test", "/testa"},
		Hostnames: []string{"test", "test.com"},
	}

	db := setup(t)
	if _, err := db.AddUpstream(upstream); err != nil {
		Fail(t, err)
	}

	upstream.Protocols = []shared.Protocol{}
	upstream.Prefixes = []string{}
	upstream.Hostnames = []string{}
	if _, err := db.AddUpstream(upstream); err == nil {
		Fail(t, DidNotError)
	}
}

func TestDatabase__AddUpstream__DuplicatePrefix(t *testing.T) {
	upstream := &shared.Upstream{
		Name:      "test",
		Timeout:   time.Second * 2,
		Protocols: []shared.Protocol{shared.HTTPPublic, shared.HTTPInternal},
		Prefixes:  []string{"/test"},
		Hostnames: []string{"test", "test.com"},
	}

	db := setup(t)
	if _, err := db.AddUpstream(upstream); err != nil {
		Fail(t, err)
	}

	upstream.Name = "testa"
	upstream.Protocols = []shared.Protocol{}
	upstream.Prefixes = []string{"/test"}
	upstream.Hostnames = []string{}
	if _, err := db.AddUpstream(upstream); err == nil {
		Fail(t, DidNotError)
	}
}

func TestDatabase__AddUpstream__DuplicateHostname(t *testing.T) {
	upstream := &shared.Upstream{
		Name:      "test",
		Timeout:   time.Second * 2,
		Protocols: []shared.Protocol{shared.HTTPPublic, shared.HTTPInternal},
		Prefixes:  []string{"/test"},
		Hostnames: []string{"test", "test.com"},
	}

	db := setup(t)
	if _, err := db.AddUpstream(upstream); err != nil {
		Fail(t, err)
	}

	upstream.Name = "testa"
	upstream.Protocols = []shared.Protocol{}
	upstream.Prefixes = []string{}
	if _, err := db.AddUpstream(upstream); err == nil {
		Fail(t, DidNotError)
	}
}

func TestDatabase__AddUpstream__Duplicates(t *testing.T) {
	upstreams := []*shared.Upstream{
		&shared.Upstream{
			Name:      "test",
			Timeout:   time.Second,
			Protocols: []shared.Protocol{shared.HTTPPublic, shared.HTTPPublic},
			Prefixes:  []string{"/test"},
			Hostnames: []string{"test"},
		},
		&shared.Upstream{
			Name:      "test",
			Timeout:   time.Second,
			Protocols: []shared.Protocol{shared.HTTPPublic},
			Prefixes:  []string{"/test", "/test"},
			Hostnames: []string{"test"},
		},
		&shared.Upstream{
			Name:      "test",
			Timeout:   time.Second,
			Protocols: []shared.Protocol{shared.HTTPPublic},
			Prefixes:  []string{"/test"},
			Hostnames: []string{"test", "test"},
		},
	}

	for _, upstream := range upstreams {
		db := setup(t)
		if _, err := db.AddUpstream(upstream); err == nil {
			Fail(t, DidNotError)
		}
		if err := db.Disconnect(); err != nil {
			Fail(t, err)
		}
	}
}

func TestDatabase__RemoveUpstream(t *testing.T) {
	upstream := &shared.Upstream{
		Name:      "test",
		Timeout:   time.Second,
		Protocols: []shared.Protocol{shared.HTTPPublic},
		Prefixes:  []string{"/test"},
		Hostnames: []string{"test"},
	}
	db := setup(t)

	upstreamID, err := db.AddUpstream(upstream)
	if err != nil {
		Fail(t, err)
	}

	// remove the upstream and make sure it doesn't exist anymore
	if err := db.RemoveUpstream(upstreamID); err != nil {
		Fail(t, err)
	}

	rows, err := db.(*database).db.Query("SELECT * FROM `upstream` WHERE `name`=? AND `id`=?", "test", upstreamID.String())
	if err != nil {
		Fail(t, err)
	}
	if rows.Next() {
		Fail(t, fmt.Errorf("Did not delete upstream properly"))
	}
	if rows.Err() != nil {
		Fail(t, rows.Err())
	}

	// make sure that all references to the upstream from the mapping table were deleted

	// make sure that the backends were deleted too
}

func TestDatabase__RemoveUpstream__NotFound(t *testing.T) {
	db := setup(t)
	if err := db.RemoveUpstream(shared.UpstreamID("invalid")); err != nil {
		Fail(t, fmt.Errorf("should not error when the backend doesn't exist"))
	}
	db.Disconnect()
	if err := db.RemoveUpstream(shared.UpstreamID("invalid")); err == nil {
		Fail(t, DidNotError)
	}
}

func TestDatabase__AddBackend(t *testing.T) {
	db := setup(t)
	upstream := &shared.Upstream{
		Name:      "test",
		Protocols: []shared.Protocol{},
		Prefixes:  []string{},
		Hostnames: []string{},
		Timeout:   time.Second,
	}
	upstreamID, err := db.AddUpstream(upstream)
	if err != nil {
		Fail(t, err)
	}

	backend := &shared.Backend{
		Address:     "localhost:port",
		HealthCheck: "test",
	}

	backendID, err := db.AddBackend(upstreamID, backend)
	if err != nil {
		Fail(t, err)
	}
	if backendID == shared.NilBackendID {
		Fail(t, fmt.Errorf("Nil backend ID generated"))
	}

	// check the database to make sure we wrote the backend correctly
	rows, err := db.(*database).db.Query("SELECT * FROM `backend` WHERE `id`=?", backendID.String())
	if err != nil {
		Fail(t, err)
	}
	if !rows.Next() || rows.Err() != nil {
		Fail(t, fmt.Errorf("Did not return backend from the database"))
	}
}

func TestDatabase__AddBackend__Duplicate(t *testing.T) {
	db := setup(t)
	upstream := &shared.Upstream{
		Name:      "test",
		Protocols: []shared.Protocol{},
		Prefixes:  []string{},
		Hostnames: []string{},
		Timeout:   time.Second,
	}
	upstreamID, err := db.AddUpstream(upstream)
	if err != nil {
		Fail(t, err)
	}

	backend := &shared.Backend{
		Address:     "localhost",
		HealthCheck: "/health",
	}

	backendID, err := db.AddBackend(upstreamID, backend)
	if err != nil {
		Fail(t, err)
	}
	if backendID == shared.NilBackendID {
		Fail(t, fmt.Errorf("Nil backend ID generated"))
	}

	backendID, err = db.AddBackend(upstreamID, backend)
	if err == nil {
		Fail(t, DidNotError)
	}
	if backendID != shared.NilBackendID {
		Fail(t, fmt.Errorf("Did Not return a nil backendID"))
	}
}

func TestDatabase__AddBackend__UpstreamNotFound(t *testing.T) {
	db := setup(t)
	backend := &shared.Backend{
		Address:     "localhost",
		HealthCheck: "/health",
	}

	backendID, err := db.AddBackend(shared.NilUpstreamID, backend)
	if err == nil {
		Fail(t, DidNotError)
	}
	if backendID != shared.NilBackendID {
		Fail(t, fmt.Errorf("Did not return a nil BackendID"))
	}
}

func TestDatabase__RemoveBackend(t *testing.T) {
	db := setup(t)
	upstream := &shared.Upstream{
		Name:      "test",
		Protocols: []shared.Protocol{},
		Prefixes:  []string{},
		Hostnames: []string{},
		Timeout:   time.Second,
	}
	upstreamID, err := db.AddUpstream(upstream)
	if err != nil {
		Fail(t, err)
	}

	backend := &shared.Backend{
		Address:     "localhost",
		HealthCheck: "/health",
	}
	backendID, err := db.AddBackend(upstreamID, backend)
	if err != nil {
		Fail(t, err)
	}
	if err := db.RemoveBackend(backendID); err != nil {
		Fail(t, err)
	}

	rows, err := db.(*database).db.Query("SELECT `id` FROM `backend` WHERE `id`=?", backendID.String())
	if err != nil {
		Fail(t, err)
	}

	if rows.Next() {
		Fail(t, fmt.Errorf("Backend returned from database after removing backend"))
	}
}

func TestDatabase__RemoveBackend__NotFound(t *testing.T) {
	db := setup(t)
	fakeBackendID := shared.BackendID("invalid")
	if err := db.RemoveBackend(fakeBackendID); err != nil {
		Fail(t, DidNotError)
	}
}

func TestDatabase__FetchUpstreams(t *testing.T) {
	db := setup(t)
	upstreams := []*shared.Upstream{
		&shared.Upstream{
			Name:      "1",
			Protocols: []shared.Protocol{shared.HTTPPublic, shared.HTTPInternal},
			Prefixes:  []string{"/1"},
			Hostnames: []string{"1.com"},
			Timeout:   time.Millisecond * 100,
		},
		&shared.Upstream{
			Name:      "2",
			Protocols: []shared.Protocol{shared.HTTPPublic, shared.HTTPInternal},
			Prefixes:  []string{"/2"},
			Hostnames: []string{"2.com"},
			Timeout:   time.Millisecond * 100,
		},
		&shared.Upstream{
			Name:      "3",
			Protocols: []shared.Protocol{shared.HTTPPublic, shared.HTTPInternal},
			Prefixes:  []string{"/3"},
			Hostnames: []string{"3.com"},
			Timeout:   time.Millisecond * 100,
		},
	}

	for _, upstream := range upstreams {
		upstreamID, err := db.AddUpstream(upstream)
		if err != nil {
			Fail(t, err)
		}
		upstream.ID = upstreamID
	}

	dbUpstreams, err := db.FetchUpstreams()
	if err != nil {
		Fail(t, err)
	}

	if len(dbUpstreams) != len(upstreams) {
		Fail(t, fmt.Errorf("Did not return correct number of upstreams..."))
	}

	for _, dbUpstream := range dbUpstreams {
		found := false
		for _, upstream := range upstreams {
			if upstream.ID == dbUpstream.ID {
				found = true
			}
		}
		if !found {
			Fail(t, fmt.Errorf("did not return all upstreams"))
		}
	}
}

func TestDatabase__FetchUpstreams__NoneExist(t *testing.T) {
	db := setup(t)
	upstreams, err := db.FetchUpstreams()
	if err != nil {
		Fail(t, err)
	}

	if len(upstreams) != 0 {
		Fail(t, fmt.Errorf("Did not return empty array"))
	}
}

func TestDatabase__FetchUpstreamBackends(t *testing.T) {
	db := setup(t)
	upstream := &shared.Upstream{
		Name:      "test",
		Protocols: []shared.Protocol{shared.HTTPPublic, shared.HTTPInternal},
		Prefixes:  []string{"/test"},
		Hostnames: []string{"test.com"},
	}
	upstreamID, err := db.AddUpstream(upstream)
	if err != nil {
		Fail(t, err)
	}

	backend := &shared.Backend{
		Address:     "localhost",
		HealthCheck: "/health",
	}
	backendID, err := db.AddBackend(upstreamID, backend)
	if err != nil {
		Fail(t, err)
	}
	if backendID == shared.NilBackendID {
		Fail(t, fmt.Errorf("did not return correct backendID"))
	}

	backends, err := db.FetchUpstreamBackends(upstreamID)
	if err != nil {
		Fail(t, err)
	}

	if len(backends) != 1 {
		Fail(t, fmt.Errorf("did not return upstream's backends"))
	}
	if backends[0].ID != backendID {
		Fail(t, fmt.Errorf("returned an invalid backend"))
	}

}
func TestDatabase__FetchUpstreamBackends__UpstreamNotFound(t *testing.T) {
	db := setup(t)
	backends, err := db.FetchUpstreamBackends(shared.UpstreamID("invalid"))
	if err != nil {
		Fail(t, err)
	}

	if len(backends) != 0 {
		Fail(t, fmt.Errorf("Did not return empty slice of backends"))
	}
}
