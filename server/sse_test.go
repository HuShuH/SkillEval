package server

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-skill-eval-go/agent"
)

func TestSSEHubBroadcastsToSubscribers(t *testing.T) {
	hub := NewSSEHub()
	hub.StartRun("run-1")

	first, cancelFirst, err := hub.Subscribe("run-1")
	if err != nil {
		t.Fatalf("unexpected first subscribe error: %v", err)
	}
	defer cancelFirst()
	second, cancelSecond, err := hub.Subscribe("run-1")
	if err != nil {
		t.Fatalf("unexpected second subscribe error: %v", err)
	}
	defer cancelSecond()

	hub.Publish(RunStreamEvent{
		RunID:     "run-1",
		CaseID:    "case-1",
		EventType: "run.started",
		Event:     agent.Event{Type: "run.started", Timestamp: time.Now().UTC()},
	})

	assertStreamEvent(t, first, "run.started")
	assertStreamEvent(t, second, "run.started")
}

func TestSSEHubCleanupOnCancel(t *testing.T) {
	hub := NewSSEHub()
	hub.StartRun("run-1")

	_, cancel, err := hub.Subscribe("run-1")
	if err != nil {
		t.Fatalf("unexpected subscribe error: %v", err)
	}
	if got := hub.SubscriberCount("run-1"); got != 1 {
		t.Fatalf("unexpected subscriber count: %d", got)
	}
	cancel()
	if got := hub.SubscriberCount("run-1"); got != 0 {
		t.Fatalf("unexpected subscriber count after cancel: %d", got)
	}
}

func TestSSEEndpointReceivesBroadcast(t *testing.T) {
	srv := New(t.TempDir())
	srv.Hub.StartRun("run-1")
	httpServer := httptest.NewServer(srv.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/api/runs/run-1/stream")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	defer response.Body.Close()

	if response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("unexpected content type: %q", response.Header.Get("Content-Type"))
	}

	srv.Hub.Publish(RunStreamEvent{
		RunID:     "run-1",
		EventType: "run.started",
		Event:     agent.Event{Type: "run.started", Timestamp: time.Now().UTC()},
	})

	scanner := bufio.NewScanner(response.Body)
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		if strings.HasPrefix(line, "data:") {
			break
		}
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "event: message") || !strings.Contains(joined, "run.started") {
		t.Fatalf("unexpected SSE payload:\n%s", joined)
	}
}

func assertStreamEvent(t *testing.T, ch <-chan RunStreamEvent, eventType string) {
	t.Helper()
	select {
	case event := <-ch:
		if event.EventType != eventType {
			t.Fatalf("unexpected event type: %q", event.EventType)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for event")
	}
}
