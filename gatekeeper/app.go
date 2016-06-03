package gatekeeper

type App struct {
	broadcaster Broadcaster

	// upstreamPublisher publishes messages from upstreamPlugins back to
	// the broadcaster
	upstreamPlugins   PluginManager
	upstreamPublisher Publisher

	// LoadBalancer listens to events from the Broadcaster and updates
	// backendPlugins with the correct state. This type also surfaces
	// backends in the request lifecycle, by proxying to the loadbalancer
	// plugins.
	loadbalancer        LoadBalancer
	loadbalancerPlugins PluginManager

	// RequestModifier is used to modify a request before proxying it to
	// the backend specified.
	requestModifier        RequestModifier
	requestModifierPlugins PluginManager

	// ResponseModifier is used to modify a response before returning the
	// response to the client.
	responseModifier        ResponseModifier
	responseModifierPlugins PluginManager
}
