package gatekeeper

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
	"github.com/tylerb/graceful"
)

type Server interface {
	Start() error
	StopAccepting() error
	Stop(time.Duration) error
}

type ProxyServer struct {
	port            uint
	protocol        shared.Protocol
	upstreamMatcher UpstreamMatcherClient
	loadBalancer    LoadBalancerClient
	modifier        Modifier
	metricWriter    MetricWriter

	stopAccepting bool

	httpServer *graceful.Server
	proxier    Proxier
	errCh      chan error
}

func (s *ProxyServer) Start() error {
	s.eventMetric(shared.ServerStartedEvent)
	return s.startHTTP()
}

func (s *ProxyServer) StopAccepting() error {
	s.stopAccepting = true
	return nil
}

func (s *ProxyServer) Stop(duration time.Duration) error {
	s.eventMetric(shared.ServerStoppedEvent)
	s.httpServer.Stop(duration)
	return <-s.errCh
}

func (s *ProxyServer) startHTTP() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.httpHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	s.httpServer = &graceful.Server{
		Server:           server,
		NoSignalHandling: true,
	}

	// the errCh is responsible for emitting an error when the server fails or closes.
	errCh := make(chan error, 1)

	// start the server in a goroutine, passing any errors back to the errCh
	go func() {
		log.Println("listening on port: ", s.port)
		err := s.httpServer.ListenAndServe()
		errCh <- err
	}()

	for {
		select {
		case err := <-errCh:
			return err
		case <-time.After(time.Millisecond * 100):
			goto finished
		}
	}

finished:
	s.errCh = errCh
	return nil
}

func (s *ProxyServer) httpHandler(rw http.ResponseWriter, rawReq *http.Request) {
	start := time.Now()
	req := shared.NewRequest(rawReq, s.protocol)

	metric := &shared.RequestMetric{
		Request:        req,
		RequestStartTS: start,
	}

	s.eventMetric(shared.RequestAcceptedEvent)

	// finish the request metric, and emit it to the MetricWriter at the
	// end of this function, after the response has been written
	defer func(metric *shared.RequestMetric) {
		metric.Timestamp = time.Now()
		metric.RequestEndTS = time.Now()
		metric.Latency = time.Now().Sub(start)
		metric.InternalLatency = metric.Latency - metric.ProxyLatency
		metric.ResponseType = shared.NewResponseType(metric.Response.StatusCode)

		// emit the request metric to the MetricWriter
		s.metricWriter.RequestMetric(metric)
	}(metric)

	if s.stopAccepting {
		resp := shared.NewErrorResponse(500, ServerShuttingDownError)
		metric.Response = resp
		metric.Error = ServerShuttingDownError
		s.writeError(rw, ServerShuttingDownError, req, resp)
		return
	}

	// build a *shared.Request for this rawReq; a wrapper with additional
	// meta information around an *http.Request object
	matchStartTS := time.Now()
	upstream, matchType, err := s.upstreamMatcher.Match(req)
	if err != nil {
		resp := shared.NewErrorResponse(400, err)
		metric.Response = resp
		metric.Error = err
		s.writeError(rw, err, req, resp)
		return
	}
	metric.UpstreamMatcherLatency = time.Now().Sub(matchStartTS)

	metric.Upstream = upstream
	req.Upstream = upstream
	req.UpstreamMatchType = matchType

	// fetch a backend from the loadbalancer to proxy this request too
	loadBalancerStartTS := time.Now()
	backend, err := s.loadBalancer.GetBackend(upstream)
	if err != nil {
		resp := shared.NewErrorResponse(500, err)
		metric.Response = resp
		metric.Error = err
		s.writeError(rw, err, req, resp)
		return
	}
	metric.LoadBalancerLatency = time.Now().Sub(loadBalancerStartTS)
	metric.Backend = backend

	modifierStartTS := time.Now()
	req, err = s.modifier.ModifyRequest(req)
	if err != nil {
		resp := shared.NewErrorResponse(500, err)
		metric.Error = err
		metric.Response = resp
		s.writeError(rw, err, req, resp)
		return
	}
	metric.RequestModifierLatency = time.Now().Sub(modifierStartTS)

	if req.Error != nil {
		resp := shared.NewErrorResponse(500, err)
		metric.Error = req.Error
		metric.Response = resp
		s.writeError(rw, err, req, resp)
		return
	}

	if req.Response != nil {
		metric.Response = req.Response
		s.writeResponse(rw, req.Response)
		return
	}

	// the proxier will only return an error when it is having an internal
	// problem and was unable to even start the proxy cycle. Any sort of
	// proxy error in the proxy lifecycle is handled internally, due to the
	// coupling that is required with the internal go httputil.ReverseProxy
	// and http.Transport types
	if err := s.proxier.Proxy(rw, rawReq, req, upstream, backend, metric); err != nil {
		resp := shared.NewErrorResponse(500, err)
		metric.Response = resp
		metric.Error = err
		s.writeError(rw, err, req, resp)
		return
	}

	s.eventMetric(shared.RequestSuccessEvent)
}

// write an error response, calling the ErrorResponse handler in the modifier plugin
func (s *ProxyServer) writeError(rw http.ResponseWriter, err error, request *shared.Request, response *shared.Response) {
	response, err = s.modifier.ModifyErrorResponse(err, request, response)
	if err != nil {
		response.Body = []byte(ModifierPluginError.String())
		response.StatusCode = 500
	}

	s.eventMetric(shared.RequestErrorEvent)
	s.writeResponse(rw, response)
}

// write a *shared.Response to an http.ResponseWriter
func (s *ProxyServer) writeResponse(rw http.ResponseWriter, response *shared.Response) {
	rw.WriteHeader(response.StatusCode)

	for header, values := range response.Header {
		for _, value := range values {
			rw.Header().Set(header, value)
		}
	}

	// TODO: add metrics around this error to see where it happens in
	// practice; adding robustness once error edges have shown
	written, err := rw.Write(response.Body)
	if err != nil {
		log.Println(ResponseWriteError, err)
	} else if written != len(response.Body) {
		log.Println(ResponseWriteError)
	}
}

func (s *ProxyServer) eventMetric(event shared.MetricEvent) {
	s.metricWriter.EventMetric(&shared.EventMetric{
		Event:     event,
		Timestamp: time.Now(),
	})
}
