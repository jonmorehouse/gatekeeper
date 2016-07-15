package core

import "time"

type startStopper interface {
	Start() error
	Stop(time.Duration) error
}
