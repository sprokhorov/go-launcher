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
	// Id returns a server id
	Id() string
	// Serve starts handle the server.
	Run() error
	// Shutdown must gracefully stop the server. It is also needs to
	// handle context status. For instance we must be able to pass a
	// context WithTimeout to be sure that it will be stopped properly.
	Shutdown(ctx context.Context) error
}

// Launcher manages Servers from internal stored list.
type Launcher struct {
	ch              chan bool
	waitGroup       *sync.WaitGroup
	servers         []Server
	shutdownTimeout time.Duration
	ctx             context.Context
}

// New returns a new Launcher. It sets shutdownTimeout to 60 seconds by default.
func New() *Launcher {
	return &Launcher{
		ch:              make(chan bool),
		waitGroup:       &sync.WaitGroup{},
		servers:         []Server{},
		shutdownTimeout: 60,
		ctx:             context.Background(),
	}
}

// Add adds new server to the internal servers list.
func (srv *Launcher) Add(server Server) {
	srv.servers = append(srv.servers, server)
	srv.waitGroup.Add(1)
}

// SetShutdownTimeout change shutdown timeout. Default is 60 seconds.
func (srv *Launcher) SetShutdownTimeout(duration time.Duration) {
	srv.shutdownTimeout = duration
}

// Run starts all servers from internal list. It will return ErrServersListEmpty
// if servers list is empty.
//
// Run method listens for syscalls(SIGINT, SIGTERM, SIGQUIT) and calls Server.Shutdown
// method.
func (srv *Launcher) Run() error {
	// Check setup
	if len(srv.servers) <= 0 {
		return ErrServersListEmpty
	}

	// Subscribe to the signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	// Listen for signals in the new goroutine
	go func() {
		for {
			sig := <-sigCh
			switch sig {
			default:
				log.Printf("Got signal to stop the servers, %v", sig)
				srv.stopServers()
			}
		}
	}()

	// Create a wait group
	wg := &sync.WaitGroup{}
	wg.Add(len(srv.servers))

	// Start servers
	srv.startServers(wg)
	wg.Wait()
	return nil
}

// Stop terminates the servers. This method is needed for manual servers stop.
func (srv *Launcher) Stop() {
	srv.stopServers()
}

// startServers loops through the servers list in the adding order and
// starts them all.
func (srv *Launcher) startServers(wg *sync.WaitGroup) {
	for _, server := range srv.servers {
		go func(s Server) {
			log.Printf("Start server %s", s.Id())
			if err := s.Run(); err != nil {
				log.Printf("Server %s has been stopped or failed to start, %+v", s.Id(), err)
			}
			log.Println("Server terminated successfully")
			wg.Done()
		}(server)
	}
}

// stopServers loops through the servers list and stops them all.
// If server didn't stop during the srv.shutdownTimeout it will be
// killed by the system.
//
// stopServers loops through the servers list in reverse order in case
// if the newer servers depend on the early created.
func (srv *Launcher) stopServers() {
	ctx, cancel := context.WithTimeout(
		srv.ctx,
		srv.shutdownTimeout*time.Second,
	)
	defer cancel()

	for i := len(srv.servers) - 1; i >= 0; i-- {
		go func(s Server) {
			log.Printf("Trying to stop server %s", s.Id())
			if err := s.Shutdown(ctx); err != nil {
				log.Printf("Failed to stop server %s, %+v", s.Id(), err)
			}
		}(srv.servers[i])
	}
}
