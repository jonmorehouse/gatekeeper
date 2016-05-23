package upstream

type UpstreamID string

var NilUpstreamID UpstreamID = ""

type Upstream struct {
	ID   UpstreamID
	Name string
}

var NilUpstream Upstream = Upstream{}

type BackendID string

var NilBackendID BackendID = ""

type Backend struct {
	ID      BackendID
	Address string
}

var NilBackend Backend = Backend{}
