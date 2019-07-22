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

type Server interface {
	Serve() error
	Shutdown(ctx context.Context) error
}

type Launcher struct {
	ch              chan bool
	waitGroup       *sync.WaitGroup
	servers         map[string]Server
	shutdownTimeout time.Duration
}

func New() *Launcher {
	return &Launcher{
		ch:              make(chan bool),
		waitGroup:       &sync.WaitGroup{},
		servers:         map[string]Server{},
		shutdownTimeout: 1,
	}
}

func (srv *Launcher) Add(name string, server Server) {
	srv.servers[name] = server
	srv.waitGroup.Add(1)
}

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

func (srv *Launcher) Run() error {
	// Check setup
	if len(srv.servers) <= 0 {
		return errors.New("servers list is empty")
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

	// log.Println(syscall.Getpid())

	// Start servers
	srv.startServers(wg)
	wg.Wait()
	return nil
}

// func main() {
// 	srv := New()
// 	hs1 := NewHS(":8080")
// 	srv.Add("http1", hs1)
// 	hs2 := NewHS(":8080")
// 	srv.Add("http2", hs2)
// 	srv.Run()
// }
