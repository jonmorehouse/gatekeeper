package gatekeeper

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

func RetryAndPanic(f func() error, retries uint) {
	// retry a function n times before panicing and closing out the
	// program. This should only be for exceptional cases
	err := f()

	for i := uint(0); i <= retries; i++ {
		if err == nil {
			return
		}

		err = f()
	}
	log.Fatal(err)
}

func NewUUID() (string, error) {
	f, err := os.Open("/dev/urandom")
	defer f.Close()
	if err != nil {
		return "", err
	}

	b := make([]byte, 16)
	f.Read(b)

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
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
