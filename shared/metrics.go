package shared

import "time"

type MetricType uint

const (
	EventMetricType MetricType = iota + 1
	ProfilingMetricType
	PluginMetricType
	RequestMetricType
	UpstreamMetricType
)

var metricTypeMapping = map[MetricType]string{
	EventMetricType:     "event metric",
	ProfilingMetricType: "profiling metric",
	PluginMetricType:    "plugin metric",
	RequestMetricType:   "request metric",
	UpstreamMetricType:  "upstream metric",
}

func (m MetricType) String() string {
	desc, found := metricTypeMapping[m]
	if !found {
		ProgrammingError("MetricType string mapping not found")
	}
	return desc
}

// EventMetrics are generic metrics that are useful for emitting events across
// the wire. These are more application specific metrics such as the
// application starting, stopping, a plugin starting/stopping, an upstream
// being added / removed etc.
type EventMetric struct {
	Timestamp time.Time
	Event     MetricEvent

	// extra values are able to be passed along to any event metric. This
	// should be treated as just metadata and not as a viable means of
	// passing granular metrics around. Specifically, we pass more granular
	// metrics by other types throughout here.
	Extra map[string]string
}

// MetricEvent's are generic events that are useful for birds eye views of the
// application. Specifically, some of these metrics are redundant to others
// across the application simply to allow for users whom are only interested in
// high level metrics to be able to garner the information they care about.
type MetricEvent uint

const (
	AppStartedEvent MetricEvent = iota + 1
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

	PluginHeartbeatNotOkEvent
	PluginHeartbeatOkEvent
)

var metricEventMapping = map[MetricEvent]string{
	AppStartedEvent: "app started",
	AppStoppedEvent: "app stopped event",

	ServerStartedEvent: "server started",
	ServerStoppedEvent: "server stopped",

	RequestAcceptedEvent: "request accepted",
	RequestSuccessEvent:  "request succeeded",
	RequestErrorEvent:    "request error",
	RequestFinishedEvent: "request finished",

	UpstreamAddedEvent:   "upstream added",
	UpstreamRemovedEvent: "upstream removed",

	BackendAddedEvent:   "backend added",
	BackendRemovedEvent: "backend removed",

	MetricsFlushedEvent:        "metrics flushed event",
	MetricsFlushedSuccessEvent: "flush metrics success",
	MetricsFlushedErrorEvent:   "flush metrics error",

	PluginStartedEvent:   "plugin started",
	PluginRestartedEvent: "plugin restarted",
	PluginErrorEvent:     "plugin error",
	PluginStoppedEvent:   "plugin stopped",
	PluginFailedEvent:    "plugin failed",

	PluginHeartbeatNotOkEvent: "plugin heartbeat not ok",
	PluginHeartbeatOkEvent:    "plugin heartbeat ok",
}

func (m MetricEvent) String() string {
	desc, found := metricEventMapping[m]
	if !found {
		ProgrammingError("MetricEvent string mapping not found")
	}

	return desc
}

// Profiling metrics are useful for periodically emitting information on the
// internal state of the application. Specifically, this is an internal
// mechanism for exposing pprof data to the metrics plugin.
type ProfilingMetric struct {
	// TODO: add in pprof internally
	Timestamp time.Time
}

type PluginResponse uint

const (
	PluginResponseOk PluginResponse = iota + 1
	PluginResponseNotOk
)

var pluginResponseMapping = map[PluginResponse]string{
	PluginResponseOk:    "ok",
	PluginResponseNotOk: "error",
}

func (p PluginResponse) String() string {
	desc, ok := pluginResponseMapping[p]
	if !ok {
		ProgrammingError("PluginResponse string mapping not found")
	}
	return desc
}

// TODO: maybe add in RequestPayloadSize and ResponsePayloadSize

// Plugin metrics are useful for monitoring latency, error and payloads that we
// are sending to Plugins via unix domain socket. These are intended to be
// higher level metrics and in a future iteration could include more granular
// metrics embedded in them.
type PluginMetric struct {
	Timestamp time.Time
	Latency   time.Duration

	PluginType string
	PluginName string
	MethodName string

	ResponseType PluginResponse
	Error        error // a general error that we'd like to emit
}

// RequestMetrics provide the most granular insight into a request and are
// useful for debugging, request tracing, performance profiling and almost
// anything else. This metric passes along the Request, Response, Upstream and
// Backend (if found) and as such, gives access to the internals of any part of
// the request for a plugin.
type RequestMetric struct {
	Timestamp time.Time

	Request      *Request
	Response     *Response
	ResponseType ResponseType //

	Upstream *Upstream
	Backend  *Backend

	RequestStartTS time.Time
	RequestEndTS   time.Time

	// Latencies
	Latency                time.Duration
	InternalLatency        time.Duration // total local latency, including
	PluginLatency          time.Duration // total latency making calls to RPC plugins
	ProxyLatency           time.Duration // total latency actually proxying the request
	UpstreamMatcherLatency time.Duration // total latency matching the request to an upstream

	// Plugin Latency
	LoadBalancerLatency     time.Duration
	RequestModifierLatency  time.Duration
	ResponseModifierLatency time.Duration

	// proxy level latencies
	BackendConnectLatency time.Duration
	BackendRequestLatency time.Duration
	BackendOverallLatency time.Duration

	// Any sort of error that could have been bubbled up throughout the
	// request path
	Error error

	// Number of outstanding requests at the total,upstream and backend
	// granularity. Specifically, this is the number of outstanding
	// requests _at_ request time.
	OutstandingRequestsCount         uint
	OutstandingUpstreamRequestsCount uint
	OutstandingBackendRequestsCount  uint
}

// UpstreamMetrics are useful for garnering granular metrics on particular
// upstreams. Specifically, this metric type allows metrics-plugin authors to
// gather granular metrics on a particular upstream or backend to profile its
// performance in a single place.

// NOTE: users of this plugin need to watch out for null values. Because its
// sort of an awkward metric and encapsulates quite a bit, we need to use
// common sense and only use the metrics we're interested in. EG: don't try to
// use the Latency attribute when the metric-type is UpstreamAdded
type UpstreamMetric struct {
	Event     MetricEvent
	Timestamp time.Time

	Upstream *Upstream
	Backend  *Backend
}
