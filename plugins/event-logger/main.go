package main

import (
	"log"

	event_plugin "github.com/jonmorehouse/gatekeeper/plugin/event"
	"github.com/jonmorehouse/gatekeeper/shared"
)

// Plugin is a type that implements the event_plugin.Plugin interface
type Plugin struct{}

func (*Plugin) Start() error                           { return nil }
func (*Plugin) Stop() error                            { return nil }
func (*Plugin) Heartbeat() error                       { return nil }
func (*Plugin) Configure(map[string]interface{}) error { return nil }

func (*Plugin) GeneralMetric(metric *shared.GeneralMetric) error {
	log.Println("general-metric received ...")
	return nil
}

func (*Plugin) RequestMetric(metric *shared.RequestMetric) error {
	log.Println("request-metric received ...")
	return nil
}

func (*Plugin) Error(err error) error {
	log.Println("error received ...")
	return nil
}

func main() {
	plugin := &Plugin{}
	if err := event_plugin.RunPlugin("statsd-event-logger", plugin); err != nil {
		log.Fatal(err)
	}
}
