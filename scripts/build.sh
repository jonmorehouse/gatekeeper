#!/usr/bin/env bash
# 
# Build gatekeeper for release or local development
# This script will output binaries to the ./bin local directory
# 

# Get the parent directory of this script, which is the `gatekeeper` repo
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

cd "$DIR"

if [[ $GATEKEEPER_DEV = "1" || $GATEKEEPER_PLUGIN_DEV = "1" ]];then
  if [[ ! -d "$GOPATH/src/github.com/jonmorehouse/gatekeeper" ]]; then
    echo "symlinking current source into $GOPATH/src/github.com/jonmorehouse/gatekeeper ..."
    mkdir -p "$GOPATH/src/github.com/jonmorehouse"
    ln -sf "$DIR" "$GOPATH/src/github.com/jonmorehouse/gatekeeper"
  fi
fi

if [[ $GATEKEEPER_DEV = "1" ]]; then
  echo "building gatekeeper in dev mode..."
  go build -o bins/gatekeeper .
  ln -sf "$DIR/bins/gatekeeper" $GOPATH/bin/gatekeeper
fi

if [[ $GATEKEEPER_PLUGIN_DEV = "1" ]];then
  echo "building gatekeeper plugins in dev mode..."
  cd plugins/static-upstreams
  go build -o "$DIR/bins/plugin-static-upstreams"
  ln -sf "$DIR/bins/plugin-static-upstreams" $GOPATH/bin/plugin-static-upstreams
fi
