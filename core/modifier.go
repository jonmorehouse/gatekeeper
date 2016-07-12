package core

import (
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type ModifierClient interface {
	ModifyRequest(*gatekeeper.Request) (*gatekeeper.Request, error)
	ModifyResponse(*gatekeeper.Request, *gatekeeper.Response) (*gatekeeper.Response, error)
	ModifyErrorResponse(error, *gatekeeper.Request, *gatekeeper.Response) (*gatekeeper.Response, error)
}

type Modifier interface {
	StartStopper
	ModifierClient
}

func NewModifier(pluginManagers []PluginManager) {
	return &modifier{
		pluginManagers: pluginManagers,
	}
}

type modifier struct {
	pluginManagers []PluginManager
}

func (r *modifier) Start(time.Duration) error { return nil }
func (r *modifier) Stop(time.Duration) error  { return nil }

func (r *modifier) ModifyRequest(req *gatekeeper.Request) (*gatekeeper.Request, error) {
	var modifiedReq *gatekeeper.Request
	var err error

	for _, pluginManager := range r.pluginManagers {
		err = pluginManager.Call("ModifyRequest", func(plugin Plugin) error {
			modifierPlugin, ok := plugin.(ModifierClient)
			if !ok {
				gatekeeper.ProgrammingError("Modifier misconfigured")
				return nil
			}

			modifiedReq, err = modifierPlugin(req)
			return err
		})

		if err != nil {
			return req, err
		}
	}

	return modifiedReq, err
}

func (r *modifier) ModifyResponse(req *gatekeeper.Request, resp *gatekeeper.Response) (*gatekeeper.Response, error) {
	var modifiedResp *gatekeeper.Response
	var err error

	for _, pluginManager := range r.pluginManagers {
		err = pluginManager.Call("ModifyResponse", func(plugin Plugin) error {
			modifierPlugin, ok := plugin.(ModifierClient)
			if !ok {
				gatekeeper.ProgrammingError("Modifier misconfigured")
				return nil
			}

			modifiedResp, err = modifierPlugin.ModifyResponse(req, resp)
			return err
		})

		if err != nil {
			return resp, err
		}
	}

	return modifiedResp, err
}

func (r *modifier) ModifyErrorResponse(err error, req *gatekeeper.Request, resp *gatekeeper.Response) (*gatekeeper.Response, error) {
	var modifiedResp *gatekeeper.Response
	var err error

	for _, pluginManager := range r.pluginManagers {
		err = pluginManager.Call("ModifyErrorResponse", func(plugin Plugin) error {
			modifierPlugin, ok := plugin.(ModifierClient)
			if !ok {
				gatekeeper.ProgrammingError("Modifier misconfigured")
				return nil
			}

			modifiedResp, err = modifierPlugin.ModifyErrorResponse(err, req, resp)
			return err
		})

		if err != nil {
			return resp, err
		}
	}

	return modifiedResp, err
}
