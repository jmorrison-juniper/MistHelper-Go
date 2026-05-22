// Package web contains tests for the HTTP server components.
package web

import (
	"context"           // for context.WithTimeout -- creates the shutdown test deadline
	"net/http"          // for http.MethodGet and http.StatusOK -- request and status constants
	"net/http/httptest" // for httptest.NewRequest and httptest.NewRecorder -- in-memory HTTP testing
	"strings"           // for strings.Contains -- checks for JSON field presence in the body
	"testing"           // for testing.T -- standard Go test utilities
	"time"              // for time.Second -- short deadline in the shutdown test

	"github.com/jmorrison-juniper/misthelper-go/internal/api" // for api.Config -- builds test server config
)

// TestRootHandler_ReturnsReady verifies that GET / returns 200 and JSON containing "status":"ready".
func TestRootHandler_ReturnsReady(t *testing.T) {
	t.Parallel()                                                             // Run independently of other tests
	req := httptest.NewRequest(http.MethodGet, "/", nil)                     // Build a synthetic GET / request
	rec := httptest.NewRecorder()                                            // Record the response without a real socket
	rootHandler(rec, req)                                                    // Invoke the handler directly
	if rec.Code != http.StatusOK {                                           // Verify the status code is 200
		t.Errorf("expected 200, got %d", rec.Code)                           // Fail with actual code for diagnostics
	}
	if !strings.Contains(rec.Body.String(), `"status":"ready"`) {           // Verify JSON payload contains status field
		t.Errorf("body missing status:ready, got: %s", rec.Body.String())    // Fail with actual body for diagnostics
	}
}

// TestHealthHandler_ReturnsOK verifies that GET /health returns 200 and JSON containing "status":"ok".
func TestHealthHandler_ReturnsOK(t *testing.T) {
	t.Parallel()                                                             // Run independently of other tests
	req := httptest.NewRequest(http.MethodGet, "/health", nil)               // Build a synthetic GET /health request
	rec := httptest.NewRecorder()                                            // Record the response without a real socket
	healthHandler(rec, req)                                                  // Invoke the handler directly
	if rec.Code != http.StatusOK {                                           // Verify the status code is 200
		t.Errorf("expected 200, got %d", rec.Code)                           // Fail with actual code for diagnostics
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {              // Verify JSON payload contains status field
		t.Errorf("body missing status:ok, got: %s", rec.Body.String())       // Fail with actual body for diagnostics
	}
}

// TestNewServer_NotNil verifies that NewServer returns a non-nil *Server for a valid config.
func TestNewServer_NotNil(t *testing.T) {
	t.Parallel()                                                             // Run independently of other tests
	cfg := api.Config{WebPort: 18055}                                        // Non-standard port avoids port conflicts
	srv := NewServer(cfg)                                                    // Call the constructor under test
	if srv == nil {                                                          // Nil return means construction failed silently
		t.Fatal("NewServer returned nil, expected a valid *Server")          // Fail with a clear message
	}
	if srv.server == nil {                                                   // Inner http.Server must also be configured
		t.Fatal("inner http.Server is nil, expected it to be initialized")   // Fail if inner server is missing
	}
}

// TestServer_ShutdownClean verifies that Shutdown on a non-started server returns without error.
func TestServer_ShutdownClean(t *testing.T) {
	t.Parallel()                                                                    // Run independently of other tests
	cfg := api.Config{WebPort: 18056}                                               // Port never bound -- server not started
	srv := NewServer(cfg)                                                           // Create the server without starting it
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)           // One-second deadline for the drain
	defer cancel()                                                                  // Release deadline resources after test
	err := srv.Shutdown(ctx)                                                        // Shutdown must not hang on unstarted server
	if err != nil {                                                                 // Graceful shutdown of unstarted server must succeed
		t.Errorf("Shutdown on non-started server returned error: %v", err)          // Fail with error detail for diagnostics
	}
}
