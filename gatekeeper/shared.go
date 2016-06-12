package gatekeeper

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
