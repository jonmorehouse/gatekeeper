package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"path/filepath"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	"github.com/jonmorehouse/gatekeeper/gatekeeper/utils"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"
)

// serviceDef represents an individual upstream configuration in a yaml
// file. Specifically, this exposes
type serviceDef struct {
	ID           string                 `yaml:"id"`
	Name         string                 `yaml:"name"`
	Timeout      time.Duration          `yaml:"timeout"`
	Protocols    []string               `yaml:"protocols"`
	Prefixes     []string               `yaml:"prefixes"`
	Hostnames    []string               `yaml:"hostnames"`
	Extra        map[string]interface{} `yaml:"extra"`
	Backends     []string               `yaml:"backends"`
	BackendExtra map[string]interface{} `yaml:"backend_extra"`
}

type serviceDefs map[string]serviceDef

// parseConfig accepts a configuration filepath and is responsible for parsing
// that into a set of upstream definitions. This method is responsible for
// verifying the filepath, loading the file and parsing it as JSON.
func parseConfig(rawPath string) (serviceDefs, error) {
	// expand the homedirectory, if one is present in the given path
	expandedPath, err := homedir.Expand(rawPath)
	if err != nil {
		log.Println(err)
		return nil, InvalidConfigErr
	}

	// attempt to load, read and parse the configuration file
	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		log.Println(err)
		return nil, InvalidConfigErr
	}

	rawYaml, err := ioutil.ReadFile(absPath)
	if err != nil {
		log.Println(err)
		return nil, InvalidConfigErr
	}

	var defs serviceDefs
	err = yaml.Unmarshal(rawYaml, &defs)
	if err != nil {
		log.Println(err)
		return nil, UnparseableConfigErr
	}

	return defs, nil
}

// syncServices accepts a config map with upstream definitions as well as
// ServiceContainer with which to sync the upstreams too. For each upstream and
// its backends, it is responsible for writing the correct types into
// serviceContainer; bubbling up and casting any errors where needed.
func syncServices(upstreams serviceDefs, container utils.ServiceContainer) error {
	for name, serviceDef := range upstreams {
		rawID := serviceDef.ID
		if rawID == "" {
			rawID = fmt.Sprintf("static-upstreams:%s", name)
		}
		id := gatekeeper.UpstreamID(rawID)

		protocols, err := gatekeeper.ParseProtocols(serviceDef.Protocols)
		if err != nil {
			return err
		}

		upstream := &gatekeeper.Upstream{
			ID:        id,
			Name:      name,
			Timeout:   serviceDef.Timeout,
			Protocols: protocols,
			Hostnames: serviceDef.Hostnames,
			Prefixes:  serviceDef.Prefixes,
			Extra:     serviceDef.Extra,
		}

		if err := container.AddUpstream(upstream); err != nil {
			return err
		}

		for idx, address := range serviceDef.Backends {
			if _, err := url.Parse(address); err != nil {
				return err
			}

			backend := &gatekeeper.Backend{
				ID:      gatekeeper.BackendID(fmt.Sprintf("%s:backend:%d", id, idx)),
				Address: address,
				Extra:   serviceDef.BackendExtra,
			}

			if err := container.AddBackend(id, backend); err != nil {
				return err
			}
		}
	}

	return nil
}
