package agent

import "testing"

func TestRunContextTracksMessagesEventsAndIterations(t *testing.T) {
	var forwarded []Event
	ctx := RunContext{
		RunID:     "run-1",
		Workspace: "/tmp/workspace",
		EventSink: func(event Event) {
			forwarded = append(forwarded, event)
		},
	}

	ctx.AddMessage(Message{Role: "user", Content: "hello"})
	ctx.AddTrace("tool:filesystem")
	ctx.Emit(Event{Type: "iteration.started", Message: "started"})
	gotIteration := ctx.NextIteration()

	if gotIteration != 1 {
		t.Fatalf("unexpected iteration: %d", gotIteration)
	}
	if len(ctx.Messages) != 1 {
		t.Fatalf("unexpected message count: %d", len(ctx.Messages))
	}
	if len(ctx.Trace) != 1 {
		t.Fatalf("unexpected trace count: %d", len(ctx.Trace))
	}
	if len(ctx.Events) != 1 || len(forwarded) != 1 {
		t.Fatalf("unexpected events tracked: ctx=%d forwarded=%d", len(ctx.Events), len(forwarded))
	}
	if ctx.Events[0].Timestamp.IsZero() {
		t.Fatalf("expected event timestamp to be populated")
	}
}
