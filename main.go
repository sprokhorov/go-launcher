package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Service interface {
	Serve() error
	Shutdown(ctx context.Context) error
}

type Server struct {
	ch              chan bool
	waitGroup       *sync.WaitGroup
	services        map[string]Service
	shutdownTimeout time.Duration
}

func New() *Server {
	return &Server{
		ch:              make(chan bool),
		waitGroup:       &sync.WaitGroup{},
		services:        map[string]Service{},
		shutdownTimeout: 1,
	}
}

func (srv *Server) Add(name string, service Service) {
	srv.services[name] = service
	srv.waitGroup.Add(1)
}

func (srv *Server) stopServices() {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		srv.shutdownTimeout*time.Second,
	)
	defer cancel()

	for name, service := range srv.services {
		go func(n string, s Service) {
			log.Printf("Trying to stop service %s", n)
			if err := s.Shutdown(ctx); err != nil {
				log.Printf("Failed to stop service %s, %+v", n, err)
			}
		}(name, service)
	}
}

func (srv *Server) startServices(wg *sync.WaitGroup) {
	for name, service := range srv.services {
		go func(n string, s Service) {
			log.Printf("Start service %s", n)
			if err := s.Serve(); err != nil {
				log.Printf("Service %s has been stoped or failed to start, %+v", n, err)
			}
			log.Println("Service terminated successfully")
			wg.Done()
		}(name, service)
	}
}

func (srv *Server) Run() {
	// Check setup
	if len(srv.services) <= 0 {
		log.Println("services list is empty")
		return
	}

	// Subscribe to signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	// Listen for signals in the new goroutine
	go func() {
		for {
			sig := <-sigCh
			switch sig {
			default:
				log.Printf("Got signal to stop server, %v", sig)
				srv.stopServices()
			}
		}
	}()

	// Create wait group
	wg := &sync.WaitGroup{}
	wg.Add(len(srv.services))

	// Start services
	srv.startServices(wg)
	wg.Wait()
	return
}

type httpserver struct {
	srv  *http.Server
	addr string
}

func NewHS(port string) *httpserver {
	return &httpserver{addr: port}
}

func (hs *httpserver) Serve() error {
	hs.srv = &http.Server{}
	hs.srv.Addr = hs.addr
	return hs.srv.ListenAndServe()
}

func (hs *httpserver) Shutdown(ctx context.Context) error {
	time.Sleep(5 * time.Second)
	return hs.srv.Shutdown(context.Background())
}

func main() {
	srv := New()
	hs1 := NewHS(":8080")
	srv.Add("http1", hs1)
	hs2 := NewHS(":8080")
	srv.Add("http2", hs2)
	srv.Run()
}
