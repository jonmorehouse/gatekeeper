package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	"github.com/tylerb/graceful"
)

const (
	HTTP_PUBLIC   = iota
	HTTP_INTERNAL = iota
)

type Server interface {
	Run()          // blocking function which waits on the server to run and finish
	Exit()         // exit the server immediately
	GracefulExit() // gracefully exit from the service
}

type ServerConfig struct {
	Protocol   int
	HostNames  []string
	ListenPort int
}

type HTTPServer struct {
	config *ServerConfig
	server *graceful.Server
}

func NewHTTPServer(config *ServerConfig) Server {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.ListenPort),
		Handler:      gatekeeper.NewProxier(0),
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}
	fmt.Println(server.Addr)
	return &HTTPServer{
		config: config,
		server: &graceful.Server{
			Server:           server,
			NoSignalHandling: true,
		},
	}
}

func (h *HTTPServer) Run() {
	h.server.ListenAndServe()
}

func (h *HTTPServer) Exit() {
	fmt.Println("Exiting immediately...")
	h.server.Stop(time.Second * 0)
}

func (h *HTTPServer) GracefulExit() {
	fmt.Println("Gracefully exiting...")
	h.server.Stop(time.Second * 20)
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

	// TODO: configuration and "building" of proxy servers should be abstracted and robustness added
	internalHTTPConfig := ServerConfig{
		Protocol:   HTTP_INTERNAL,
		HostNames:  internalHostNames,
		ListenPort: internalHTTPPort,
	}
	internalHTTPServer := NewHTTPServer(&internalHTTPConfig)

	// TODO: configuration and "building" of proxy servers should be abstracted and robustness added
	publicHTTPConfig := ServerConfig{
		Protocol:   HTTP_PUBLIC,
		HostNames:  publicHostNames,
		ListenPort: publicHTTPPort,
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
