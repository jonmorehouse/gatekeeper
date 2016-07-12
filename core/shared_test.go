package core

import (
	"net/http"
	"net/url"
	"testing"
)

func TestReqPrefix_findsPrefix(t *testing.T) {
	testCases := []struct {
		url    string
		prefix string
	}{
		{"https://github.com/", ""},
		{"https://github.com/foo", "foo"},
		{"https://github.com/foo/", "foo"},
		{"https://github.com/foo/bar", "foo"},
		{"https://github.com?foo=bar", ""},
	}

	for _, testCase := range testCases {
		url, _ := url.Parse(testCase.url)
		req := &http.Request{
			URL: url,
		}
		if ReqPrefix(req) != testCase.prefix {
			t.Log(ReqPrefix(req))
			t.Fatalf("did not parse prefix correctly")
		}
	}
}
