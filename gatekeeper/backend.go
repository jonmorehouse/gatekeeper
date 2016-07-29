package gatekeeper

type BackendID string

func (b BackendID) String() string {
	return string(b)
}

var NilBackendID BackendID = ""
var NilBackend Backend = Backend{}

func NewBackendID() BackendID {
	return BackendID(GetUUID())
}

type Backend struct {
	ID      BackendID
	Address string
	Extra   map[string]interface{}
}
