package launcher

import (
	"context"
	"net/http"
	"syscall"
	"testing"
	"time"
)

type HTTPServer struct {
	srv  *http.Server
	addr string
}

func NewHS(port string) *HTTPServer {
	return &HTTPServer{addr: port}
}

func (hs *HTTPServer) Serve() error {
	hs.srv = &http.Server{}
	hs.srv.Addr = hs.addr

	// hs.srv.Ha
	// http.Handle("/foo", fooHandler)

	// http.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	// })

	return hs.srv.ListenAndServe()
}

func (hs *HTTPServer) Shutdown(ctx context.Context) error {
	return hs.srv.Shutdown(context.Background())
}

func TestRun(t *testing.T) {
	l := New()
	hs := NewHS(":8080")
	l.Add("http1", hs)
	go func() {
		time.Sleep(1 * time.Second)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()
	l.Run()
}
