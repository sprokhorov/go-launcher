package main

import (
	"context"
	"fmt"
	"io"
	"net/http"

	launcher "github.com/sprokhorov/go-launcher"
)

type httpServer struct {
	srv  *http.Server
	addr string
}

func newHS(port string) launcher.Server {
	return &httpServer{addr: port}
}

func (hs *httpServer) Serve() error {
	// Set up handler
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)

		io.WriteString(w, fmt.Sprint("<h1>Hello world!</h1>"))
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
	hs1 := newHS(":8080")
	l.Add("http1", hs1)
	hs2 := newHS(":8081")
	l.Add("http2", hs2)
	l.Run()
}
