package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/jonmorehouse/gatekeeper/core"
)

func flagSetToMap(set *flag.FlagSet) map[string]interface{} {
	vals := make(map[string]interface{})

	set.VisitAll(func(f *flag.Flag) {
		vals[f.Name] = f.Value.String()
	})

	return vals
}

func parseFlags(options *core.Options) error {
	commandLine := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// plugin configuration
	useLocalLoadBalancer := commandLine.Bool("local-loadbalancer", false, "use the provided in process load balancer. default: false")
	loadBalancerPlugin := commandLine.String("loadbalancer-plugin", "simple-loadbalancer", "loadbalancer-plugin cmd. default: simple-loadbalancer")

	useLocalRouter := commandLine.Bool("local-router", false, "use the provided in process router. default: false")
	routerPlugin := commandLine.String("router-plugin", "example-router", "router-plugin cmd. default: example-router")

	// accept comma delimited lists of plugins for metric, upstream and modifier plugins
	upstreamPlugins := commandLine.String("upstream-plugins", "static-upstreams", "comma-delimited upstream-plugin cmds. default: static-upstreams")
	metricPlugins := commandLine.String("metric-plugins", "metric-logger", "comma-delimited metric-plugin cmds. default: metric-plugins")
	modifierPlugins := commandLine.String("modifier-plugins", "example-modifier", "comma-delimited modifier-plugin cmds. default: example-modifier")

	// server configuration
	httpPublic := commandLine.Bool("http-public", true, "http-public enabled. default: true")
	httpPublicPort := commandLine.Uint("http-public-port", 8000, "http-public listen port. default: 8000")

	httpInternal := commandLine.Bool("http-internal", false, "http-internal enabled. default: false")
	httpInternalPort := commandLine.Uint("http-internal-port", 8001, "http-public listen port. default: 8001")

	httpsPublic := commandLine.Bool("https-public", false, "https-public enabled. default: false")
	httpsPublicPort := commandLine.Uint("https-public-port", 443, "http-public listen port. default: 443")

	httpsInternal := commandLine.Bool("https-internal", false, "https-internal false")
	httpsInternalPort := commandLine.Uint("https-internal-port", 444, "http-internal listen port. default: 444")

	// configure both a plugin and request timeout
	pluginTimeout := commandLine.Duration("plugin-timeout", 10*time.Millisecond, "plugin call timeout. default 10ms")
	proxyTimeout := commandLine.Duration("default-proxy-timeout", 5*time.Second, "default proxy request timeout. default 5s")

	metricBufferSize := commandLine.Uint("metric-buffer-size", 10000, "metric buffer size")
	metricFlushInterval := commandLine.Duration("metrif-flush-interval", 100*time.Millisecond, "max interval between metric flushes")

	knownFlags := map[string]struct{}{
		"local-loadbalancer":    struct{}{},
		"loadbalancer-plugin":   struct{}{},
		"local-router":          struct{}{},
		"router-plugin":         struct{}{},
		"upstream-plugins":      struct{}{},
		"metric-plugins":        struct{}{},
		"modifier-plugins":      struct{}{},
		"http-public":           struct{}{},
		"http-public-port":      struct{}{},
		"http-internal":         struct{}{},
		"http-internal-port":    struct{}{},
		"https-public":          struct{}{},
		"https-public-port":     struct{}{},
		"https-internal":        struct{}{},
		"https-internal-port":   struct{}{},
		"plugin-timeout":        struct{}{},
		"proxy-timeout":         struct{}{},
		"metric-buffer-size":    struct{}{},
		"metric-flush-interval": struct{}{},
	}

	flagSets := map[string]*flag.FlagSet{
		"core":         commandLine,
		"upstream":     flag.NewFlagSet("upstream", flag.ContinueOnError),
		"loadbalancer": flag.NewFlagSet("loadbalancer", flag.ContinueOnError),
		"metric":       flag.NewFlagSet("metric", flag.ContinueOnError),
		"router":       flag.NewFlagSet("router", flag.ContinueOnError),
		"modifier":     flag.NewFlagSet("modifier", flag.ContinueOnError),
	}
	flagArgs := map[string][]string{
		"core":         []string(nil),
		"upstream":     []string(nil),
		"loadbalancer": []string(nil),
		"metric":       []string(nil),
		"router":       []string(nil),
		"modifier":     []string(nil),
	}

	current := "core"
	for i := 1; i < len(os.Args); i++ {
		re := regexp.MustCompile(`^-{1,2}([a-zA-Z-]+)=?`)
		matches := re.FindStringSubmatch(os.Args[i])

		// this is not a flag, rather a value so put it onto the last known set
		if len(matches) == 0 || len(matches) == 1 {
			flagArgs[current] = append(flagArgs[current], os.Args[i])
			continue
		}

		flagName := matches[1]
		if _, ok := knownFlags[flagName]; ok {
			current = "core"
			flagArgs[current] = append(flagArgs[current], os.Args[i])
			continue
		}

		// flagName is going to be something like local-loadbalancer-something or upstream-config
		current = strings.SplitN(flagName, "-", 2)[0]
		if _, ok := flagSets[current]; !ok {
			flagArgs["core"] = append(flagArgs["core"], os.Args[i])
			continue
		}

		flagArgs[current] = append(flagArgs[current], os.Args[i])

		// if this value has an equal, then treat it as a string flag
		if len(strings.Split(os.Args[i], "=")) > 1 {
			flagSets[current].String(flagName, "", fmt.Sprintf("plugin-type: %s config", current))
			// otherwise, if there is a directly after this item, treat it as a string flag
		} else if i < len(os.Args)-1 && !strings.HasPrefix(os.Args[i+1], "-") {
			flagSets[current].String(flagName, "", fmt.Sprintf("plugin-type: %s config", current))
		} else {
			flagSets[current].Bool(flagName, true, fmt.Sprintf("plugin-type: %s bool config", current))
		}
	}

	for name, flagSet := range flagSets {
		if err := flagSet.Parse(flagArgs[name]); err != nil {
			if name == "core" {
				return err
			} else {
				return errors.New(fmt.Sprintf("invalid plugin configuration for ", name, " err: ", err))
			}
		}
	}

	options.UpstreamPlugins = strings.Split(*upstreamPlugins, ",")
	options.UpstreamPluginArgs = flagSetToMap(flagSets["upstream"])

	options.MetricPlugins = strings.Split(*metricPlugins, ",")
	options.MetricPluginArgs = flagSetToMap(flagSets["metric"])

	options.ModifierPlugins = strings.Split(*modifierPlugins, ",")
	options.ModifierPluginArgs = flagSetToMap(flagSets["modifier"])

	options.LoadBalancerPlugin = *loadBalancerPlugin
	options.LoadBalancerPluginArgs = flagSetToMap(flagSets["loadbalancer"])
	options.UseLocalLoadBalancer = *useLocalLoadBalancer

	options.RouterPlugin = *routerPlugin
	options.RouterPluginArgs = flagSetToMap(flagSets["router"])
	options.UseLocalRouter = *useLocalRouter

	options.HTTPPublic = *httpPublic
	options.HTTPPublicPort = *httpPublicPort
	options.HTTPInternal = *httpInternal
	options.HTTPInternalPort = *httpInternalPort
	options.HTTPSPublic = *httpsPublic
	options.HTTPSPublicPort = *httpsPublicPort
	options.HTTPSInternal = *httpsInternal
	options.HTTPSInternalPort = *httpsInternalPort
	options.DefaultProxyTimeout = *proxyTimeout
	options.PluginTimeout = *pluginTimeout

	options.MetricBufferSize = *metricBufferSize
	options.MetricFlushInterval = *metricFlushInterval

	return nil
}

func main() {
	options := &core.Options{}

	if err := parseFlags(options); err != nil {
		log.Fatal(err)
	}

	log.Println(options.UpstreamPluginArgs)

	// build the server application which manages multiple servers
	// listening on multiple ports.
	app, err := core.New(*options)
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
