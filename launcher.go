package launcher

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sprokhorov/logkit"
)

// ErrGoroutinesListEmpty returned by Run method if there are no goroutines added.
var (
	ErrGoroutinesListEmpty = errors.New("goroutines list is empty")
)

// Goroutine describes required goroutine methods.
type Goroutine interface {
	// Id returns a goroutine id
	Id() string
	// Run starts handle the goroutine.
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
	shuttingDown    bool
	ctx             context.Context
	log             logkit.Logger
}

// New returns a new Launcher. It sets shutdownTimeout to 60 seconds by default.
func New() *Launcher {
	return &Launcher{
		ch:              make(chan bool),
		waitGroup:       &sync.WaitGroup{},
		Goroutines:      []Goroutine{},
		shutdownTimeout: 60,
		ctx:             context.Background(),
		log:             &logkit.DefaultLogger{},
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

func (srv *Launcher) SetLogger(logger logkit.Logger) {
	srv.log = logger
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
				srv.shuttingDown = true
				if s, ok := sig.(syscall.Signal); ok {
					srv.log.Infof("The main process got an %s (%d) signal, stopping goroutines", signalName(s), int(s))
				}
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
	for i := 0; i <= len(srv.Goroutines)-1; i++ {
		go func(g Goroutine) {
			srv.log.Infof("Start goroutine with id %s", g.Id())
			if err := g.Run(); err != nil {
				if srv.shuttingDown {
					srv.log.Errorf("Goroutine with id %s has been terminated, %+v", g.Id(), err)
				} else {
					srv.log.Fatalf("Failed to start goroutine %s, %v", g.Id(), err)
				}
			} else {
				srv.log.Infof("Goroutine with id %s has been terminated without an error", g.Id())
			}
			wg.Done()
		}(srv.Goroutines[i])
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
			srv.log.Infof("Trying to stop goroutine with id %s", g.Id())
			if err := g.Shutdown(ctx); err != nil {
				srv.log.Errorf("Failed to stop goroutine with id %s, %+v", g.Id(), err)
			}
		}(srv.Goroutines[i])
	}
}

// signalName returns a name of the signal
func signalName(sig syscall.Signal) string {
	switch sig {
	case syscall.SIGINT:
		return "SIGINT"
	case syscall.SIGTERM:
		return "SIGTERM"
	case syscall.SIGKILL:
		return "SIGKILL"
	case syscall.SIGQUIT:
		return "SIGQUIT"
	case syscall.SIGHUP:
		return "SIGHUP"
	case syscall.SIGUSR1:
		return "SIGUSR1"
	case syscall.SIGUSR2:
		return "SIGUSR2"
	// Add other signals as needed
	default:
		return "UNKNOWN"
	}
}
