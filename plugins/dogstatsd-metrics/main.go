package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/datadog/datadog-go/statsd"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	"github.com/jonmorehouse/gatekeeper/gatekeeper/utils"
	metrics_plugin "github.com/jonmorehouse/gatekeeper/plugin/metric"
)

func milliseconds(i time.Duration) float64 {
	return float64(i.Nanoseconds()) / float64(1000)
}

type config struct {
	Namespace  string   `flag:"metric-namespace"`
	Tags       []string `flag:"metric-tags"`
	SampleRate float64  `flag:"metric-sample-rate"`

	DogStatsdAddress string `flag:"metric-dogstatsd-address"`
	Debug            bool   `flag:"metric-debug"`
}

type plugin struct {
	statsd Statsd
	config *config
}

func newPlugin() *plugin {
	return &plugin{}
}

func (p *plugin) Start() error {
	var client Statsd
	if p.config.Debug {
		client = NewDebugStatsd(p.config.Namespace, p.config.Tags)
	} else {
		client, err := statsd.New(p.config.DogStatsdAddress)
		if err != nil {
			return err
		}
		client.Tags = p.config.Tags
		client.Namespace = p.config.Namespace
	}

	p.statsd = client
	p.statsd.Count("plugin.started", 1, []string{}, p.config.SampleRate)
	return nil
}

func (p *plugin) Stop() error {
	p.statsd.Count("plugin.stopped", 1, []string{}, p.config.SampleRate)
	return p.statsd.Close()
}

func (p *plugin) Configure(opts map[string]interface{}) error {
	var config config
	if err := utils.ParseConfig(opts, &config); err != nil {
		return err
	}
	p.config = &config
	return nil
}

func (p *plugin) Heartbeat() error {
	p.statsd.Count("plugin.heartbeat", 1, []string{}, p.config.SampleRate)
	return nil
}

func (p *plugin) EventMetric(metric *gatekeeper.EventMetric) error {
	p.statsd.Count("event."+metric.Event.String(), 1.0, []string{}, p.config.SampleRate)
	return nil
}

func (p *plugin) PluginMetric(metric *gatekeeper.PluginMetric) error {
	tags := []string{
		"plugin_type:" + metric.PluginType,
		"plugin_name:" + metric.PluginName,
		"plugin_method:" + metric.MethodName,
	}

	p.statsd.Count("plugin.call", 1.0, tags, p.config.SampleRate)
	p.statsd.TimeInMilliseconds("plugin.latency", 1.0, tags, p.config.SampleRate)
	return nil
}

func (p *plugin) ProfilingMetric(metric *gatekeeper.ProfilingMetric) error {
	// general statistics
	p.statsd.Histogram("mem.alloc_bytes", float64(metric.MemStats.Alloc), []string{}, p.config.SampleRate)
	p.statsd.Histogram("mem.total_alloc_bytes", float64(metric.MemStats.TotalAlloc), []string{}, p.config.SampleRate)
	p.statsd.Histogram("mem.sys_bytes", float64(metric.MemStats.Sys), []string{}, p.config.SampleRate)
	p.statsd.Histogram("mem.lookups", float64(metric.MemStats.Lookups), []string{}, p.config.SampleRate)
	p.statsd.Histogram("mem.mallocs", float64(metric.MemStats.Mallocs), []string{}, p.config.SampleRate)
	p.statsd.Histogram("mem.frees", float64(metric.MemStats.Frees), []string{}, p.config.SampleRate)

	// heap statistics
	p.statsd.Histogram("mem.heap_alloc_bytes", float64(metric.MemStats.HeapAlloc), []string{}, p.config.SampleRate)
	p.statsd.Histogram("mem.heap_sys_bytes", float64(metric.MemStats.HeapSys), []string{}, p.config.SampleRate)
	p.statsd.Histogram("mem.heap_idle_bytes", float64(metric.MemStats.HeapIdle), []string{}, p.config.SampleRate)
	p.statsd.Histogram("mem.heap_inuse_bytes", float64(metric.MemStats.HeapInuse), []string{}, p.config.SampleRate)
	p.statsd.Histogram("mem.heap_released_bytes", float64(metric.MemStats.HeapReleased), []string{}, p.config.SampleRate)
	p.statsd.Histogram("mem.heap_objects", float64(metric.MemStats.HeapObjects), []string{}, p.config.SampleRate)

	// garbage collector statistics
	p.statsd.Histogram("gc.count", float64(metric.MemStats.NumGC), []string{}, p.config.SampleRate)
	p.statsd.Histogram("gc.cpu_fraction", metric.MemStats.GCCPUFraction, []string{}, p.config.SampleRate)
	p.statsd.Histogram("gc.pause_total_ns", float64(metric.MemStats.PauseTotalNs), []string{}, p.config.SampleRate)

	return nil
}

func (p *plugin) RequestMetric(metric *gatekeeper.RequestMetric) error {
	tags := []string{
		"upstream.id:" + string(metric.Upstream.ID),
		"upstream.name:" + metric.Upstream.Name,
		"backend.id:" + string(metric.Backend.ID),
		"backend.address:" + metric.Backend.Address,
	}

	// latencies
	p.statsd.TimeInMilliseconds("request.latency", milliseconds(metric.Latency), tags, p.config.SampleRate)
	p.statsd.TimeInMilliseconds("request.internal_latency", milliseconds(metric.InternalLatency), tags, p.config.SampleRate)
	p.statsd.TimeInMilliseconds("request.dns_lookup_latency", milliseconds(metric.DNSLookupLatency), tags, p.config.SampleRate)
	p.statsd.TimeInMilliseconds("request.tcp_connect_latency", milliseconds(metric.TCPConnectLatency), tags, p.config.SampleRate)
	p.statsd.TimeInMilliseconds("request.proxy_latency", milliseconds(metric.ProxyLatency), tags, p.config.SampleRate)

	// connection meta information
	p.statsd.Count("request.dns_lookup", 1.0, tags, p.config.SampleRate)
	p.statsd.Count("request.conn_reused", 1.0, tags, p.config.SampleRate)
	p.statsd.Count("request.conn_was_idle", 1.0, tags, p.config.SampleRate)
	p.statsd.TimeInMilliseconds("request.conn_idle_time", milliseconds(metric.ConnIdleTime), tags, p.config.SampleRate)

	// plugin latencies
	p.statsd.TimeInMilliseconds("request.router_latency", milliseconds(metric.RouterLatency), tags, p.config.SampleRate)
	p.statsd.TimeInMilliseconds("request.load_balancer_latency", milliseconds(metric.LoadBalancerLatency), tags, p.config.SampleRate)
	p.statsd.TimeInMilliseconds("request.request_modifier_latency", milliseconds(metric.RequestModifierLatency), tags, p.config.SampleRate)
	p.statsd.TimeInMilliseconds("request.response_modifier_latency", milliseconds(metric.ResponseModifierLatency), tags, p.config.SampleRate)
	p.statsd.TimeInMilliseconds("request.error_response_modifier_latency", milliseconds(metric.ErrorResponseModifierLatency), tags, p.config.SampleRate)

	if metric.Response.Error != nil {
		p.statsd.Count("request.error", 1, append(tags, "error:"+metric.Response.Error.Error()), p.config.SampleRate)
	}

	// request metrics
	p.statsd.Count("request.count", 1.0, tags, p.config.SampleRate)
	tags = append(tags, fmt.Sprintf("code:%d", metric.Response.StatusCode))
	tags = append(tags, fmt.Sprintf("status:%dxx", metric.Response.StatusCode/100))
	p.statsd.Count("request.response", 1.0, tags, p.config.SampleRate)

	return nil
}

func (p *plugin) UpstreamMetric(metric *gatekeeper.UpstreamMetric) error {
	tags := []string{
		"upstream.name:" + metric.Upstream.Name,
		"upstream.id:" + string(metric.Upstream.ID),
	}

	var key string
	switch metric.Event {
	case gatekeeper.UpstreamAddedEvent:
		key = "upstream.upstream_added"
	case gatekeeper.UpstreamRemovedEvent:
		key = "upstream.upstream_removed"
	case gatekeeper.BackendAddedEvent:
		key = "upstream.backend_added"
	case gatekeeper.BackendRemovedEvent:
		key = "upstream.backend_removed"
	default:
		return errors.New("invalid upstream metric event")
	}
	p.statsd.Count(key, 1.0, tags, p.config.SampleRate)
	return nil
}

func main() {
	plugin := newPlugin()
	if err := metrics_plugin.RunPlugin("", plugin); err != nil {
		log.Fatal(err)
	}
}
