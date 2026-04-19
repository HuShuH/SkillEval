// Package agent contains Phase 1 migration skeleton types for the new architecture.
// It currently provides only minimal run context state, structured events, and no persistence.
package agent

import "time"

// Event describes one runtime event emitted during a run.
type Event struct {
	Type      string         `json:"type"`
	Iteration int            `json:"iteration"`
	Timestamp time.Time      `json:"timestamp"`
	Message   string         `json:"message,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// EventSink receives emitted events.
type EventSink func(Event)

// Message describes one intermediate conversation item.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// RunContext holds the state of one run.
type RunContext struct {
	RunID     string
	Workspace string
	Iteration int
	EventSink EventSink
	Events    []Event
	Messages  []Message
	Trace     []string
}

// Emit records and forwards one event.
func (r *RunContext) Emit(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	r.Events = append(r.Events, event)
	if r.EventSink != nil {
		r.EventSink(event)
	}
}

// EmitEvent creates and records one structured event.
func (r *RunContext) EmitEvent(eventType string, iteration int, message string, metadata map[string]any) {
	r.Emit(Event{
		Type:      eventType,
		Iteration: iteration,
		Message:   message,
		Metadata:  metadata,
	})
}

// AddMessage stores one intermediate message.
func (r *RunContext) AddMessage(message Message) {
	r.Messages = append(r.Messages, message)
}

// AddTrace stores a simple trace entry.
func (r *RunContext) AddTrace(entry string) {
	r.Trace = append(r.Trace, entry)
}

// NextIteration advances the loop iteration count.
func (r *RunContext) NextIteration() int {
	r.Iteration++
	return r.Iteration
}
