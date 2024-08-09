package main

import (
	"context"
	"io"
	"net/http"

	launcher "github.com/sprokhorov/go-launcher"
)

type httpServer struct {
	id   string
	srv  *http.Server
	addr string
}

func newHS(id string, port string) launcher.Server {
	return &httpServer{id: id, addr: port}
}

func (hs *httpServer) Id() string {
	return hs.id
}

func (hs *httpServer) Run() error {
	// Set up handler
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)

		io.WriteString(w, "<h1>Hello world!</h1>")
	})

	// Configure server
	hs.srv = &http.Server{}
	hs.srv.Addr = hs.addr
	hs.srv.Handler = handler

	return hs.srv.ListenAndServe()
}

func (hs *httpServer) Shutdown(ctx context.Context) error {
	return hs.srv.Shutdown(context.Background())
}

func main() {
	l := launcher.New()
	hs1 := newHS("http1", ":8080")
	l.Add(hs1)
	hs2 := newHS("http2", ":8081")
	l.Add(hs2)
	l.Run()
}
