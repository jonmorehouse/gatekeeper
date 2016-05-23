package gatekeeper

type PluginOpts struct {
	Name string
	Cmd  string
	Opts map[string]interface{}
}
