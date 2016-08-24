package core

import "github.com/jonmorehouse/gatekeeper/gatekeeper"

type ServerContainer map[gatekeeper.Protocol]Server

func buildServers(options Options, router Router, loadBalancer LoadBalancer, modifier Modifier, proxier Proxier, metricWriter MetricWriter) ServerContainer {
	servers := make(ServerContainer)

	pairings := [][3]interface{}{
		[3]interface{}{options.HTTPPublic, gatekeeper.HTTPPublic, options.HTTPPublicPort},
		[3]interface{}{options.HTTPInternal, gatekeeper.HTTPInternal, options.HTTPInternalPort},
		[3]interface{}{options.HTTPSPublic, gatekeeper.HTTPSPublic, options.HTTPSPublicPort},
		[3]interface{}{options.HTTPSInternal, gatekeeper.HTTPSInternal, options.HTTPSInternalPort},
	}

	for _, pairing := range pairings {
		if !pairing[0].(bool) {
			continue
		}

		prot := pairing[1].(gatekeeper.Protocol)
		method := NewHTTPServer
		if prot == gatekeeper.HTTPSPublic || prot == gatekeeper.HTTPSInternal {
			method = NewHTTPServer
		}
		servers[prot] = method(
			prot,
			pairing[2].(uint),
			router,
			loadBalancer,
			modifier,
			proxier,
			metricWriter,
		)
	}

	return servers
}

func filterServers(servers ServerContainer, typs []gatekeeper.Protocol, errorHandler ErrorHandler, cb func(Server) error) error {
	errs := NewMultiError()

	if typs == nil {
		typs = []gatekeeper.Protocol{gatekeeper.HTTPPublic, gatekeeper.HTTPInternal, gatekeeper.HTTPSPublic, gatekeeper.HTTPSInternal}
	}

	for _, typ := range typs {
		server, ok := servers[typ]
		if !ok {
			continue
		}

		if err := cb(server); err != nil {
			errs.Add(err)
			if errorHandler != ContinueOnError {
				break
			}
		}
	}

	return errs.ToErr()
}
