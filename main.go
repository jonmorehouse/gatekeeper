package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

func parseJSONOpts(blob string) (map[string]interface{}, error) {
	var opts map[string]interface{}
	return opts, json.Unmarshal([]byte(blob), &opts)
}

func main() {
	options := gatekeeper.Options{}

	// UpstreamPlugin configuration
	upstreamPlugins := flag.String("upstream-plugins", "static-upstreams", "comma delimited list of plugin executables")
	flag.UintVar(&options.UpstreamPluginsCount, "upstream-plugins-count", 1, "number of instances of each upstream plugin to operate")
	upstreamPluginOpts := flag.String("upstream-plugins-opts", "{}", "json encoded options to be passed to each upstream plugin")

	// LoadBalancerPlugin configuration
	flag.StringVar(&options.LoadBalancerPlugin, "loadbalancer-plugin", "simple-loadbalancer", "name of loadbalancer plugin to use")
	flag.UintVar(&options.LoadBalancerPluginsCount, "loadbalancer-plugins-count", 1, "number of instances of each loadbalancer plugin to operate")
	loadBalancerPluginOpts := flag.String("loadbalancer-plugins-opts", "{}", "json encoded options to be passed to each loadbalancer plugin")

	// eventPlugin configuration
	eventPlugins := flag.String("event-plugins", "event-logger", "comma delimited list of event plugin executables. default: event-logger")
	flag.UintVar(&options.EventPluginsCount, "event-plugins-count", 1, "number of instances of each event plugin to operate")
	eventPluginOpts := flag.String("event-plugins-opts", "{}", "json encoded options to be passed to each event plugin")

	// modifierPlugin configuration
	modifierPlugins := flag.String("modifier-plugins", "modifier", "comma delimited list of modifier plugin executables. default: modifier")
	flag.UintVar(&options.ModifierPluginsCount, "modifier-plugins-count", 1, "number of instances of each modifier plugin to operate")
	modifierPluginOpts := flag.String("modifier-plugins-opts", "{}", "json encoded options to be passed to each modifier plugin")

	// Configure Listen Ports for different protocols
	flag.UintVar(&options.HTTPPublicPort, "http-public-port", 8000, "listen port for http-public traffic. default: 8000")
	flag.UintVar(&options.HTTPInternalPort, "http-internal-port", 8001, "listen port for http-internal traffic. default: 8001")

	defaultTimeoutSeconds := flag.Uint("default-timeout", 10, "default-timeout in seconds. default: 10s")

	flag.Parse()

	// parse flags into the correct gatekeeper.Options attributes
	var err error

	// validate upstream plugin options
	options.UpstreamPlugins = strings.Split(*upstreamPlugins, ",")
	options.UpstreamPluginOpts, err = parseJSONOpts(*upstreamPluginOpts)
	if err != nil {
		log.Fatal("Invalid JSON for upstream-plugin-opts")
	}

	// validate load balancer plugin configuration
	options.LoadBalancerPluginOpts, err = parseJSONOpts(*loadBalancerPluginOpts)
	if err != nil {
		log.Fatal("Invalid JSON for loadbalancer-plugin-opts")
	}

	// validate event plugin configuration
	options.EventPlugins = strings.Split(*eventPlugins, ",")
	options.EventPluginOpts, err = parseJSONOpts(*eventPluginOpts)
	if err != nil {
		log.Fatal("Invalid JSON for event-plugin-opts")
	}

	options.ModifierPlugins = strings.Split(*modifierPlugins, ",")
	options.ModifierPluginOpts, err = parseJSONOpts(*modifierPluginOpts)
	if err != nil {
		log.Fatal("Invalid JSON for response-plugin-opts")
	}

	options.DefaultTimeout = time.Second * time.Duration(*defaultTimeoutSeconds)

	// build the server application which manages multiple servers
	// listening on multiple ports.
	app, err := gatekeeper.New(options)
	if err != nil {
		log.Fatal(err)
	}

	stopCh := make(chan interface{})
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

		<-signals
		log.Println("Caught a signal...")
		// by default, we give 10 seconds for the app to shut down gracefully
		if err := app.Stop(time.Second * 10); err != nil {
			log.Fatal(err)
		}
		log.Println("Successfully shutdown application")
		stopCh <- struct{}{}
	}()

	// Start and run the application. This blocks
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
	// wait for the application to finish shutting down
	<-stopCh
}
