package utils

import (
	"net"
	"testing"

	"github.com/jonmorehouse/gatekeeper/gatekeeper/test"
)

func TestGetPort__Ok(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	test.AssertNil(t, err)

	port, err := GetListenerPort(listener)
	test.AssertNil(t, err)
	test.AssertEqual(t, true, port > 0 && port < 665535)
}
