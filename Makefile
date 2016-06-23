default: test

# def setup sets up the local environment
dev_setup:
	@echo "installing godeps ..."
	@GOPATH=/gopath sh -c "curl -s https://raw.githubusercontent.com/pote/gpm/v1.4.0/bin/gpm | bash"
	@echo "symlinking `pwd` to /gopath/src/github.com/jonmorehouse/gatekeeper for development"
	@mkdir -p /gopath/src/github.com/jonmorehouse
	@ln -sf `pwd` /gopath/src/github.com/jonmorehouse/

# dev builds gatekeeper locally and places the compiled binaries into ./bin as
# well as $GOPATH/bin.
dev_build:
	@GATEKEEPER_DEV=1 sh -c "$(CURDIR)/scripts/build.sh"

# dev_run builds and runs gatekeeper locally greedily taking ports 8000 and 8001. This is just for development!
dev_run: dev
	@sh -c "$(CURDIR)/bins/gatekeeper"

# build all plugins in the plugins/ directory
plugin_dev_build:
	@GATEKEEPER_PLUGIN_DEV=1 GATEKEEPER_DEV=1 sh -c "$(CURDIR)/scripts/build.sh"
