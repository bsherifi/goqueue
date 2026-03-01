package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/betim/goqueue/queue"
)

// Server wraps the HTTP server and its dependencies.
type Server struct {
	httpServer *http.Server
}

// NewServer creates an HTTP server with all routes registered.
func NewServer(port int, manager *queue.Manager) *Server {
	h := &Handlers{Manager: manager}
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", h.Health)
	mux.HandleFunc("/api/stats", h.Stats)
	mux.HandleFunc("/api/jobs", h.Jobs)
	mux.HandleFunc("/api/jobs/", h.Jobs)

	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: loggingMiddleware(mux),
		},
	}
}

// Start begins listening for HTTP requests. Blocks until the server stops.
func (s *Server) Start() error {
	fmt.Printf("HTTP server listening on %s\n", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server, waiting for active requests to finish.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("%s %s %s\n", r.Method, r.URL.Path, time.Since(start))
	})
}
