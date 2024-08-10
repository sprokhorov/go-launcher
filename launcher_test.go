package launcher

import (
	"context"
	"log"
	"net/http"
	"syscall"
	"testing"
	"time"
)

type httpGoroutine struct {
	id   string
	srv  *http.Server
	addr string
}

func newHS(id string, port string) Goroutine {
	return &httpGoroutine{id: id, addr: port}
}

func (hs *httpGoroutine) Id() string {
	return hs.id
}

func (hs *httpGoroutine) Run() error {
	hs.srv = &http.Server{}
	hs.srv.Addr = hs.addr
	return hs.srv.ListenAndServe()
}

func (hs *httpGoroutine) Shutdown(ctx context.Context) error {
	return hs.srv.Shutdown(context.Background())
}

func TestRun(t *testing.T) {
	l := New()
	hs := newHS("http1", ":8080")
	l.Add(hs)
	go func() {
		time.Sleep(1 * time.Second)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()
	l.Run()
}

type customGoroutine struct {
	id       string
	shutdown chan struct{}
}

func (cs *customGoroutine) Id() string {
	return cs.id
}

func (cs *customGoroutine) Run() error {
	<-cs.shutdown
	log.Println("Custom Goroutine is shutdown")
	return nil
}

func (cs *customGoroutine) Shutdown(ctx context.Context) error {
	<-ctx.Done()
	close(cs.shutdown)
	return ctx.Err()
}

func newCS(id string) Goroutine {
	return &customGoroutine{id: id, shutdown: make(chan struct{})}
}

func TestShutdownTimeout(t *testing.T) {
	cs := newCS("custom1")

	l := New()
	l.SetShutdownTimeout(1)
	l.Add(cs)

	go func() {
		log.Println("Stop Goroutine in 2 seconds")
		time.Sleep(2 * time.Second)
		l.Stop()
	}()

	l.Run()
}
