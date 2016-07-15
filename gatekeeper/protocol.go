package gatekeeper

import (
	"fmt"
	"log"
)

type Protocol uint

const (
	HTTPPublic Protocol = iota + 1
	HTTPInternal
	HTTPSPublic
	HTTPSInternal
)

var formattedProtocols = map[Protocol]string{
	HTTPPublic:    "http-public",
	HTTPInternal:  "http-internal",
	HTTPSPublic:   "https-public",
	HTTPSInternal: "https-internal",
}

func (p Protocol) String() string {
	str, ok := formattedProtocols[p]
	if !ok {
		log.Fatal("programming error; unformatted protocol")
	}
	return str
}

func NewProtocol(value string) (Protocol, error) {
	for protocol, str := range formattedProtocols {
		if str == value {
			return protocol, nil
		}
	}

	return Protocol(0), fmt.Errorf("unknown protocol")
}

func NewProtocols(values []string) ([]Protocol, error) {
	protocols := make([]Protocol, len(values))
	for idx, value := range values {
		protocol, err := NewProtocol(value)
		if err != nil {
			return []Protocol(nil), err
		}
		protocols[idx] = protocol

	}

	return protocols, nil
}
