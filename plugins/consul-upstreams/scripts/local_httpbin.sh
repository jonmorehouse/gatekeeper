#!/usr/bin/env bash

# 
# Start a local instance of httpbin running as a service in consul
# 

cleanup()
{
  docker stop gatekeeper-httpbin
  docker rm gatekeeper-httpbin
}

trap cleanup SIGINT SIGTERM

docker run \
  -p 8000:8000 \
  zoidbergwill/httpbin
