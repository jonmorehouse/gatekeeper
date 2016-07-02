package gatekeeper

import "time"

func InStrList(value string, list []string) bool {
	for _, item := range list {
		if value == item {
			return true
		}
	}

	return false
}

func Retry(retries uint, f func() error) error {
	err := f()
	if err == nil {
		return nil
	}

	if retries == 0 {
		return err
	}
	return Retry(f, retries-1)
}

func CallWithTimeout(dur time.Duration, f func() error) (error, bool) {
	var err error
	doneCh := make(struct{})

	go func() {
		err = f()
		doneCh <- struct{}{}
	}()

	select {
	case <-doneCh:
		break
	case <-time.After(dur):
		return nil, false
	}

	return err, true
}
