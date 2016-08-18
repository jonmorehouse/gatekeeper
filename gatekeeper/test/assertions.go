package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
)

func fail(t *testing.T, msg string) {
	debug.PrintStack()

	t.Fatalf(msg)
}

func AssertTrue(t *testing.T, value bool) {
	if value == true {
		return
	}

	fail(t, fmt.Sprintf("expected true got %v", value))
}

func AssertFalse(t *testing.T, value bool) {
	if value == false {
		return
	}

	fail(t, fmt.Sprintf("expected false got %v", value))
}

func isNil(obj interface{}) bool {
	if obj == nil {
		return true
	}

	value := reflect.ValueOf(obj)
	kind := value.Kind()
	if kind >= reflect.Chan && kind <= reflect.Slice && value.IsNil() {
		return true
	}

	return false
}

func AssertNil(t *testing.T, value interface{}) {
	if isNil(value) {
		return
	}

	fail(t, fmt.Sprintf("expected nil got %v", value))
}

func AssertNotNil(t *testing.T, value interface{}) {
	if !isNil(value) {
		return
	}

	fail(t, fmt.Sprintf("expected not nil, got %v", value))
}

func AssertEqual(t *testing.T, a, b interface{}) {
	if reflect.DeepEqual(a, b) {
		return
	}

	fail(t, fmt.Sprintf("expected %v got %v", a, b))
}

func AssertJSON(t *testing.T, buf []byte, value interface{}) {
	byt, err := json.Marshal(buf)
	AssertNil(t, err)
	AssertEqual(t, strings.TrimSpace(bytes.NewBuffer(buf).String()), string(byt))
}
