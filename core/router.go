package core

import (
	"log"
	"sync"
	"time"

	router_plugin "github.com/jonmorehouse/gatekeeper/plugin/router"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type RouterClient interface {
	RouteRequest(*gatekeeper.Request) (*gatekeeper.Upstream, *gatekeeper.Request, error)
}

type Router interface {
	StartStopper

	RouterClient
}

func NewLocalRouter(broadcaster Broadcaster, metricWriter MetricWriter) Router {
	return &localRouter{
		broadcaster: broadcaster,
		eventCh:     make(EventCh, 10),

		upstreams:     make(map[gatekeeper.UpstreamID]*gatekeeper.Upstream),
		prefixCache:   make(map[string]*gatekeeper.Upstream),
		hostnameCache: make(map[string]*gatekeeper.Upstream),
	}
}

type localRouter struct {
	broadcaster Broadcaster
	listenerID  ListenerID
	eventCh     EventCh

	sync.RWMutex

	upstreams     map[gatekeeper.UpstreamID]*gatekeeper.Upstream
	prefixCache   map[string]*gatekeeper.Upstream
	hostnameCache map[string]*gatekeeper.Upstream
}

func (l *localRouter) Start() error {
	go worker()
	l.listenerID = l.broadcaster.AddListener(l.eventCh, []gatekeeper.Event{gatekeeper.UpstreamAddedEvent, gatekeeper.UpstreamRemovedEvent})
}

func (l *localRouter) Stop(dur time.Duration) error {
	l.broadcaster.RemoveListener(l.listenerID)
	close(l.eventCh)
}

func (l *localRouter) RouteRequest(req *gatekeeper.Request) (*gatekeeper.Upstream, *gatekeeper.Request, error) {
	l.RLock()
	defer l.RUnlock()

	upstream, hit := l.prefixCache[req.Prefix]
	if hit {
		req.Path = req.PrefixlessPath
		return upstream, req, nil
	}

	upstream, hit = l.hostnameCache[req.Hostname]
	if hit {
		return upstream, req, nil
	}

	// check the upstream store for any and all matches
	for _, upstream := range l.upstreams {
		if InStrList(req.Hostname, upstream.Hostnames) {
			l.hostnameCache[req.Hostname] = upstream
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

func (l *localRouter) worker() {
	// for each event, grab the event and update the local cache
	for _, event := range l.eventCh {
		upstreamEvent, err := event.AsUpstreamEvent()
		if !err {
			log.Println(err)
			continue
		}
		if event.Type() != gatekeeper.UpstreamAddedEvent && event.Type() != gatekeeper.UpstreamRemovedEvent {
			log.Println(UnsubscribedEventError)
			continue
		}

		l.Lock()
		if event.Type() == gatekeeper.UpstreamAddedEvent {
			l.upstreams[upstreamEvent.UpstreamID] = upstreamEvent.Upstream
		} else {
			// clear all state
			delete(l.upstreams, upstreamEvent.UpstreamID)
			for _, prefix := range upstreamEvent.Upstream.Prefixes {
				delete(l.prefixCache, prefix)
			}
			for _, hostname := range upstreamEvent.Upstream.Hostnames {
				delete(l.hostnameCache, hostname)
			}
		}
		l.Unlock()
	}
}

func NewPluginRouter(broadcaster Broadcaster, pluginManager PluginManager) Router {
	return &pluginRouter{
		subscriber:    NewSubscriber(broadcaster),
		pluginManager: pluginManager,
	}
}

type pluginRouter struct {
	subscriber    Subscriber
	pluginManager PluginManager
}

func (p *pluginRouter) Start() error {
	p.subscriber.AddUpstreamEventHook(gatekeeper.UpstreamAddedEvent, p.addUpstreamHook)
	p.subscriber.AddUpstreamEventHook(gatekeeper.UpstreamRemovedEvent, p.removeUpstreamHook)
	return p.subscriber.Start()
}

func (p *pluginRouter) RouteRequest(req *gatekeeper.Request) (*gatekeeper.Upstream, *gatekeeper.Request, error) {
	var upstream *gatekeeper.Upstream
	var err error

	callErr := p.pluginManager.Call("RouteRequest", func(plugin Plugin) error {
		routerPlugin, ok := plugin.(router_plugin.PluginClient)
		if !ok {
			gatekeeper.ProgrammingError(InternalPluginError)
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

func (p *pluginRouter) addUpstreamHook(event *gatekeeper.UpstreamEvent) {
	callErr := p.pluginManager.Call("AddUpstream", func(plugin Plugin) error {
		routerPlugin, ok := plugin.(router_plugin.PluginClient)
		if !ok {
			gatekeeper.ProgrammingError(InternalPluginError)
			return nil
		}

		return routerPlugin.AddUpstream(event.Upstream)
	})

	if callErr != nil {
		log.Println(callErr)
	}
}

func (p *pluginRouter) removeUpstreamHook(event *gatekeeper.UpstreamEvent) {
	callErr := p.pluginManager.Call("RemoveUpstream", func(plugin Plugin) error {
		routerPlugin, ok := plugin.(router_plugin.PluginClient)
		if !ok {
			gatekeeper.ProgrammingError(InternalPluginError)
			return nil
		}

		return routerPlugin.RemoveUpstream(event.UpstreamID)
	})

	if callErr != nil {
		log.Println(callErr)

	}
}
