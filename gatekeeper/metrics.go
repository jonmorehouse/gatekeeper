package gatekeeper

import (
	"runtime"
	"time"
)

// Metric is the general type that all metrics implement
type Metric interface{}

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
	Event     Event

	// extra values are able to be passed along to any event metric. This
	// should be treated as just metadata and not as a viable means of
	// passing granular metrics around. Specifically, we pass more granular
	// metrics by other types throughout here.
	Extra map[string]string
}

// Profiling metrics are useful for periodically emitting information on the
// internal state of the application. Specifically, this is an internal
// mechanism for exposing pprof data to the metrics plugin.
type ProfilingMetric struct {
	Timestamp time.Time
	MemStats  runtime.MemStats
}

type PluginResponse uint

const (
	PluginResponseOk PluginResponse = iota + 1
	PluginResponseNotOk
)

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

	Error *Error // an error that may or may not have arisen
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
	Latency           time.Duration
	InternalLatency   time.Duration // total local latency, including
	DNSLookupLatency  time.Duration
	TCPConnectLatency time.Duration
	ProxyLatency      time.Duration

	// Connection meta inforamtion
	DNSLookup    bool
	ConnReused   bool
	ConnWasIdle  bool
	ConnIdleTime time.Duration

	// Plugin Latencies
	RouterLatency                time.Duration
	LoadBalancerLatency          time.Duration
	RequestModifierLatency       time.Duration
	ResponseModifierLatency      time.Duration
	ErrorResponseModifierLatency time.Duration

	// Any sort of error that could have been bubbled up throughout the
	// request path
	Error *Error
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
	Event     Event
	Timestamp time.Time

	Upstream *Upstream
	Backend  *Backend
}
