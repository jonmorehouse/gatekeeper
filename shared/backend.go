package shared

type BackendID string

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
	HealthCheck string
}
