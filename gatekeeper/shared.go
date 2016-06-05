package gatekeeper

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

func Retry(f func() error, retries uint) error {
	err := f()
	if err == nil {
		return nil
	}

	if retries == 0 {
		return err
	}
	return Retry(f, retries-1)
}

func ReqPrefix(req *http.Request) string {
	// normalize the requestPath to end with a "/" if it doesn't already
	re := regexp.MustCompile(".*/$")
	if !re.MatchString(req.URL.Path) {
		req.URL.Path = fmt.Sprintf("%s/", req.URL.Path)
	}

	pieces := strings.Split(req.URL.Path, "/")
	if len(pieces) == 0 || len(pieces) == 1 {
		return ""
	}

	return pieces[1]
}
