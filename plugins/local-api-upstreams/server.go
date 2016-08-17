package main

import (
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

type Service interface {
	Router() http.Handler
}

type Options struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

var defaultOpts = &Options{
	ReadTimeout:  0,
	WriteTimeout: 0,
}

type Server interface {
	// StartOnPort starts the server on the specified port, creating and managing a listener for it
	StartOnPort(int) error

	// StartAnyhwere starts the server on _any_ available port, and returns the port to the caller
	StartAnywhere() (int, error)

	// StartListener accepts a listener and starts the service running on it
	StartListener(net.Listener) error

	// GetPort returns the port that this server is listening on
	GetPort() (int, error)

	// Stop closes the connections and rejects connections immediately
	Stop() error
}

func NewDefaultServer(service Service) Server {
	return &server{
		opts:    defaultOpts,
		service: service,
	}
}

func NewServer(service Service, opts *Options) Server {
	return &server{
		opts:    opts,
		service: service,
	}
}

type server struct {
	opts    *Options
	service Service

	listener net.Listener
}

func (s *server) StartOnPort(port int) error {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}

	return s.StartListener(listener)
}

func (s *server) StartAnywhere() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}

	port, err := GetListenerPort(listener)
	if err != nil {
		listener.Close()
		return 0, err
	}

	if err := s.StartListener(listener); err != nil {
		return 0, err
	}

	return port, nil
}

func (s *server) StartListener(listener net.Listener) error {
	//s.listener = listener

	return nil
}

func (s *server) Stop() error {
	return s.listener.Close()
}

func (s *server) run() error {
	// http.Server accepts a *net.TCPListener; cast the Listener to the
	// explicit type to create an http.Server listener.
	if _, ok := s.listener.(*net.TCPListener); ok {
		log.Panicf("programming error; invalid listener")
	}

	server := &http.Server{
		Handler: s.service.Router(),
	}

	if err := server.Serve(s.listener); err != nil {
		log.Println(err)
	}

	return nil
}

func (s *server) GetPort() (int, error) {
	return GetListenerPort(s.listener)
}
