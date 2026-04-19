// Package server contains the Phase 1 read-only HTTP API for persisted eval outputs.
// It exposes report and event browsing only; it does not start runs or stream events.
package server

import (
	"net/http"
	"strings"
)

// Server is a read-only HTTP API over an eval output root.
type Server struct {
	OutputRoot string
	mux        *http.ServeMux
	Hub        *SSEHub
}

// New creates a read-only server rooted at outputRoot.
func New(outputRoot string) *Server {
	s := &Server{OutputRoot: outputRoot, Hub: NewSSEHub()}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleWeb)
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/api/runs", s.handleRuns)
	mux.HandleFunc("/api/runs/", s.handleRunSubtree)
	s.mux = mux
	return s
}

// Handler returns the HTTP handler for this server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe(addr string) error {
	if strings.TrimSpace(addr) == "" {
		addr = ":8080"
	}
	return http.ListenAndServe(addr, s.Handler())
}
