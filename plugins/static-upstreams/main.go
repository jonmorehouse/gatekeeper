package main

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"time"

	"github.com/go-yaml/yaml"
	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
	"github.com/jonmorehouse/gatekeeper/shared"
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
	Name        string        `yaml:"name"`
	Timeout     time.Duration `yaml:"timeout"`
	Protocols   []string      `yaml:"protocols"`
	Prefixes    []string      `yaml:"prefixes"`
	Hostnames   []string      `yaml:"hostnames"`
	Backends    []string      `yaml:"backends"`
	Healthcheck string        `yaml:"healthcheck"`
}

type Config map[string]UpstreamConfig

type staticUpstreams struct {
	manager upstream_plugin.Manager
	data    []*upstreamAndBackends
}

type upstreamAndBackends struct {
	upstream *shared.Upstream
	backends []*shared.Backend
}

func (s *staticUpstreams) Configure(args map[string]interface{}) error {
	rawConfigFile, ok := args["config"]
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

	// store each upstream and its backends locally
	for _, item := range config {
		protocols, err := shared.NewProtocols(item.Protocols)
		if err != nil {
			return err
		}
		if item.Name == "" {
			return UpstreamNameRequiredError
		}

		// build out upstream object
		upstream := &shared.Upstream{
			ID:        shared.NewUpstreamID(),
			Name:      item.Name,
			Timeout:   item.Timeout,
			Prefixes:  item.Prefixes,
			Hostnames: item.Hostnames,
			Protocols: protocols,
		}

		// build out backends
		backends := make([]*shared.Backend, len(item.Backends))
		for idx, address := range item.Backends {
			backends[idx] = &shared.Backend{
				ID:          shared.NewBackendID(),
				Address:     address,
				Healthcheck: item.Healthcheck,
			}
		}

		// store all backend/upstream combinations
		s.data = append(s.data, &upstreamAndBackends{
			upstream: upstream,
			backends: backends,
		})
	}

	return nil
}

func (s *staticUpstreams) Heartbeat() error { return nil }

func (s *staticUpstreams) Start() error {
	if s.manager == nil {
		return NoManagerErr
	}

	// emit each upstream and its backends to the parent process
	for _, item := range s.data {
		if err := s.manager.AddUpstream(item.upstream); err != nil {
			return err
		}

		for _, backend := range item.backends {
			if err := s.manager.AddBackend(item.upstream.ID, backend); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *staticUpstreams) Stop() error {
	var err error

	// make a best effort to remove each upstream and its backends. If any
	// errors occur, save the error and continue
	for _, item := range s.data {
		if currentErr := s.manager.RemoveUpstream(item.upstream.ID); currentErr != nil {
			err = currentErr
		}

		for _, backend := range item.backends {
			if currentErr := s.manager.RemoveBackend(backend.ID); currentErr != nil {
				err = currentErr
			}
		}
	}

	return err
}

func (s *staticUpstreams) SetManager(manager upstream_plugin.Manager) error {
	s.manager = manager
	return nil
}

func (s *staticUpstreams) UpstreamMetric(metric *shared.UpstreamMetric) error { return nil }

// loads and parses a configFile, returning a `MultiUpstreams` dictionary or an error
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

func main() {
	staticUpstreams := &staticUpstreams{}
	if err := upstream_plugin.RunPlugin("static-upstreams", staticUpstreams); err != nil {
		log.Fatal(err)
	}
}
