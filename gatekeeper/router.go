package gatekeeper

import (
	"log"
	"sync"
	"time"

	router_plugin "github.com/jonmorehouse/gatekeeper/plugin/router"
	"github.com/jonmorehouse/gatekeeper/shared"
)

type RouterClient interface {
	RouteRequest(*shared.Request) (*shared.Upstream, *shared.Request, error)
}

type Router interface {
	StartStopper

	RouterClient
}

func NewLocalRouter(broadcaster Broadcaster, metricWriter MetricWriter) Router {
	return &localRouter{
		broadcaster: broadcaster,
		eventCh:     make(EventCh, 10),

		upstreams:     make(map[shared.UpstreamID]*shared.Upstream),
		prefixCache:   make(map[string]*shared.Upstream),
		hostnameCache: make(map[string]*shared.Upstream),
	}
}

type localRouter struct {
	broadcaster Broadcaster
	listenerID  ListenerID
	eventCh     EventCh

	sync.RWMutex

	upstreams     map[shared.UpstreamID]*shared.Upstream
	prefixCache   map[string]*shared.Upstream
	hostnameCache map[string]*shared.Upstream
}

func (l *localRouter) Start() error {
	go worker()
	l.listenerID = l.broadcaster.AddListener(l.eventCh, []shared.Event{shared.UpstreamAddedEvent, shared.UpstreamRemovedEvent})
}

func (l *localRouter) Stop(dur time.Duration) error {
	l.broadcaster.RemoveListener(l.listenerID)
	close(l.eventCh)
}

func (l *localRouter) RouteRequest(req *shared.Request) (*shared.Upstream, *shared.Request, error) {
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
		if event.Type() != shared.UpstreamAddedEvent && event.Type() != shared.UpstreamRemovedEvent {
			log.Println(UnsubscribedEventError)
			continue
		}

		l.Lock()
		if event.Type() == shared.UpstreamAddedEvent {
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

func NewPluginRouter(pluginManager PluginManager) Router {
	return &pluginRouter{
		pluginManager: pluginManager,
	}
}

type pluginRouter struct {
	pluginManager PluginManager
}

func (p *pluginRouter) Start() error {
	// pass AddedUpstreams along to the plugin
	p.pluginManager.AddListener(shared.UpstreamAddedEvent, func(plugin Plugin, event Event) {
		routerPlugin, ok := plugin.(router_plugin.PluginClient)
		if !ok {
			log.Println(InternalPluginError)
			return
		}

		upstreamEvent, ok := event.(*UpstreamEvent)
		if !ok {
			log.Println(InternalPluginError)
			return
		}

		if err := routerPlugin.AddUpstream(upstreamEvent.Upstream); err != nil {
			log.Println(err)
			return
		}
	})

	// pass RemovedUpstreams along to the plugin
	p.pluginManager.AddListener(shared.UpstreamRemovedEvent, func(plugin Plugin, event Event) {
		routerPlugin, ok := plugin.(router_plugin.PluginClient)
		if !ok {
			log.Println(InternalPluginError)
			return
		}

		upstreamEvent, ok := event.(*UpstreamEvent)
		if !ok {
			log.Println(InternalPluginError)
			return
		}

		if err := routerPlugin.RemoveUpstream(upstreamEvent.UpstreamID); err != nil {
			log.Println(err)
			return
		}
	})
}

func (p *pluginRouter) Stop(time.Duration) error {}

func (p *pluginRouter) RouteRequest(req *shared.Request) (*shared.Upstream, *shared.Request, error) {
	var upstream *shared.Upstream
	var err error

	callErr := p.pluginManager.Call("RouteRequest", func(plugin Plugin) error {
		routerPlugin, ok := plugin.(router_plugin.PluginClient)
		if !ok {
			shared.ProgrammingError(InternalPluginError)
			return nil
		}

		upstream, req, err = routerPlugin.RouteRequest(req)
		return err
	})

	return upstream, req, err
}
