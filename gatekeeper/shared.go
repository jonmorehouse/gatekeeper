package gatekeeper

import (
	"fmt"
	"log"
	"os"
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
