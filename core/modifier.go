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
	starter
	stopper
	ModifierClient
}

func NewLocalModifier() Modifier {
	return &localModifier{}
}

type localModifier struct{}

func (*localModifier) Start() error { return nil }
func (*localModifier) Stop() error  { return nil }

func (*localModifier) ModifyRequest(req *gatekeeper.Request) (*gatekeeper.Request, error) {
	return req, nil
}

func (*localModifier) ModifyResponse(req *gatekeeper.Request, resp *gatekeeper.Response) (*gatekeeper.Response, error) {
	return resp, nil
}

func (*localModifier) ModifyErrorResponse(err error, req *gatekeeper.Request, resp *gatekeeper.Response) (*gatekeeper.Response, error) {
	return resp, nil
}

func NewPluginModifier(pluginManagers []PluginManager) Modifier {
	return &pluginModifier{
		pluginManagers: pluginManagers,
	}
}

type pluginModifier struct {
	pluginManagers []PluginManager
}

func (r *pluginModifier) Start() error             { return nil }
func (r *pluginModifier) Stop(time.Duration) error { return nil }

func (r *pluginModifier) ModifyRequest(req *gatekeeper.Request) (*gatekeeper.Request, error) {
	var modifiedReq *gatekeeper.Request
	var err error

	for _, pluginManager := range r.pluginManagers {
		err = pluginManager.Call("ModifyRequest", func(plugin Plugin) error {
			modifierPlugin, ok := plugin.(ModifierClient)
			if !ok {
				gatekeeper.ProgrammingError("Modifier misconfigured")
				return nil
			}

			modifiedReq, err = modifierPlugin.ModifyRequest(req)
			return err
		})

		if err != nil {
			return req, err
		}
	}

	return modifiedReq, err
}

func (r *pluginModifier) ModifyResponse(req *gatekeeper.Request, resp *gatekeeper.Response) (*gatekeeper.Response, error) {
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

func (r *pluginModifier) ModifyErrorResponse(reqErr error, req *gatekeeper.Request, resp *gatekeeper.Response) (*gatekeeper.Response, error) {
	var modifiedResp *gatekeeper.Response
	var err error

	for _, pluginManager := range r.pluginManagers {
		err = pluginManager.Call("ModifyErrorResponse", func(plugin Plugin) error {
			modifierPlugin, ok := plugin.(ModifierClient)
			if !ok {
				gatekeeper.ProgrammingError("Modifier misconfigured")
				return nil
			}

			modifiedResp, err = modifierPlugin.ModifyErrorResponse(reqErr, req, resp)
			return err
		})

		if err != nil {
			return resp, err
		}
	}

	return modifiedResp, err
}
