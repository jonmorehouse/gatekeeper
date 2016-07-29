package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"path/filepath"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
	"github.com/mitchellh/go-homedir"
)

type Error uint

const (
	ConfigFileRequiredError Error = iota + 1
	InvalidConfigFileError
	EmptyConfigError
	UpstreamNameRequiredError
	NoManagerErr
)

func (e Error) Error() string {
	switch e {
	case ConfigFileRequiredError:
		return "-upstream-config flag is required and must point to a valid yaml file"
	case InvalidConfigFileError:
		return "-upstream-config must be a valid yaml file"
	case EmptyConfigError:
		return "no upstreams found in -upstream-config file"
	case UpstreamNameRequiredError:
		return "name attribute required for each upstream"
	case NoManagerErr:
		return "no manager error"
	default:
	}

	log.Fatal("Programming error in static-upstreams plugin")
	return ""
}

// upstreamConfig represents an individual upstream configuration in a yaml
// file. Specifically, this exposes
type UpstreamConfig struct {
	ID        string                 `yaml:"id"`
	Name      string                 `yaml:"name"`
	Timeout   time.Duration          `yaml:"timeout"`
	Protocols []string               `yaml:"protocols"`
	Prefixes  []string               `yaml:"prefixes"`
	Hostnames []string               `yaml:"hostnames"`
	Extra     map[string]interface{} `yaml:"extra"`
	Backends  []string               `yaml:"backends"`
}

type Config map[string]UpstreamConfig

type staticUpstreams struct {
	config  Config
	manager gatekeeper.ServiceContainer
}

func (s *staticUpstreams) Configure(args map[string]interface{}) error {
	rawConfigFile, ok := args["upstream-config"]
	if !ok {
		return ConfigFileRequiredError
	}

	configFile, ok := rawConfigFile.(string)
	if !ok {
		return ConfigFileRequiredError
	}

	config, err := s.parseConfig(configFile)
	if err != nil {
		return err
	}

	if len(config) == 0 {
		return EmptyConfigError
	}

	s.config = config
	return nil
}

func (s *staticUpstreams) Heartbeat() error {
	log.Println("static-upstreams heartbeat ...")
	return nil
}

func (s *staticUpstreams) Start() error {
	if s.manager == nil {
		return NoManagerErr
	}

	if err := s.sync(); err != nil {
		return err
	}

	return nil
}

func (s *staticUpstreams) Stop() error {
	var err error

	if e := s.manager.RemoveAllUpstreams(); e != nil {
		err = e
	}

	if e := s.manager.RemoveAllBackends(); e != nil {
		err = e
	}
	return err
}

func (s *staticUpstreams) SetManager(manager upstream_plugin.Manager) error {
	s.manager = gatekeeper.NewSyncedServiceContainer(manager)
	return nil
}

func (s *staticUpstreams) UpstreamMetric(metric *gatekeeper.UpstreamMetric) error { return nil }

// loads and parses a configFile, returning a `Config` dictionary or an error
func (s *staticUpstreams) parseConfig(rawPath string) (Config, error) {
	// expand the homedirectory, if one is present in the given path
	expandedPath, err := homedir.Expand(rawPath)
	if err != nil {
		return nil, ConfigFileRequiredError
	}

	// attempt to load, read and parse the configuration file
	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return nil, ConfigFileRequiredError
	}

	rawYaml, err := ioutil.ReadFile(absPath)
	if err != nil {
		return nil, InvalidConfigFileError
	}

	var config Config
	err = yaml.Unmarshal(rawYaml, &config)
	if err != nil {
		return nil, InvalidConfigFileError
	}

	return config, nil
}

func (s *staticUpstreams) sync() error {
	for name, config := range s.config {
		rawID := config.ID
		if rawID == "" {
			rawID = fmt.Sprintf("static-upstreams:%s", name)
		}
		id := gatekeeper.UpstreamID(rawID)

		protocols, err := gatekeeper.NewProtocols(config.Protocols)
		if err != nil {
			return err
		}

		upstream := &gatekeeper.Upstream{
			ID:        id,
			Name:      name,
			Timeout:   config.Timeout,
			Protocols: protocols,
			Hostnames: config.Hostnames,
			Prefixes:  config.Prefixes,
			Extra:     config.Extra,
		}

		if err := s.manager.AddUpstream(upstream); err != nil {
			return err
		}

		for idx, address := range config.Backends {
			if _, err := url.Parse(address); err != nil {
				return err
			}

			backend := &gatekeeper.Backend{
				ID:      gatekeeper.BackendID(fmt.Sprintf("%s:backend:%d", id, idx)),
				Address: address,
				Extra:   config.Extra,
			}

			if err := s.manager.AddBackend(id, backend); err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	staticUpstreams := &staticUpstreams{}
	if err := upstream_plugin.RunPlugin("static-upstreams", staticUpstreams); err != nil {
		log.Fatal(err)
	}
}
