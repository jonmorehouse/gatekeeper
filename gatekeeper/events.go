package gatekeeper

// Event's are generic events that are useful for birds eye views of the
// application. Specifically, some of these metrics are redundant to others
// across the application simply to allow for users whom are only interested in
// high level metrics to be able to garner the information they care about.
type Event uint

const (
	AppStartedEvent Event = iota + 1
	AppStoppedEvent

	ServerStartedEvent
	ServerStoppedEvent

	RequestAcceptedEvent
	RequestSuccessEvent
	RequestErrorEvent
	RequestFinishedEvent

	UpstreamAddedEvent
	UpstreamRemovedEvent

	BackendAddedEvent
	BackendRemovedEvent

	MetricsFlushedEvent
	MetricsFlushedSuccessEvent
	MetricsFlushedErrorEvent

	PluginStartedEvent
	PluginRestartedEvent
	PluginErrorEvent
	PluginStoppedEvent
	PluginFailedEvent
	PluginRetryEvent

	PluginHeartbeatNotOkEvent
	PluginHeartbeatOkEvent
)

var eventMapping = map[Event]string{
	AppStartedEvent: "app.started",
	AppStoppedEvent: "app.stopped",

	ServerStartedEvent: "server.started",
	ServerStoppedEvent: "server.stopped",

	RequestAcceptedEvent: "request.accepted",
	RequestSuccessEvent:  "request.succeeded",
	RequestErrorEvent:    "request.error",
	RequestFinishedEvent: "request.finished",

	UpstreamAddedEvent:   "upstream.added",
	UpstreamRemovedEvent: "upstream.removed",

	BackendAddedEvent:   "backend.added",
	BackendRemovedEvent: "backend.removed",

	MetricsFlushedEvent:        "metrics.flush",
	MetricsFlushedSuccessEvent: "metrics.flush_success",
	MetricsFlushedErrorEvent:   "metrics.flush_error",

	PluginStartedEvent:   "plugin.started",
	PluginRestartedEvent: "plugin.restarted",
	PluginErrorEvent:     "plugin.error",
	PluginStoppedEvent:   "plugin.stopped",
	PluginFailedEvent:    "plugin.failed",

	PluginHeartbeatNotOkEvent: "plugin.heartbeat_failure",
	PluginHeartbeatOkEvent:    "plugin.hearbeat",
}

func (m Event) String() string {
	desc, found := eventMapping[m]
	if !found {
		ProgrammingError("Event string mapping not found")
	}

	return desc
}
