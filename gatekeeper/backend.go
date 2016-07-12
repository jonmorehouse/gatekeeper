package gatekeeper

type BackendID string

func (b BackendID) String() string {
	return string(b)
}

var NilBackendID BackendID = ""
var NilBackend Backend = Backend{}

func NewBackendID() BackendID {
	var uuid string
	RetryAndPanic(func() error {
		var err error
		uuid, err = NewUUID()
		return err
	}, 3)
	return BackendID(uuid)
}

type Backend struct {
	ID          BackendID
	Address     string
	Healthcheck string
}
