// Package httpapi implements the OmniProxy local Control API.
//
// The Control API serves external clients (CLI, agents, scripts, MCP). The
// desktop UI does NOT call it: the UI talks to Go through Wails typed
// bindings. Both entry points must reuse the same application services
// (技术架构选择.md §13). The server binds to loopback only.
package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"omniproxy/internal/platform/profile"
	"omniproxy/internal/version"
)

// Server is the local Control API server.
type Server struct {
	prof *profile.Profile
	http *http.Server
}

// New builds the Control API router for the given profile.
func New(prof *profile.Profile) *Server {
	r := chi.NewRouter()
	r.Get("/healthz", healthz(prof))
	return &Server{
		prof: prof,
		http: &http.Server{
			Handler:           r,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// Router exposes the HTTP handler, primarily for tests.
func (s *Server) Router() http.Handler { return s.http.Handler }

// Start binds the Control API to 127.0.0.1:<control port> and serves in the
// background. It returns an error if the port is already in use.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", s.prof.ControlPort))
	if err != nil {
		return fmt.Errorf("httpapi: listen on 127.0.0.1:%d: %w", s.prof.ControlPort, err)
	}
	go func() {
		if err := s.http.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Error("httpapi: server stopped unexpectedly", "error", err)
		}
	}()
	return nil
}

// Shutdown gracefully stops the Control API.
func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("httpapi: shutdown: %w", err)
	}
	return nil
}

func healthz(prof *profile.Profile) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":      "ok",
			"version":     version.Version,
			"profile":     string(prof.Kind),
			"controlPort": prof.ControlPort,
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
