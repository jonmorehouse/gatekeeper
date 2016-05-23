#!/usr/bin/env bash
#
# Run test suite for gatekeeper locally.
#

# Get the parent directory of this script, the `gatekeeper` repo
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

cd gatekeeper
go test -v .
