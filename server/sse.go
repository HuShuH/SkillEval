// Package server contains the Phase 1 read-only HTTP API and minimal SSE stream support.
// It exposes report/event browsing and live event streaming only; it does not start jobs itself.
package server

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"agent-skill-eval-go/agent"
)

// RunStreamEvent is the JSON payload sent over SSE.
type RunStreamEvent struct {
	RunID     string      `json:"run_id"`
	CaseID    string      `json:"case_id,omitempty"`
	Side      string      `json:"side,omitempty"`
	EventType string      `json:"event_type"`
	Event     agent.Event `json:"event"`
	Timestamp time.Time   `json:"timestamp"`
}

// SSEHub manages live subscribers for run event streams.
type SSEHub struct {
	mu      sync.Mutex
	streams map[string]*runStream
}

type runStream struct {
	subscribers map[chan RunStreamEvent]struct{}
	closed      bool
}

// NewSSEHub creates an empty hub.
func NewSSEHub() *SSEHub {
	return &SSEHub{streams: make(map[string]*runStream)}
}

// StartRun marks a run as live and subscribable.
func (h *SSEHub) StartRun(runID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.streams[runID] = &runStream{subscribers: make(map[chan RunStreamEvent]struct{})}
}

// Publish broadcasts one event to current subscribers.
func (h *SSEHub) Publish(event RunStreamEvent) {
	h.mu.Lock()
	stream := h.streams[event.RunID]
	if stream == nil || stream.closed {
		h.mu.Unlock()
		return
	}
	subscribers := make([]chan RunStreamEvent, 0, len(stream.subscribers))
	for ch := range stream.subscribers {
		subscribers = append(subscribers, ch)
	}
	h.mu.Unlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.EventType == "" {
		event.EventType = event.Event.Type
	}

	for _, ch := range subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

// CompleteRun broadcasts a completion event and closes all subscribers.
func (h *SSEHub) CompleteRun(runID string) {
	event := RunStreamEvent{
		RunID:     runID,
		EventType: "run.completed",
		Event: agent.Event{
			Type:      "run.completed",
			Timestamp: time.Now().UTC(),
			Message:   "run completed",
		},
		Timestamp: time.Now().UTC(),
	}

	h.mu.Lock()
	stream := h.streams[runID]
	if stream == nil || stream.closed {
		h.mu.Unlock()
		return
	}
	stream.closed = true
	subscribers := make([]chan RunStreamEvent, 0, len(stream.subscribers))
	for ch := range stream.subscribers {
		subscribers = append(subscribers, ch)
		delete(stream.subscribers, ch)
	}
	h.mu.Unlock()

	for _, ch := range subscribers {
		ch <- event
		close(ch)
	}
}

// Subscribe attaches a client to a live run stream.
func (h *SSEHub) Subscribe(runID string) (<-chan RunStreamEvent, func(), error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	stream := h.streams[runID]
	if stream == nil || stream.closed {
		return nil, nil, fmt.Errorf("run stream %q not found", runID)
	}

	ch := make(chan RunStreamEvent, 32)
	stream.subscribers[ch] = struct{}{}
	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		current := h.streams[runID]
		if current == nil {
			return
		}
		if _, ok := current.subscribers[ch]; ok {
			delete(current.subscribers, ch)
			close(ch)
		}
	}
	return ch, cancel, nil
}

// SubscriberCount returns the number of subscribers for tests and diagnostics.
func (h *SSEHub) SubscriberCount(runID string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	stream := h.streams[runID]
	if stream == nil {
		return 0
	}
	return len(stream.subscribers)
}

func encodeSSEData(event RunStreamEvent) ([]byte, error) {
	return json.Marshal(event)
}
