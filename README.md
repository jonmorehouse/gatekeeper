# Gatekeeper
Plugin-based http(s) communications.

* issues: https://github.com/jonmorehouse/gatekeeper/issues
* current release: https://github.com/jonmorehouse/gatekeeper/releases/0.0.1


[![build status](https://travis-ci.org/jonmorehouse/gatekeeper.svg?branch=master)]() [![GitHub release](https://img.shields.io/github/release/jonmorehouse/gatekeeper.svg?maxAge=2592000)]()

**Disclaimer** this isn't real software yet! Follow along at https://github.com/jonmorehouse/gatekeeper/pulls

Gatekeeper is an http proxy that exposes aspects of an http request to plugin authors over RPC. By building hooks into the proxy lifecycle, Gatekeeper allows fine-tuned control over request routing, error handling, request modification, response modification and much more. 

## Why Gatekeeper?

By relying upon a hook based proxy cycle, `gatekeeper` leaves any and all decision making up to its users. Instead of attempting to make good decisions about aspects of the request lifecycle such as routing, discovery, authentication or tracing, `gatekeeper` focuses on making it as easy as possible for users to build their own.

Writing a plugin, is as simple as writing a few lines of `golang` and implementing a simple interface with hooks into the parent `gatekeeper` process.

## Performance

**todo**

## Use Cases

Gatekeeper doesn't make strong opinions on any part of your http stack. For instance, by exposing a simple interface for adding upstreams and backends users can integrate with Etcd, Zookeeper, Consul or any other service discovery tool (or none!).

* token authentication
* rate limiting
* traffic shaping
* canary deploys 
* custom load balancing
* service discovery integration (

# Getting Started

## Installation

`gatekeeper` is a series of binaries which are made available for download on [github](https://github.com/jonmorehouse/gatekeeper/releases).

An official `Docker` image for `Gatekeeper` can be found at [docker hub](https://hub.docker.com/r/jonmorehouse/gatekeeper/).

## Configuration

# Plugins

By writing custom plugins you can intercept and modify HTTP requests through various stages of the HTTP request. There are 5 distinct plugin types:

* `upstream plugin`: register/deregister upstreams (services) and backends (instances of a service). This is useful for integrating with a service discovery tool and implementing health checks.

* `modifier plugin`: modify requests, responses and handle error responses. Do anything from changing routes, to adding/removing headers to totally rewriting the request or response body.

* `router plugin`: route requests. If the out of the box prefix / hostname based routing isn't suitable, then a `router` can be used to add custom routing behaviour.

* `metric plugin`: consume metrics emitted from the parent process. This is particularly helpful for writing custom logs, writing statsd metrics, integrating with a request tracing framework or setting up monitoring. Every aspect of the request, internally is instrumented, recorded and emitted to the metric plugin.

* `loadbalancer plugin`: gatekeeper has the ability to proxy to a single host, but in many cases an upstream will have _many_ instances to proxy between. Load balancer plugins receive information about requests, upstreams and can expose a method to load balance between instances.

## Upstream Plugins

Building an Upstream Plugin is as simple as writing a progam implementing the following interface in `go`. The `upstream` plugin differs from other plugins because it is used to _push_ back into the parent process. Underneath the hood, `gatekeeper` exposes what is called a "manager" with methods to add/remove an upstream or backend. A plugin is responsible for calling the correct methods on that manager to communicate back to the parent process.

```go
type Plugin interface {
    // grab configuration from the parent process. Any flag with the -upstream- prefix will be passed along for configuration
    Configure(map[string]interface) error

    // start and stop are used to tell the plugin to open/close any connections and do any cleanup
    Start() error
    Stop() error

    // heartbeats are used to tell the parent process that a plugin is alive and healthy
    Heartbeat() error


    // SetManager is used to pass a manager along to the plugin so it can communicate back to the parent process. This enables an upstream or backend to be added or removed.
    SetManager(Manager) error
}

```

Out of the box, `gatekeeper` includes `static-upstreams` which is a plugin for configuring upstreams with a yaml file. It's source can be found [here](https://github.com/jonmorehouse/gatekeeper/tree/master/plugins/static-upstreams/main.go).


## Metric Plugins

Metrics are first class in `gatekeeper`; the `metric-plugin` type enables users to build cool monitoring, instrumentation, logging and request tracing experiences by giving insight into each aspect of the request and response. 

Writing a `metric-plugin` is as simple as implementing the following interface in a go program:

```go
type Plugin interface {
    // grab configuration from the parent process. Any flag with the -upstream- prefix will be passed along for configuration
    Configure() error

    // start and stop are used to tell the plugin to open/close any connections and do any cleanup
    Start() error
    Stop() error

    // heartbeats are used to tell the parent process that a plugin is alive and healthy
    Heartbeat() error

    // event metrics are used to signify events across the application
    EventMetric(*gatekeeper.EventMetric) error

    // TODO: add profiling metrics
    ProfilingMetric(*gatekeeper.ProfilingMetric) error

    // plugin metrics including plugin latencies, failures, restarts etc
    PluginMetric(*gatekeeper.PluginMetric) error

    // request metrics expose every aspect of the request to the metrics plugin. This includes the status code, request/response headers, upstream, backend and various latencies of internal processes.
    RequestMetric(*gatekeeper.RequestMetric) error

    // upstream specific metrics include upstream/backend proxy latencies, added / removed events and failures.
    UpstreamMetric(*gatekeeper.UpstreamMetric) error
}
```

Individual metric types can be found [here](https://gtihub.com/jonmorehouse/gatekeeper/tree/master/gatekeeper/metrics.go); metrics are buffered in the parent process and flushed to the plugins in batches.

Out of the box, `gatekeeper` comes with `datadog-metrics` which writes **statsd** metrics to datadog. It's source can be found [here](https://github.com/jonmorehouse/gatekeeper/tree/master/plugins/datadog-metrics).

## Modifier Plugins

`gatekeeper` allows for any aspect of a request or response to be modified by a `modifier-plugins`.

```go
type Plugin interface {
    // grab configuration from the parent process. Any flag with the -upstream- prefix will be passed along for configuration
    Configure() error

    // start and stop are used to tell the plugin to open/close any connections and do any cleanup
    Start() error
    Stop() error

    // heartbeats are used to tell the parent process that a plugin is alive and healthy
    Heartbeat() error

    // modify a request. This enables you to change headers, routes, cancel a request or change the upstream this request is proxying too.
    ModifyRequest(*gatekeeper.Request) (*gatekeeper.Request, error)

    // modify a response. The response body, headers, status code and any other component are configurable.
    ModifyResponse(*gatekeeper.Request, *gatekeeper.Response) (*gatekeeper.Response, error)

    // Whenever an error occurs, this method is called so as to allow for custom error handling or responses.
    ModifyErrorResponse(error, *gatekeeper.Request, *gatekeeper.Response) (*gatekeeper.Response, error)
}
```

Behind the scenes, `gatekeeper` creates a [request](https://github.com/jonmorehouse/gatekeeper/tree/master/gatekeeper/request.go#L19) and a [response](https://github.com/jonmorehouse/gatekeeper/tree/master/gatekeeper/response.go#43) for each request it proxies. Plugins are exposed to these and can change any part of the request or response.

## Router Plugins

Out of the box, `gatekeeper` provides a basic router that matches requests to upstreams based upon either a **prefix** or a **hostname**. Each upstream configures a set of hostnames or prefixes upon which it accepts traffic.

`router` plugins can override the routing lifecycle by implementing a plugin with the following interface:

```go
type Plugin interface {
    // grab configuration from the parent process. Any flag with the -upstream- prefix will be passed along for configuration
    Configure() error

    // start and stop are used to tell the plugin to open/close any connections and do any cleanup
    Start() error
    Stop() error

    // heartbeats are used to tell the parent process that a plugin is alive and healthy
    Heartbeat() error

    // receive new upstreams as they are emitted to the parent process
    AddUpstream(*gatekeeper.Upstream) error

    // remove an upstream, as its no longer available to the process
    RemoveUpstream(gatekeeper.UpstreamID) error

    // accept a request and determine which upstream to proxy it to
    RouteRequest(*gatekeeper.Request) (*gatekeeper.Upstream, *gatekeeper.Request, error)
}
```

## Load Balancer Plugins

The default `simple-loadbalancer` plugin load balances between upstream-backends randomly. Many use cases will require more sophisticated load balancing which is exposed via the `loadbalancer` plugin.

By implementing the following interface, a `loadbalancer` plugin can integrate custom load balancing:

```go
type Plugin interface {
    // grab configuration from the parent process. Any flag with the -upstream- prefix will be passed along for configuration
    Configure() error

    // start and stop are used to tell the plugin to open/close any connections and do any cleanup
    Start() error
    Stop() error

    // heartbeats are used to tell the parent process that a plugin is alive and healthy
    Heartbeat() error

    // 
    AddBackend(gatekeeper.UpstreamID, *gatekeeper.Backend) error

    RemoveBackend(*gatekeeper.Backend) error

    UpstreamMetric(*gatekeeper.UpstreamMetric) error

    GetBackend(gatekeeper.UpstreamID) (*gatekeeper.Backend, error)
}
```

Whenever a backend is added or removed, the `loadbalancer-plugin` is responsible for maintaining state so it can choose which backends to loadbalance. Furthermore, the parent process emits upstream metrics to the plugin so things such as proxy latency can be taken into account for load balancing.

Finally, during a request's lifecycle, the router is called to "get" a backend. The `GetBackend` method is responsible for returning a backend to proxy too.

# Contributing

# Other

