#!/usr/bin/env bash
#
# Run tests for gatekeeper. This script expects a bootstrapped environment such
# as ./setup.sh would provide.
# 
# Usage: ./scripts/test dir
#   ex: ./scripts/test internal
#   ex: ./scripts/test core
#   ex: ./scripts/test plugin/router
#   ex: ./scripts/test plugins/static-upstreams
# 
set -e

# Get the parent directory of this script, the `gatekeeper` repo
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

PACKAGE=$1
[ -z $PACKAGE ] && {
  echo "Usage: $(basename $0) <package>"
  exit 1
}

[ ! -d $DIR/$PACKAGE ] && {
  echo "package not found"
  exit 1
}

cd $DIR/$PACKAGE
go test . "${@:2}"
