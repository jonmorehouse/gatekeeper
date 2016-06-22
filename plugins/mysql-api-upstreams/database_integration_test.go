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
}

func TestDatabase__RemoveUpstream__NotFound(t *testing.T) {
	// test that this returns an erro when not found
}

func TestDatabase__AddBackend(t *testing.T) {
	// test successfully adding a backend
}

func TestDatabase__AddBackend__Duplicate(t *testing.T) {
	// pass
}

func TestDatabase__AddBackend__UpstreamNotFound(t *testing.T) {
	// pass
}

func TestDatabase__RemoveBackend(t *testing.T) {
	// pass
}

func TestDatabase__RemoveBackend__NotFound(t *testing.T) {
	// pass
}
