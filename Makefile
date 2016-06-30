default: test

# def setup sets up the local environment
dev-setup:
	@echo "installing godeps ..."
	@GOPATH=/gopath sh -c "curl -s https://raw.githubusercontent.com/pote/gpm/v1.4.0/bin/gpm | bash"
	@echo "symlinking `pwd` to /gopath/src/github.com/jonmorehouse/gatekeeper for development"
	@mkdir -p /gopath/src/github.com/jonmorehouse
	@ln -sf `pwd` /gopath/src/github.com/jonmorehouse/

# dev builds gatekeeper locally and places the compiled binaries into ./bin as
# well as $GOPATH/bin.
build:
	@echo "build"
	@sh -c "GATEKEEPER_DEV=1 $(CURDIR)/scripts/build.sh"

