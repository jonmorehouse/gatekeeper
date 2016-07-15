#!/usr/bin/env bash

# 
# Start a local instance of consul running with docker, printing output and
# then stopping it afterwards.
# 
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

cleanup()
{
  docker stop gatekeeper-consul-upstreams
  docker rm gatekeeper-consul-upstreams
}

trap cleanup SIGINT SIGTERM

docker run \
  -v $DIR:/config \
  -p 8500:8500 \
  -p 8400:8400 \
  --name=gatekeeper-consul-upstreams \
  consul:latest agent -dev \
  --config-file=/config/httpbin.json \
  --client=0.0.0.0 \
  --ui
