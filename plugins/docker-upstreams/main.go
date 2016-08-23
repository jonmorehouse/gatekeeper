package main

import (
	"log"
	"time"

	"github.com/docker/engine-api/client"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	"github.com/jonmorehouse/gatekeeper/gatekeeper/utils"
	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

type config struct {
	DockerAddr       string        `flag:"-upstream-docker-addr" default:"unix:///var/run/docker.sock"`
	DockerAPIVersion string        `flag:"-upstream-docker-api-version" default:"v1.24"`
	RefreshInterval  time.Duration `flag:"-upstream-refresh-interval" default:"10s"`

	// default upstream values
	DefaultProtocols []string      `flag:"-upstream-default-protocols"`
	DefaultTimeout   time.Duration `flag:"-upstream-default-timeout"`

	// parsed protocols
	protocols []gatekeeper.Protocol
}

func newPlugin() upstream_plugin.Plugin {
	return &plugin{
		doneCh: make(chan struct{}, 1),
	}
}

type plugin struct {
	config   *config
	client   *client.Client
	services utils.ServiceContainer
	manager  upstream_plugin.Manager

	doneCh chan struct{}
}

func (p *plugin) UpstreamMetric(metric *gatekeeper.UpstreamMetric) error { return nil }
func (p *plugin) Heartbeat() error                                       { return nil }

func (p *plugin) Stop() error {
	p.doneCh <- struct{}{}
	<-p.doneCh
	return p.services.RemoveAllUpstreams()
}

func (p *plugin) SetManager(manager upstream_plugin.Manager) error {
	p.manager = manager
	p.services = utils.NewSyncedServiceContainer(manager)
	return nil
}

func (p *plugin) Configure(opts map[string]interface{}) error {
	var config config
	if err := utils.ParseConfig(opts, &config); err != nil {
		return err
	}

	protocols, err := gatekeeper.ParseProtocols(config.DefaultProtocols)
	if err != nil {
		return err
	}
	config.protocols = protocols

	p.config = &config
	return nil
}

func (p *plugin) Start() error {
	client, err := client.NewClient(p.config.DockerAddr, p.config.DockerAPIVersion, nil, nil)
	if err != nil {
		return err
	}

	p.client = client
	go p.loop()
	return nil
}

func (p *plugin) loop() {
	sync := newDockerSync(p.client, p.services, p.config.DefaultTimeout, p.config.protocols)

	timer := time.NewTimer(p.config.RefreshInterval)
	for {
		select {
		case <-timer.C:
			if err := sync.Sync(); err != nil {
				log.Println(err)
			}
		case <-p.doneCh:
			timer.Stop()
			goto done
		}
	}
done:
	close(p.doneCh)
}

func main() {
	plugin := newPlugin()
	if err := upstream_plugin.RunPlugin("docker-upstreams", plugin); err != nil {
		log.Fatal(err)
	}

}
