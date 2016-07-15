#!/usr/bin/env bash
#
# Build gatekeeper core and bundled plugins for release. Currently, we build
# for the following architectures:
#   - darwin-amd64
#   - freebsd-amd64
#   - linux-amd64
#   - windows-amd64
# 
# Each set of binaries is built and bundled into a single tarbell to be
# uploaded. Reference: https://github.com/jonmorehouse/gatekeeper/releases for
# releases and to build a new release.
#
set -e

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

PLUGINS="mysql-api-upstreams simple-loadbalancer static-upstreams"
GOVERSION=$(go version | awk '{ print $3 }')
VERSION=$1
[ -z $VERSION ] && {
    echo "Usage: $(basename $0) VERSION"
    exit 1
}


build() 
{
  GOOS=$1
  GOARCH=$2

  # build-dir/archive filename
  name="gatekeeper-$VERSION.$GOOS-$GOARCH.$GOVERSION"

  mkdir -p /tmp/$name

  echo "building gatekeeper core for $GOOS-$GOARCH ..."
  cd $DIR
  GOOS=$GOOS GOARCH=$GOARCH go build -o /tmp/$name/gatekeeper .

  for plugin in $PLUGINS; do
    echo "building $plugin plugin for $GOOS-$GOARCH ..."
    cd $DIR/plugins/mysql-api-upstreams
    GOOS=$GOOS GOARCH=$GOARCH go build -o /tmp/$name/$plugin
  done

  tar -czvf /tmp/$name.tar.gz /tmp/$name
  echo "archive built for $GOOS-$GOARCH at /tmp/$name.tar.gz"
}

echo "building darwin-amd64 release"
build darwin amd64

echo "building freebsd-amd64 release"
build freebsd amd64

echo "building linux-amd64 release"
build linux amd64

echo "building windows-amd64 release"
build windows amd64
