package gatekeeper

import (
	"fmt"
	"os"
)

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

func GetUUID() string {
	var uuid string

	RetryAndPanic(3, func() error {
		var err error
		uuid, err = NewUUID()
		return err
	})

	return uuid
}
