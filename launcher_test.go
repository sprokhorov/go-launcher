package launcher

import (
	"context"
	"log"
	"net/http"
	"syscall"
	"testing"
	"time"
)

type httpServer struct {
	srv  *http.Server
	addr string
}

func newHS(port string) Server {
	return &httpServer{addr: port}
}

func (hs *httpServer) Run() error {
	hs.srv = &http.Server{}
	hs.srv.Addr = hs.addr
	return hs.srv.ListenAndServe()
}

func (hs *httpServer) Shutdown(ctx context.Context) error {
	return hs.srv.Shutdown(context.Background())
}

func TestRun(t *testing.T) {
	l := New()
	hs := newHS(":8080")
	l.Add("http1", hs)
	go func() {
		time.Sleep(1 * time.Second)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()
	l.Run()
}

type customServer struct {
	shutdown chan struct{}
	ctx      context.Context
}

func (cs *customServer) Run() error {
	<-cs.shutdown
	log.Println("Custom server is shutdown")
	return nil
}

func (cs *customServer) Shutdown(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			close(cs.shutdown)
			return ctx.Err()
		}
	}
}

func newCS() Server {
	return &customServer{shutdown: make(chan struct{})}
}

func TestShutdownTimeout(t *testing.T) {
	cs := newCS()

	l := New()
	l.SetShutdownTimeout(1)
	l.Add("custom1", cs)

	go func() {
		log.Println("Stop server in 2 seconds")
		time.Sleep(2 * time.Second)
		l.Stop()
	}()

	l.Run()
}
