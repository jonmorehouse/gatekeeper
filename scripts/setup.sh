#!/usr/bin/env bash
# 
# Install dependencies for test / build / dev and benchmarking. This runs GPM
# installing all current dependencies. Secondly, this symlinks the local
# revision into the correct $GOPATH package location.
# 
set -e

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

echo -e "installing godeps from $DIR/Godeps ..."
cd $DIR
curl -s https://raw.githubusercontent.com/pote/gpm/v1.4.0/bin/gpm | bash

echo -e "symlinking local copy into \$GOPATH/src/github.com/jonmorehouse/gatekeeper"
mkdir -p $GOPATH/src/github.com/jonmorehouse
ln -sf $DIR $GOPATH/src/github.com/jonmorehouse/
