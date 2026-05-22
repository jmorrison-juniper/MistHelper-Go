// Package web implements the HTTP web server for MistHelper-Go.
// It exposes a status endpoint and a health check for container orchestration probes.
package web

import (
	"context"  // for context.Context -- accepted by Shutdown for graceful drain
	"errors"   // for errors.Is -- distinguishes ErrServerClosed from a real bind error
	"fmt"      // for fmt.Sprintf and fmt.Errorf -- address formatting and error wrapping
	"log/slog" // for slog.Info / slog.Debug / slog.Error -- structured logging throughout
	"net/http" // for http.Server, http.ServeMux, http.HandleFunc, ResponseWriter, Request

	"github.com/jmorrison-juniper/misthelper-go/internal/api" // for api.Config -- WebPort field
)

// Server is the HTTP server listening on cfg.WebPort.
type Server struct {
	cfg    api.Config   // Runtime config carrying WebPort -- read-only after construction
	server *http.Server // Underlying net/http server -- configured in NewServer
}

// NewServer creates and configures the HTTP server (does not start listening).
// All routes are registered on a fresh ServeMux to avoid polluting the global default mux.
func NewServer(cfg api.Config) *Server {
	mux := http.NewServeMux()                     // Create an isolated mux -- avoids global state
	mux.HandleFunc("/", rootHandler)              // Register the status endpoint on GET /
	mux.HandleFunc("/health", healthHandler)      // Register the health check on GET /health
	addr := fmt.Sprintf(":%d", cfg.WebPort)       // Build the listen address from the configured port
	srv := &http.Server{Addr: addr, Handler: mux} // Wire address and mux into the underlying server
	return &Server{cfg: cfg, server: srv}         // Return the fully wired Server
}

// ListenAndServe starts the HTTP listener on cfg.WebPort and blocks until the server shuts down.
// Returns nil on graceful shutdown (ErrServerClosed); returns a wrapped error on bind failure.
func (s *Server) ListenAndServe() error {
	slog.Info("starting HTTP server", "addr", s.server.Addr)    // Log before binding the port
	err := s.server.ListenAndServe()                             // Block until shutdown or bind error
	if errors.Is(err, http.ErrServerClosed) {                   // ErrServerClosed is normal after Shutdown
		slog.Debug("HTTP server closed gracefully")              // Log normal shutdown as debug
		return nil                                               // Translate ErrServerClosed to nil
	}
	slog.Error("HTTP server stopped unexpectedly", "error", err) // Log unexpected stop with detail
	return fmt.Errorf("HTTP server: %w", err)                    // Wrap with context for diagnostics
}

// Shutdown gracefully stops the server, waiting up to ctx's deadline for in-flight requests.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down HTTP server")                       // Log before initiating the drain
	err := s.server.Shutdown(ctx)                                // Drain in-flight requests then stop listener
	if err != nil {                                              // Shutdown can fail if ctx deadline is exceeded
		return fmt.Errorf("HTTP server shutdown: %w", err)       // Wrap with context for diagnostics
	}
	slog.Debug("HTTP server shutdown complete")                  // Log after successful drain
	return nil                                                   // Shutdown completed cleanly
}

// rootHandler handles GET / and returns a JSON service status document.
func rootHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("handling request", "method", r.Method, "path", r.URL.Path)         // Log before responding
	writeJSON(w, `{"service":"misthelper-go","status":"ready","version":"dev"}`)  // Write the status body
	slog.Debug("root handler responded", "status", http.StatusOK)                 // Log after response
}

// healthHandler handles GET /health and returns a minimal JSON health document.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("handling health request", "method", r.Method, "path", r.URL.Path) // Log before responding
	writeJSON(w, `{"status":"ok"}`)                                               // Write the health body
	slog.Debug("health handler responded", "status", http.StatusOK)              // Log after response
}

// writeJSON sets the JSON Content-Type header, writes HTTP 200, and writes the body string.
// Extracted from handlers to keep them short and avoid repeating header-setting boilerplate.
func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")            // Set header before writing status
	w.WriteHeader(http.StatusOK)                                   // Explicit 200 -- must come after headers
	_, err := fmt.Fprint(w, body)                                  // Write the JSON body to the response
	if err != nil {                                                // Log failures -- cannot change response after WriteHeader
		slog.Error("failed to write JSON response", "error", err)  // Log with error detail for diagnostics
	}
}
