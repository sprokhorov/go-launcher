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

// ErrGoroutinesListEmpty returned by Run method if there are no goroutines added.
var (
	ErrGoroutinesListEmpty = errors.New("goroutines list is empty")
)

// Goroutine describes required goroutine methods.
type Goroutine interface {
	// Id returns a goroutine id
	Id() string
	// Serve starts handle the goroutine.
	Run() error
	// Shutdown must gracefully stop the goroutine. It is also needs to
	// handle context status. For instance we must be able to pass a
	// context WithTimeout to be sure that it will be stopped properly.
	Shutdown(ctx context.Context) error
}

// Launcher manages goroutines from internal stored list.
type Launcher struct {
	ch              chan bool
	waitGroup       *sync.WaitGroup
	Goroutines      []Goroutine
	shutdownTimeout time.Duration
	ctx             context.Context
}

// New returns a new Launcher. It sets shutdownTimeout to 60 seconds by default.
func New() *Launcher {
	return &Launcher{
		ch:              make(chan bool),
		waitGroup:       &sync.WaitGroup{},
		Goroutines:      []Goroutine{},
		shutdownTimeout: 60,
		ctx:             context.Background(),
	}
}

// Add adds new goroutine to the internal goroutines list.
func (srv *Launcher) Add(Goroutine Goroutine) {
	srv.Goroutines = append(srv.Goroutines, Goroutine)
	srv.waitGroup.Add(1)
}

// SetShutdownTimeout change shutdown timeout. Default is 60 seconds.
func (srv *Launcher) SetShutdownTimeout(duration time.Duration) {
	srv.shutdownTimeout = duration
}

// Run starts all Goroutines from internal list. It will return ErrGoroutinesListEmpty
// if goroutines list is empty.
//
// Run method listens for syscalls(SIGINT, SIGTERM, SIGQUIT) and calls goroutine.Shutdown
// method.
func (srv *Launcher) Run() error {
	// Check setup
	if len(srv.Goroutines) <= 0 {
		return ErrGoroutinesListEmpty
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
				log.Printf("Got signal to stop the Goroutines, %v", sig)
				srv.stopGoroutines()
			}
		}
	}()

	// Create a wait group
	wg := &sync.WaitGroup{}
	wg.Add(len(srv.Goroutines))

	// Start goroutines
	srv.startGoroutines(wg)
	wg.Wait()
	return nil
}

// Stop terminates the goroutines. This method is needed for manual goroutines stop.
func (srv *Launcher) Stop() {
	srv.stopGoroutines()
}

// startGoroutines loops through the goroutines list in the adding order and
// starts them all.
func (srv *Launcher) startGoroutines(wg *sync.WaitGroup) {
	for _, goroutine := range srv.Goroutines {
		go func(g Goroutine) {
			log.Printf("Start goroutine %s", g.Id())
			if err := g.Run(); err != nil {
				log.Printf("Goroutine %s has been stopped or failed to start, %+v", g.Id(), err)
			}
			log.Println("Goroutine terminated successfully")
			wg.Done()
		}(goroutine)
	}
}

// stopGoroutines loops through the goroutines list and stops them all.
// If goroutine didn't stop during the srv.shutdownTimeout it will be
// killed by the system.
//
// stopGoroutines loops through the goroutines list in reverse order in case
// if the newer goroutines depend on the early created.
func (srv *Launcher) stopGoroutines() {
	ctx, cancel := context.WithTimeout(
		srv.ctx,
		srv.shutdownTimeout*time.Second,
	)
	defer cancel()

	for i := len(srv.Goroutines) - 1; i >= 0; i-- {
		go func(g Goroutine) {
			log.Printf("Trying to stop Goroutine %s", g.Id())
			if err := g.Shutdown(ctx); err != nil {
				log.Printf("Failed to stop Goroutine %s, %+v", g.Id(), err)
			}
		}(srv.Goroutines[i])
	}
}
