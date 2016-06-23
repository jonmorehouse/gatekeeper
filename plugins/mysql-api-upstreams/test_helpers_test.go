package main

import (
	"log"
	"testing"
)

type testError uint

func (t testError) Error() string {
	msg, found := testErrorMapping[t]
	if !found {
		log.Fatal("invalid mapping for error")
	}

	return msg
}

var testErrorMapping = map[testError]string{
	DidNotError: "did not error",
}

const (
	DidNotError testError = iota
)

func Fail(t *testing.T, err error) {
	t.Error(err)
	t.FailNow()
}
