package main

import (
	"log"

	metric_plugin "github.com/jonmorehouse/gatekeeper/plugin/metric"
	"github.com/jonmorehouse/gatekeeper/shared"
)

// Plugin is a type that implements the event_plugin.Plugin interface
type Plugin struct{}

func (*Plugin) Start() error {
	log.Println("metric-logger started...")
	return nil
}
func (*Plugin) Stop() error {
	log.Println("metric-logger stopped...")
	return nil
}
func (*Plugin) Heartbeat() error                       { return nil }
func (*Plugin) Configure(map[string]interface{}) error { return nil }

func (*Plugin) EventMetric(metric *shared.EventMetric) error {
	return nil
}

func (*Plugin) ProfilingMetric(metric *shared.ProfilingMetric) error {
	return nil
}

func (*Plugin) PluginMetric(metric *shared.PluginMetric) error {
	return nil
}

func (*Plugin) RequestMetric(metric *shared.RequestMetric) error {
	log.Println("OverallLatency: ", metric.Latency)
	log.Println("InternalLatency: ", metric.InternalLatency)
	log.Println("ProxyLatency: ", metric.ProxyLatency)
	log.Println("RequestModifierLatency: ", metric.RequestModifierLatency)
	log.Println("LoadBalancerLatency: ", metric.LoadBalancerLatency)
	log.Println("ResponseModifierLatency: ", metric.ResponseModifierLatency)
	return nil
}

func (*Plugin) UpstreamMetric(metric *shared.UpstreamMetric) error {
	return nil
}

func main() {
	plugin := &Plugin{}
	if err := metric_plugin.RunPlugin("metric-logger", plugin); err != nil {
		log.Fatal(err)
	}
}
