// Package server contains the Phase 1 read-only HTTP API for persisted eval outputs.
// It exposes report and event browsing only; it does not start runs or stream events.
package server

import (
	"agent-skill-eval-go/agent"
	"agent-skill-eval-go/eval"
)

// RunsResponse is returned by GET /api/runs.
type RunsResponse struct {
	Runs    []eval.RunIndexEntry `json:"runs"`
	Skipped []eval.RunIndexError `json:"skipped,omitempty"`
}

// ErrorResponse is the common JSON error payload.
type ErrorResponse struct {
	Error string `json:"error"`
}

// EventsResponse is returned by case events endpoints.
type EventsResponse struct {
	Events []agent.Event `json:"events"`
}
