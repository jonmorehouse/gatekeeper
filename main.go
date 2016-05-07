package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	HTTP_PUBLIC   = iota
	HTTP_INTERNAL = iota
)

type Server interface {
	Run()          // blocking function which waits on the server to run and finish
	Exit()         // exit the service
	GracefulExit() //
}

type ServerConfig struct {
	Protocol   int
	HostNames  []string
	ListenPort int
}

type HTTPServer struct {
	config *ServerConfig

	// temp for debugging
	stopped bool
}

func NewHTTPServer(config *ServerConfig) Server {
	return &HTTPServer{
		config: config,
	}
}

func (h *HTTPServer) Run() {
	for {
		if h.stopped {
			break
		}

		time.Sleep(time.Second)
	}

}

func (h *HTTPServer) Exit() {
	fmt.Println("stopping")
	h.stopped = true

}

func (h *HTTPServer) GracefulExit() {
	fmt.Println("gracefully stopping")
	h.stopped = true
}

func main() {
	// by default, we accept to
	var internalHTTPPort int
	var publicHTTPPort int
	// hostNames that this service itself, listens on (optional)
	var internalHostNames []string
	var publicHostNames []string

	flag.Parse()

	flag.IntVar(&publicHTTPPort, "public-http-port", 8000, "public http listen port")
	flag.IntVar(&internalHTTPPort, "internal-http-port", 8001, "internal http listen port")
	rawPublicHostNames := flag.String("public-host-names", "", "comma delimited list of public facing host names")
	rawInternalHostNames := flag.String("internal-host-names", "", "comma delimited list of internal host names")

	// parse any configuration flags that need to be parsed
	publicHostNames = strings.Split(*rawPublicHostNames, ",")
	internalHostNames = strings.Split(*rawInternalHostNames, ",")

	// TODO add support to handle errors being passed back!
	internalHTTPConfig := ServerConfig{
		Protocol:   HTTP_INTERNAL,
		HostNames:  internalHostNames,
		ListenPort: internalHTTPPort,
	}
	internalHTTPServer := NewHTTPServer(&internalHTTPConfig)

	publicHTTPConfig := ServerConfig{
		Protocol:   HTTP_PUBLIC,
		HostNames:  publicHostNames,
		ListenPort: internalHTTPPort,
	}
	publicHTTPServer := NewHTTPServer(&publicHTTPConfig)

	servers := []Server{internalHTTPServer, publicHTTPServer}
	var wg sync.WaitGroup
	for _, server := range servers {
		wg.Add(1)
		go func(server Server) {
			server.Run()
			wg.Done()
		}(server)
	}

	// configure a signal handler which will pass along signals to each
	// running server
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	go func() {
		for {
			signal := <-signalCh
			forced := false
			switch signal {
			case syscall.SIGQUIT:
				forced = true
			default:
				forced = false
			}

			for _, server := range servers {
				if server == nil {
					continue
				}

				if forced {
					server.Exit()
				} else {
					server.GracefulExit()
				}
			}
		}
	}()

	wg.Wait()
}
