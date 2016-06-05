package gatekeeper

import (
	"log"
	"time"
)

type App struct {
	servers []*Server

	broadcaster         EventBroadcaster
	upstreamPlugins     []PluginManager
	loadBalancerPlugins []PluginManager
}

func New(options Options) (*App, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	broadcasters := NewUpstreamEventBroadcaster()
	upstreamPlugins

	return &App{
		servers:             servers,
		broadcaster:         broadcaster,
		upstreamPlugins:     upstreamPlugins,
		loadBalancerPlugins: loadBalancerPlugins,
	}, nil
}

func (a *App) Start() error {
	for {
		log.Println("here")
		time.Sleep(time.Second)
	}
	return nil
}

func (a *App) Stop(duration time.Duration) error {

	return nil
}
