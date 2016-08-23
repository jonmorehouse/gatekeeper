package gatekeeper

import "log"

func RetryAndPanic(retries uint, f func() error) {
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
