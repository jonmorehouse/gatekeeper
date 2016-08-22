package core

import (
	"log"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	router_plugin "github.com/jonmorehouse/gatekeeper/plugin/router"
)

type RouterClient interface {
	RouteRequest(*gatekeeper.Request) (*gatekeeper.Upstream, *gatekeeper.Request, error)
}

type Router interface {
	starter

	RouterClient
}

func NewLocalRouter(broadcaster Broadcaster, metricWriter MetricWriter) Router {
	return &localRouter{
		broadcaster: broadcaster,
		eventCh:     make(EventCh, 10),

		upstreams:     make(map[gatekeeper.UpstreamID]*gatekeeper.Upstream),
		prefixCache:   make(map[string]*gatekeeper.Upstream),
		hostnameCache: make(map[string]*gatekeeper.Upstream),

		Subscriber: NewSubscriber(broadcaster),
	}
}

type localRouter struct {
	broadcaster Broadcaster
	listenerID  ListenerID
	eventCh     EventCh

	RWMutex

	upstreams     map[gatekeeper.UpstreamID]*gatekeeper.Upstream
	prefixCache   map[string]*gatekeeper.Upstream
	hostnameCache map[string]*gatekeeper.Upstream

	Subscriber
}

func (l *localRouter) Start() error {
	l.Subscriber.AddUpstreamEventHook(gatekeeper.UpstreamAddedEvent, l.addUpstreamHook)
	l.Subscriber.AddUpstreamEventHook(gatekeeper.UpstreamRemovedEvent, l.removeUpstreamHook)
	return l.Subscriber.Start()
}

func (l *localRouter) RouteRequest(req *gatekeeper.Request) (*gatekeeper.Upstream, *gatekeeper.Request, error) {
	l.RLock()
	defer l.RUnlock()

	upstream, hit := l.prefixCache[req.Prefix]
	if hit {
		req.Path = req.PrefixlessPath
		return upstream, req, nil
	}

	upstream, hit = l.hostnameCache[req.Host]
	if hit {
		return upstream, req, nil
	}

	log.Println(l.upstreams)
	// check the upstream store for any and all matches
	for _, upstream := range l.upstreams {
		log.Println(upstream)
		if InStrList(req.Host, upstream.Hostnames) {
			l.hostnameCache[req.Host] = upstream
			return upstream, req, nil
		}

		if InStrList(req.Prefix, upstream.Prefixes) {
			l.prefixCache[req.Prefix] = upstream
			req.Path = req.PrefixlessPath
			return upstream, req, nil
		}
	}

	return nil, req, RouteNotFoundError
}

func (l *localRouter) addUpstreamHook(event *UpstreamEvent) {
	log.Println("add upstream...", event.Upstream)
	l.Lock()
	defer l.Unlock()
	l.upstreams[event.UpstreamID] = event.Upstream
}

func (l *localRouter) removeUpstreamHook(event *UpstreamEvent) {
	log.Println("remove upstream")
	l.Lock()
	defer l.Unlock()

	delete(l.upstreams, event.UpstreamID)
	for _, host := range event.Upstream.Hostnames {
		delete(l.hostnameCache, host)
	}

	for _, prefix := range event.Upstream.Prefixes {
		delete(l.prefixCache, prefix)
	}
}

func NewPluginRouter(broadcaster Broadcaster, pluginManager PluginManager) Router {
	return &pluginRouter{
		Subscriber:    NewSubscriber(broadcaster),
		pluginManager: pluginManager,
	}
}

type pluginRouter struct {
	pluginManager PluginManager
	Subscriber
}

func (p *pluginRouter) Start() error {
	p.Subscriber.AddUpstreamEventHook(gatekeeper.UpstreamAddedEvent, p.addUpstreamHook)
	p.Subscriber.AddUpstreamEventHook(gatekeeper.UpstreamRemovedEvent, p.removeUpstreamHook)
	return p.Subscriber.Start()
}

func (p *pluginRouter) RouteRequest(req *gatekeeper.Request) (*gatekeeper.Upstream, *gatekeeper.Request, error) {
	var upstream *gatekeeper.Upstream
	var err error

	callErr := p.pluginManager.Call("RouteRequest", func(plugin Plugin) error {
		routerPlugin, ok := plugin.(router_plugin.PluginClient)
		if !ok {
			gatekeeper.ProgrammingError(InternalPluginError.Error())
			return nil
		}

		upstream, req, err = routerPlugin.RouteRequest(req)
		return err
	})

	if callErr != nil {
		return nil, req, callErr
	}

	return upstream, req, err
}

func (p *pluginRouter) addUpstreamHook(event *UpstreamEvent) {
	callErr := p.pluginManager.Call("AddUpstream", func(plugin Plugin) error {
		routerPlugin, ok := plugin.(router_plugin.PluginClient)
		if !ok {
			gatekeeper.ProgrammingError(InternalPluginError.Error())
			return nil
		}

		return routerPlugin.AddUpstream(event.Upstream)
	})

	if callErr != nil {
		log.Println(callErr)
	}
}

func (p *pluginRouter) removeUpstreamHook(event *UpstreamEvent) {
	callErr := p.pluginManager.Call("RemoveUpstream", func(plugin Plugin) error {
		routerPlugin, ok := plugin.(router_plugin.PluginClient)
		if !ok {
			gatekeeper.ProgrammingError(InternalPluginError.Error())
			return nil
		}

		return routerPlugin.RemoveUpstream(event.UpstreamID)
	})

	if callErr != nil {
		log.Println(callErr)
	}
}
