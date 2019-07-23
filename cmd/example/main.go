package main

import (
	"context"
	"net/http"

	launcher "github.com/sprokhorov/go-launcher"
)

type HTTPServer struct {
	srv  *http.Server
	addr string
}

func NewHS(port string) launcher.Server {
	return &HTTPServer{addr: port}
}

func (hs *HTTPServer) Serve() error {
	hs.srv = &http.Server{}
	hs.srv.Addr = hs.addr

	return hs.srv.ListenAndServe()
}

func (hs *HTTPServer) Shutdown(ctx context.Context) error {
	return hs.srv.Shutdown(context.Background())
}

func main() {
	l := launcher.New()
	hs1 := NewHS(":8080")
	l.Add("http1", hs1)
	hs2 := NewHS(":8081")
	l.Add("http2", hs2)
	l.Run()
}
