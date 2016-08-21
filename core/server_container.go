package core

import "github.com/jonmorehouse/gatekeeper/gatekeeper"

type ServerContainer map[gatekeeper.Protocol]Server

func buildServers(options Options, router Router, loadBalancer LoadBalancer, modifier Modifier, proxier Proxier, metricWriter MetricWriter) ServerContainer {
	servers := make(ServerContainer)

	pairings := [][2]interface{}{
		[2]interface{}{gatekeeper.HTTPPublic, options.HTTPPublicPort},
		[2]interface{}{gatekeeper.HTTPInternal, options.HTTPInternalPort},
		[2]interface{}{gatekeeper.HTTPSPublic, options.HTTPSPublicPort},
		[2]interface{}{gatekeeper.HTTPSInternal, options.HTTPSInternalPort},
	}

	for _, pairing := range pairings {
		prot := pairing[0].(gatekeeper.Protocol)
		method := NewHTTPServer
		if prot == gatekeeper.HTTPSPublic || prot == gatekeeper.HTTPSInternal {
			method = NewHTTPServer
		}

		servers[prot] = method(
			prot,
			pairing[1].(uint),
			router,
			loadBalancer,
			modifier,
			proxier,
			metricWriter,
		)
	}

	return servers
}

func filterServers(servers ServerContainer, typs []gatekeeper.Protocol, cb func(Server) error) error {
	errs := NewMultiError()

	if typs == nil {
		typs = []gatekeeper.Protocol{gatekeeper.HTTPPublic, gatekeeper.HTTPInternal, gatekeeper.HTTPSPublic, gatekeeper.HTTPSInternal}
	}

	for _, typ := range typs {
		server, ok := servers[typ]
		if !ok {
			continue
		}
		errs.Add(cb(server))
	}

	return errs.ToErr()
}
