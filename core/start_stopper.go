package core

import "time"

type starter interface {
	Start() error
}

type stopper interface {
	Stop() error
}

type gracefulStopper interface {
	Stop(time.Duration) error
}

func filterStarters(input []interface{}, cb func(starter) error) error {
	errs := NewMultiError()

	for _, item := range input {
		if i, ok := item.(starter); ok {
			if err := cb(i); err != nil {
				errs.Add(err)
			}
		}
	}

	return errs.ToErr()
}

func filterStoppers(input []interface{}, cb func(stopper) error) error {
	errs := NewMultiError()

	for _, item := range input {
		if i, ok := item.(stopper); ok {
			if err := cb(i); err != nil {
				errs.Add(err)
			}
		}
	}

	return errs.ToErr()
}

func filterGracefulStoppers(input []interface{}, cb func(gracefulStopper) error) error {
	errs := NewMultiError()

	for _, item := range input {
		if i, ok := item.(gracefulStopper); ok {
			if err := gracefulStopper(i); err != nil {
				errs.Add(err)
			}
		}
	}

	return errs.ToErr()
}
