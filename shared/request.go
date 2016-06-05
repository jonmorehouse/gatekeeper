package shared

// An RPC safe request that permits us to pass it around between plugins
type Request struct {
	Protocol
	Headers  map[string]interface{}
	Upstream Upstream
	Backend  Backend
	Err      error
	// defaults to nil
	Response Response
}
