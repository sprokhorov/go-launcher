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
	id   string
	srv  *http.Server
	addr string
}

func newHS(id string, port string) Server {
	return &httpServer{id: id, addr: port}
}

func (hs *httpServer) Id() string {
	return hs.id
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
	hs := newHS("http1", ":8080")
	l.Add(hs)
	go func() {
		time.Sleep(1 * time.Second)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()
	l.Run()
}

type customServer struct {
	id       string
	shutdown chan struct{}
}

func (cs *customServer) Id() string {
	return cs.id
}

func (cs *customServer) Run() error {
	<-cs.shutdown
	log.Println("Custom server is shutdown")
	return nil
}

func (cs *customServer) Shutdown(ctx context.Context) error {
	<-ctx.Done()
	close(cs.shutdown)
	return ctx.Err()
}

func newCS(id string) Server {
	return &customServer{id: id, shutdown: make(chan struct{})}
}

func TestShutdownTimeout(t *testing.T) {
	cs := newCS("custom1")

	l := New()
	l.SetShutdownTimeout(1)
	l.Add(cs)

	go func() {
		log.Println("Stop server in 2 seconds")
		time.Sleep(2 * time.Second)
		l.Stop()
	}()

	l.Run()
}
