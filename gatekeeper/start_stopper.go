package gatekeeper

import "time"

type StartStopper interface {
	Start() error
	Stop(time.Duration) error
}
