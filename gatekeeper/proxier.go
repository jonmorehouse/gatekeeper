package gatekeeper

import (
	"io"
	"net/http"
)

type Proxier struct {
	protocol int
}

func NewProxier(protocol int) *Proxier {
	return &Proxier{
		protocol: protocol,
	}
}

func (p Proxier) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	io.WriteString(rw, "not implemented yet")
}
