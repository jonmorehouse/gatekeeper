FROM gliderlabs/alpine:3.4
MAINTAINER jon morehouse <morehousej09@gmail.com>

EXPOSE 8000 8001 443 444

RUN apk add tar curl && \
	curl -L  https://github.com/jonmorehouse/gatekeeper/releases/download/v0.0.1/gatekeeper-v0.0.1.linux-amd64.go.1.6.2.tar.gz | tar zx -o /usr/local/bin/
