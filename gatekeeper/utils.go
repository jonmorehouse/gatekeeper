package gatekeeper

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

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
