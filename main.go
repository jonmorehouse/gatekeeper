package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type stringValue struct {
	value string
}

func (s *stringValue) Set(value string) error {
	s.value = value
	return nil
}

func (s *stringValue) String() string {
	return s.value
}

func parseExtraFlags(args []string) []*flag.Flag {
	flags := make([]*flag.Flag, 0, 0)

	var current *flag.Flag
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			if current == nil {
				log.Fatal(gatekeeper.InvalidPluginArgs)
			}
			current.Value.Set(arg)
			continue
		}

		// if this is a flag and current is non-nil, then we need to create a new flag
		if current != nil {
			flags = append(flags, current)
		}

		pieces := strings.SplitN(arg, "=", 2)
		current = &flag.Flag{
			Name:  pieces[0],
			Value: &stringValue{},
		}

		if len(pieces) == 2 {
			current.Value.Set(pieces[1])
		}
	}

	if current != nil {
		flags = append(flags, current)
	}

	return flags
}

func prefixedArgs(flags []*flag.Flag, prefix string) map[string]interface{} {
	args := make(map[string]interface{})

	for _, argFlag := range flags {
		if !strings.HasPrefix(argFlag.Name, prefix) {
			continue
		}

		key := strings.TrimLeft(argFlag.Name, prefix)
		if argFlag.Value.String() == "" {
			args[key] = struct{}{}
		} else {
			args[key] = argFlag.Value.String()
		}
	}
	return args
}

func main() {
	// plugin configuration
	loadBalancerPlugin := flag.String("loadbalancer-plugin", "simple-loadbalancer", "loadbalancer-plugin cmd. default: simple-loadbalancer")
	// accept comma delimited lists of plugins for metric, upstream and modifier plugins
	upstreamPlugins := flag.String("upstream-plugins", "static-upstreams", "comma-delimited upstream-plugin cmds. default: static-upstreams")
	metricPlugins := flag.String("metric-plugins", "metric-logger", "comma-delimited metric-plugin cmds. default: metric-plugins")
	modifierPlugins := flag.String("modifier-plugins", "modifier", "comma-delimited modifier-plugin cmds. default: modifier-plugins")

	// server configuration
	httpPublic := flag.Bool("http-public", true, "http-public enabled. default: true")
	httpPublicPort := flag.Uint("http-public-port", 8000, "http-public listen port. default: 8000")

	httpInternal := flag.Bool("http-internal", false, "http-internal enabled. default: false")
	httpInternalPort := flag.Uint("http-internal-port", 8001, "http-public listen port. default: 8001")

	httpsPublic := flag.Bool("https-public", false, "https-public enabled. default: false")
	httpsPublicPort := flag.Uint("https-public-port", 443, "http-public listen port. default: 443")

	httpsInternal := flag.Bool("https-internal", false, "https-internal false")
	httpsInternalPort := flag.Uint("https-internal-port", 444, "http-internal listen port. default: 444")

	// configure both a plugin and request timeout
	pluginTimeoutStr := flag.String("plugin-timeout", "10ms", "plugin call timeout. default 10ms")
	proxyTimeoutStr := flag.String("default-proxy-timeout", "5s", "default proxy request timeout, for when an upstream doesn't declare one. default: 5s")

	commandLine := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	commandLine.Parse(os.Args)
	extraFlags := parseExtraFlags(commandLine.Args()[1:])

	pluginTimeout, err := time.ParseDuration(*pluginTimeoutStr)
	if err != nil {
		log.Fatal(gatekeeper.InvalidPluginTimeoutError)
		return
	}

	proxyTimeout, err := time.ParseDuration(*proxyTimeoutStr)
	if err != nil {
		log.Fatal(gatekeeper.InvalidProxyTimeoutError)
		return
	}

	options := gatekeeper.Options{
		UpstreamPlugins:    strings.Split(*upstreamPlugins, ","),
		UpstreamPluginArgs: prefixedArgs(extraFlags, "-upstream-"),

		MetricPlugins:    strings.Split(*metricPlugins, ","),
		MetricPluginArgs: prefixedArgs(extraFlags, "-metric-"),

		ModifierPlugins:    strings.Split(*modifierPlugins, ","),
		ModifierPluginArgs: prefixedArgs(extraFlags, "-modifier-"),

		LoadBalancerPlugin:     *loadBalancerPlugin,
		LoadBalancerPluginArgs: prefixedArgs(extraFlags, "-loadbalancer-"),

		HTTPPublic:     *httpPublic,
		HTTPPublicPort: *httpPublicPort,

		HTTPInternal:     *httpInternal,
		HTTPInternalPort: *httpInternalPort,

		HTTPSPublic:     *httpsPublic,
		HTTPSPublicPort: *httpsPublicPort,

		HTTPSInternal:     *httpsInternal,
		HTTPSInternalPort: *httpsInternalPort,

		DefaultProxyTimeout: proxyTimeout,
		PluginTimeout:       pluginTimeout,
	}

	// build the server application which manages multiple servers
	// listening on multiple ports.
	app, err := gatekeeper.New(options)
	if err != nil {
		log.Fatal(err)
	}

	stopCh := make(chan struct{})
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

		<-signals
		// by default, we give 10 seconds for the app to shut down gracefully
		if err := app.Stop(time.Second * 10); err != nil {
			log.Fatal(err)
		}
		stopCh <- struct{}{}
	}()

	// Start and run the application. This blocks
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
	// wait for the application to finish shutting down
	<-stopCh
}
