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


if [[ $GATEKEEPER_DEV = "1" ]]; then
  echo "building gatekeeper in dev mode..."
  go build -o bins/gatekeeper .
else
  echo "non-dev not supported yet..."
  exit 1
fi
