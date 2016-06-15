// +build integration
package main

import (
	"testing"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
)

func TestDatabase__Connect(t *testing.T) {
	// pass
}

func TestDatabase__Disconnect(t *testing.T) {
	// pass
}

func TestDatabase__AddUpstream(t *testing.T) {
	upstream := &shared.Upstream{
		ID:        shared.NilUpstreamID,
		Name:      "test",
		Hostnames: []string{"test.com"},
		Prefixes:  []string{"/test"},
		Protocols: []shared.Protocol{shared.HTTPPublic},
		Timeout:   time.Second,
	}

	dsn := "gatekeeper:gatekeeper@tcp(127.0.0.1:3306)/gatekeeper"
	database := NewDatabase()
	if err := database.Connect(dsn); err != nil {
		t.Fatalf(err.Error())
	}
	if _, err := database.AddUpstream(upstream); err != nil {
		t.Fatalf(err.Error())
	}
	database.Disconnect()
}
