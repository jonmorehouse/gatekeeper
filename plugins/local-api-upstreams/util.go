package main

import (
	"errors"
	"net"
	"strconv"
)

func GetListenerPort(listener net.Listener) (int, error) {
	if listener == nil {
		return 0, errors.New("nil listener")
	}

	_, portStr, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		return 0, err
	}

	port64, err := strconv.ParseInt(portStr, 10, 0)
	if err != nil {
		return 0, err
	}

	return int(port64), nil
}
