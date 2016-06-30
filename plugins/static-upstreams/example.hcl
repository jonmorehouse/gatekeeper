variable "upstreams" {
	name = "httpbin"
	timeout = "10s"
	protocols = [ "http-internal", "http-public" ]
	prefixes = [ "httpbin" ]
	hostnames = [ "httpbin", "httpbin.org"

	dev = [ "https://httpbin.org" ]
	live = [ "https://httpbin.org"]
}
