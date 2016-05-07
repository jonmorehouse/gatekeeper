default: test

# dev builds gatekeeper locally and places the compiled binaries into ./bin as
# well as $GOPATH/bin.
dev:
	@GATEKEEPER_DEV=1 sh -c "$(CURDIR)/scripts/build.sh"


# dev_run builds and runs gatekeeper locally greedily taking ports 8000 and 8001. This is just for development!
dev_run: dev
	@sh -c "$(CURDIR)/bins/gatekeeper"
