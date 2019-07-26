package launcher

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ErrServersListEmpty returned by Run method if there are no servers added.
var (
	ErrServersListEmpty = errors.New("servers list is empty")
)

// Server describes required server methods.
type Server interface {
	// Serve starts handle the server.
	Serve() error
	// Shutdown must gracefully stop the server. It is also needs to
	// handle context status. For instance we must be able to pass a
	// context WithTimeout and to be sure that it will be stoped properly.
	Shutdown(ctx context.Context) error
}

// Launcher manages Servers from internal stored list.
type Launcher struct {
	ch              chan bool
	waitGroup       *sync.WaitGroup
	servers         map[string]Server
	shutdownTimeout time.Duration
}

// New returns a new Launcher. It sets shutdownTimeout to 60 seconds by default.
func New() *Launcher {
	return &Launcher{
		ch:              make(chan bool),
		waitGroup:       &sync.WaitGroup{},
		servers:         map[string]Server{},
		shutdownTimeout: 60,
	}
}

// Add adds new server to the internal servers list.
func (srv *Launcher) Add(name string, server Server) {
	srv.servers[name] = server
	srv.waitGroup.Add(1)
}

// SetShutdownTimeout change shutdown timeout. Default is 60 seconds.
func (srv *Launcher) SetShutdownTimeout(duration time.Duration) {
	srv.shutdownTimeout = duration
}

// Run starts all servers from internal list. It will return ErrServersListEmpty
// if servers list is empty.
//
// Run method listens syscalls(SIGINT, SIGTERM, SIGQUIT) and calles Server.Shutdown
// method.
func (srv *Launcher) Run() error {
	// Check setup
	if len(srv.servers) <= 0 {
		return ErrServersListEmpty
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
				srv.stopServers()
			}
		}
	}()

	// Create wait group
	wg := &sync.WaitGroup{}
	wg.Add(len(srv.servers))

	// Start servers
	srv.startServers(wg)
	wg.Wait()
	return nil
}

// Stop terminates servers. This method needed for manual servers stop.
func (srv *Launcher) Stop() {
	srv.stopServers()
}

// stopServers loops through the servers list and stops all of those.
func (srv *Launcher) stopServers() {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		srv.shutdownTimeout*time.Second,
	)
	defer cancel()

	for name, server := range srv.servers {
		go func(n string, s Server) {
			log.Printf("Trying to stop server %s", n)
			if err := s.Shutdown(ctx); err != nil {
				log.Printf("Failed to stop server %s, %+v", n, err)
			}
		}(name, server)
	}
}

// startServers loops through the servers list and starts all of those.
func (srv *Launcher) startServers(wg *sync.WaitGroup) {
	for name, server := range srv.servers {
		go func(n string, s Server) {
			log.Printf("Start server %s", n)
			if err := s.Serve(); err != nil {
				log.Printf("Server %s has been stoped or failed to start, %+v", n, err)
			}
			log.Println("Server terminated successfully")
			wg.Done()
		}(name, server)
	}
}
