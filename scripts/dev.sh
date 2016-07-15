#!/usr/bin/env bash
# 
# Build gatekeeper and bundled plugins. This expects that the ./setup.sh script
# has been run to a.) bootstrap dependencies and b.) symlink the locally
# checked out version of the source code into the correct GOPATH location.
#
set -e

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"


build_core () 
{
  cd $DIR

  echo "building gatekeeper core binary ..."
  go build -o $DIR/bins/gatekeeper .

  echo "symlinking gatekeeper core into \$GOPATH ..."
  ln -sf $DIR/bins/gatekeeper $GOPATH/bin/gatekeeper
}


build_plugins ()
{
  echo "building gatekeeper bundled plugins ..."
  for dir in $(ls $DIR/plugins); do
    cd $DIR/plugins/$dir

    echo "building $dir ..."
    go build -o $DIR/bins/$dir .

    echo "symlinking $dir into \$GOPATH/bin"
    ln -sf $DIR/bins/$dir $GOPATH/bin/$dir
  done
}


case "$1" in
  all)
    build_core
    build_plugins
    ;;
  plugins)
    build_plugins
    ;;
  core)
    build_core
    ;;
  *)
    echo -e "build binaries into ./bins and symlink them into \$GOPATH/bin for development"
    echo
    echo "Usage: ./dev.sh plugins|core|all"
    echo -e "\t./dev.sh plugins - build bundled plugins only"
    echo -e "\t./dev.sh core - build core only"
    echo -e "\t./dev.sh all - build bundled plugins and core binary"
esac 
