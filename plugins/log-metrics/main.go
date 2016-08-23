package main

import (
	"fmt"
	"log"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	metric_plugin "github.com/jonmorehouse/gatekeeper/plugin/metric"
)

func maxIdx(vals []uint64) int {
	return 0
}

// Plugin is a type that implements the event_plugin.Plugin interface
type plugin struct{}

func (*plugin) Start() error                           { return nil }
func (*plugin) Stop() error                            { return nil }
func (*plugin) Configure(map[string]interface{}) error { return nil }

func (*plugin) Heartbeat() error {
	log.Println("metric-logger heartbeat ...")
	return nil
}

func (*plugin) EventMetric(metric *gatekeeper.EventMetric) error {
	msg := fmt.Sprintf("metric.event.%s ", metric.Event.String())
	for k, v := range metric.Extra {
		msg += fmt.Sprintf("extra.%s=%s ", k, v)
	}
	log.Println(msg)
	return nil
}

func (*plugin) ProfilingMetric(metric *gatekeeper.ProfilingMetric) error {
	// write out general profiling statistics
	log.Println(fmt.Sprintf("metric.profiling.memstats.alloc bytes=%v", metric.MemStats.Alloc))
	log.Println(fmt.Sprintf("metric.profiling.memstats.total_alloc bytes=%v", metric.MemStats.TotalAlloc))
	log.Println(fmt.Sprintf("metric.profiling.memstats.sys bytes=%v", metric.MemStats.Sys))
	log.Println(fmt.Sprintf("metric.profiling.memstats.lookups count=%v", metric.MemStats.Lookups))
	log.Println(fmt.Sprintf("metric.profiling.memstats.mallocs count=%v", metric.MemStats.Mallocs))
	log.Println(fmt.Sprintf("metric.profiling.memstats.frees count=%v", metric.MemStats.Frees))

	// heap allocation statistics
	log.Println(fmt.Sprintf("metric.profiling.memstats.heap_alloc bytes=%v", metric.MemStats.HeapAlloc))
	log.Println(fmt.Sprintf("metric.profiling.memstats.heap_sys bytes=%v", metric.MemStats.HeapSys))
	log.Println(fmt.Sprintf("metric.profiling.memstats.heap_idle bytes=%v", metric.MemStats.HeapIdle))
	log.Println(fmt.Sprintf("metric.profiling.memstats.heap_inuse bytes=%v", metric.MemStats.HeapInuse))
	log.Println(fmt.Sprintf("metric.profiling.memstats.heap_released bytes%v=", metric.MemStats.HeapReleased))
	log.Println(fmt.Sprintf("metric.profiling.memstats.heap_objects count=%v", metric.MemStats.HeapObjects))

	// low level structure allocation statistics
	log.Println(fmt.Sprintf("metric.profiling.memstats.stack_inuse bytes=%v", metric.MemStats.StackInuse))
	log.Println(fmt.Sprintf("metric.profiling.memstats.stack_sys bytes=%v", metric.MemStats.StackSys))
	log.Println(fmt.Sprintf("metric.profiling.memstats.mspan_inuse bytes=%v", metric.MemStats.MSpanInuse))
	log.Println(fmt.Sprintf("metric.profiling.memstats.mspan_sys bytes=%v", metric.MemStats.MSpanSys))
	log.Println(fmt.Sprintf("metric.profiling.memstats.mcache_inuse bytes=%v", metric.MemStats.MCacheInuse))
	log.Println(fmt.Sprintf("metric.profiling.memstats.mcache_sys bytes=%v", metric.MemStats.MCacheSys))
	log.Println(fmt.Sprintf("metric.profiling.memstats.buck_hash_sys bytes=%v", metric.MemStats.BuckHashSys))
	log.Println(fmt.Sprintf("metric.profiling.memstats.gc_sys bytes=%v", metric.MemStats.GCSys))
	log.Println(fmt.Sprintf("metric.profiling.memstats.other_sys bytes=%v", metric.MemStats.OtherSys))

	// garbage collector statistics
	log.Println(fmt.Sprintf("metric.profiling.memstats.next_gc count=%v", metric.MemStats.NextGC))
	log.Println(fmt.Sprintf("metric.profiling.memstats.last_gc ts=%v", metric.MemStats.LastGC))
	log.Println(fmt.Sprintf("metric.profiling.memstats.pause_total_ns ns=%v", metric.MemStats.PauseTotalNs))
	idx := maxIdx(metric.MemStats.PauseNs[:])
	log.Println(fmt.Sprintf("metric.profiling.memstats.longest_recent_pause ns%v=", metric.MemStats.PauseNs[idx]))
	log.Println(fmt.Sprintf("metric.profiling.memstats.longest_recent_pause_end ts%v=", metric.MemStats.PauseEnd[idx]))
	log.Println(fmt.Sprintf("metric.profiling.memstats.num_gc count=%v", metric.MemStats.NumGC))
	log.Println(fmt.Sprintf("metric.profiling.memstats.gc_cpu_fraction percent=%v", metric.MemStats.GCCPUFraction*100))

	return nil
}

func (*plugin) PluginMetric(metric *gatekeeper.PluginMetric) error {
	log.Println(fmt.Sprintf("metric.plugin.%s.%s.%s latency=%s", metric.PluginType, metric.PluginName, metric.MethodName, metric.Latency))
	return nil
}

func (*plugin) RequestMetric(metric *gatekeeper.RequestMetric) error {
	log := func(msg string) {
		// append upstream name / ID
		// append backend name / ID
		msg = fmt.Sprintf("%s upstream.name=%s upstream.id=%s", metric.Upstream.Name, metric.Upstream.ID)
		if metric.Backend == nil {
			msg = fmt.Sprintf("%s backend.id=%s backend.address=%s", metric.Backend.ID, metric.Backend.Address)
		}
		log.Println(msg)
	}

	// print out request metrics
	log(fmt.Sprintf("metric.request.remote_addr value=%s", metric.Request.RemoteAddr))
	log(fmt.Sprintf("metric.request.method value=%s", metric.Request.Method))
	log(fmt.Sprintf("metric.request.host value=%s", metric.Request.Host))
	log(fmt.Sprintf("metric.request.prefix value=%s", metric.Request.Prefix))
	log(fmt.Sprintf("metric.request.path value=%s", metric.Request.Path))
	log(fmt.Sprintf("metric.request.upstream_match_type value=%s", metric.Request.UpstreamMatchType.String()))
	for k, vs := range metric.Request.Header {
		for _, v := range vs {
			log(fmt.Sprintf("metric.request.header %s=%s", k, v))
		}
	}

	// print out response metrics
	log(fmt.Sprintf("metric.response.status value=%d", metric.Response.StatusCode))
	log(fmt.Sprintf("metric.response.proto value=%s", metric.Response.Proto))
	log(fmt.Sprintf("metric.response.content_length value=%d", metric.Response.ContentLength))
	log(fmt.Sprintf("metric.response.transfer_encoding value=%s", metric.Response.TransferEncoding))
	for k, vs := range metric.Response.Header {
		for _, v := range vs {
			log(fmt.Sprintf("metric.response.header %s=%s", k, v))
		}
	}

	// internal latencies
	log(fmt.Sprintf("metric.request.start_ts value=%d", metric.RequestStartTS.Unix()))
	log(fmt.Sprintf("metric.request.end_ts value=%d", metric.RequestEndTS.Unix()))
	log(fmt.Sprintf("metric.request.latency value=%d", metric.Latency))
	log(fmt.Sprintf("metric.request.internal_latency value=%d", metric.InternalLatency))
	log(fmt.Sprintf("metric.request.dns_lookup_latency value=%d", metric.DNSLookupLatency))
	log(fmt.Sprintf("metric.request.tcp_connect_latency value=%d", metric.TCPConnectLatency))
	log(fmt.Sprintf("metric.request.proxy_latency value=%d", metric.ProxyLatency))

	// connection meta information
	log(fmt.Sprintf("metric.request.dns_lookup value=%b", metric.DNSLookup))
	log(fmt.Sprintf("metric.request.conn_reused value=%b", metric.ConnReused))
	log(fmt.Sprintf("metric.request.conn_was_idle value=%b", metric.ConnReused))
	log(fmt.Sprintf("metric.request.conn_idle_time value=%b", metric.ConnIdleTime))

	// plugin latencies
	log(fmt.Sprintf("metric.request.router_latency value=%s", metric.RouterLatency))
	log(fmt.Sprintf("metric.request.loadbalancer_latency value=%s", metric.LoadBalancerLatency))
	log(fmt.Sprintf("metric.request.response_modifier_latency value=%s", metric.ResponseModifierLatency))
	log(fmt.Sprintf("metric.request.request_modifier_latency value=%s", metric.RequestModifierLatency))

	if metric.Error != nil {
		log(fmt.Sprintf("metric.request.error value=%s", metric.Error))
		log(fmt.Sprintf("metric.request.error_response_modifier value=%s", metric.ErrorResponseModifierLatency))
	}
	return nil
}

func (*plugin) UpstreamMetric(metric *gatekeeper.UpstreamMetric) error {
	msg := fmt.Sprintf("metric.upstream.%s upstream.name=%s upstream.id=%s", metric.Event.String(), metric.Upstream.Name, metric.Upstream.ID)
	if metric.Backend != nil {
		msg += fmt.Sprintf(" backend.ID=%s backend.Address=%s", metric.Backend.ID, metric.Backend.Address)
	}
	log.Println(msg)
	return nil
}

func main() {
	if err := metric_plugin.RunPlugin("metric-logger", &plugin{}); err != nil {
		log.Fatal(err)
	}
}
