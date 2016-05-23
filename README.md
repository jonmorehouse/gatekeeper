# Gatekeeper

A gatekeeper for SOA services

**Disclaimer** this isn't real software yet! Follow along at https://github.com/jonmorehouse/gatekeeper/pulls

## Goals

* routing
  - statically configurable
  - dynamically configurable with _eg: consul, etcd_

* support for multiple protocols:
  - public HTTP
  - internal - vpc HTTP
  - tcp
  - websockets

* a pluggable architecture to enable future features such as:
  - load balancing
  - token authentication
  - request decoration
  - request logging / tracing
  - rate limiting

## Terminology

upstream-plugin
  - discovers, registers and updates plugins
  - responsible for healthchecks?

balancer-plugin
  - gets hooked with a set of 

* Upstream
  - a routable upstream, for instance "httpbin"
  - has one or more backends

* Backend
  - a single instance of an upstream
* Upstream Manager
  - manages upstreams
  - plugin based

