package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-skill-eval-go/agent"
	"agent-skill-eval-go/eval"
)

func TestHealthz(t *testing.T) {
	srv := New(t.TempDir())
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `"ok":true`) {
		t.Fatalf("unexpected body: %s", res.Body.String())
	}
}

func TestStaticWebAssets(t *testing.T) {
	srv := New(t.TempDir())

	assertStaticAsset(t, srv, "/", http.StatusOK, "text/html; charset=utf-8", "Agent Skill Eval")
	assertStaticAsset(t, srv, "/app.js", http.StatusOK, "application/javascript; charset=utf-8", "loadRuns")
	assertStaticAsset(t, srv, "/styles.css", http.StatusOK, "text/css; charset=utf-8", ".panel")
}

func TestStaticWebDoesNotBreakAPI(t *testing.T) {
	srv := New(t.TempDir())
	assertStatusAndBody(t, srv, "/api/runs", http.StatusOK, `"runs"`)
}

func TestRunsReportAndEvents(t *testing.T) {
	root := t.TempDir()
	store := eval.NewOutputStore(root, "run-1")
	report := eval.BuildRunReport([]eval.CaseResult{
		{
			CaseID:          "case-1",
			Passed:          false,
			StopReason:      "timed_out",
			Iterations:      1,
			Check:           eval.CheckResult{Checked: true, Passed: false},
			Error:           "request timed out",
			ErrorClass:      "timeout",
			FailedIteration: 1,
			Events: []agent.Event{
				{Type: "run.started", Timestamp: time.Now().UTC().Add(-2 * time.Second)},
				{Type: "provider.request.failed", Iteration: 1, Timestamp: time.Now().UTC().Add(-1 * time.Second), Metadata: map[string]any{"error_class": "timeout", "status_code": 504}},
				{Type: "provider.request.retried", Iteration: 1, Timestamp: time.Now().UTC(), Metadata: map[string]any{"attempt": 1, "next_attempt": 2}},
				{Type: "run.timed_out", Iteration: 1, Timestamp: time.Now().UTC().Add(1 * time.Second), Metadata: map[string]any{"error_class": "timeout"}},
			},
		},
	})
	if _, err := store.WriteRunReport(report); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	srv := New(root)
	assertStatusAndBody(t, srv, "/api/runs", http.StatusOK, `"run_id":"run-1"`)
	assertStatusAndBody(t, srv, "/api/runs", http.StatusOK, `"timed_out_count":1`)
	assertStatusAndBody(t, srv, "/api/runs/run-1", http.StatusOK, `"error_class":"timeout"`)
	assertStatusAndBody(t, srv, "/api/runs/run-1", http.StatusOK, `"failed_iteration":1`)
	assertStatusAndBody(t, srv, "/api/runs/run-1/summary", http.StatusOK, `"timed_out_count":1`)
	assertStatusAndBody(t, srv, "/api/runs/run-1/cases/case-1/events", http.StatusOK, `"type":"provider.request.failed"`)
	assertStatusAndBody(t, srv, "/api/runs/run-1/cases/case-1/events", http.StatusOK, `"type":"provider.request.retried"`)
	assertStatusAndBody(t, srv, "/api/runs/run-1/cases/case-1/events", http.StatusOK, `"type":"run.timed_out"`)
}

func TestPairEvents(t *testing.T) {
	root := t.TempDir()
	store := eval.NewOutputStore(root, "pair-run")
	report := eval.BuildPairReport([]eval.PairResult{
		{
			CaseID: "case-1",
			A: eval.SingleRunResult{CaseResult: eval.CaseResult{
				Passed: true,
				Events: []agent.Event{{Type: "a.event", Timestamp: time.Now().UTC()}},
			}},
			B: eval.SingleRunResult{CaseResult: eval.CaseResult{
				Passed: true,
				Events: []agent.Event{{Type: "b.event", Timestamp: time.Now().UTC()}},
			}},
			Score: eval.ScoreResult{Reason: "not_scored"},
		},
	})
	if _, err := store.WritePairReport(report); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	srv := New(root)
	assertStatusAndBody(t, srv, "/api/runs/pair-run/cases/case-1/events?side=a", http.StatusOK, `"type":"a.event"`)
	assertStatusAndBody(t, srv, "/api/runs/pair-run/cases/case-1/events?side=b", http.StatusOK, `"type":"b.event"`)
}

func TestEventsAPIIncludesDiagnosticFields(t *testing.T) {
	root := t.TempDir()
	store := eval.NewOutputStore(root, "run-1")
	report := eval.BuildRunReport([]eval.CaseResult{
		{
			CaseID:          "case-1",
			Passed:          false,
			StopReason:      "provider_error",
			Iterations:      2,
			Check:           eval.CheckResult{Checked: true, Passed: false},
			Error:           "provider failed",
			ErrorClass:      "server_error",
			FailedIteration: 2,
			Events: []agent.Event{
				{Type: "provider.request.failed", Iteration: 2, Timestamp: time.Now().UTC(), Metadata: map[string]any{"error_class": "server_error", "status_code": 502}},
				{Type: "tool.validation.failed", Iteration: 2, Timestamp: time.Now().UTC().Add(time.Second), Metadata: map[string]any{"tool": "finish", "error_class": "tool_validation_error"}},
			},
		},
	})
	if _, err := store.WriteRunReport(report); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	srv := New(root)
	assertStatusAndBody(t, srv, "/api/runs/run-1/cases/case-1/events", http.StatusOK, `"error_class":"server_error"`)
	assertStatusAndBody(t, srv, "/api/runs/run-1/cases/case-1/events", http.StatusOK, `"status_code":502`)
	assertStatusAndBody(t, srv, "/api/runs/run-1/cases/case-1/events", http.StatusOK, `"tool":"finish"`)
}

func TestStreamEndpointNotFoundAndHeaders(t *testing.T) {
	root := t.TempDir()
	srv := New(root)

	assertStatusAndBody(t, srv, "/api/runs/missing/stream", http.StatusNotFound, `"error"`)

	srv.Hub.StartRun("run-1")
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/runs/run-1/stream", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	cancel()

	srv.Handler().ServeHTTP(res, req)
	if res.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("unexpected stream content type: %q", res.Header().Get("Content-Type"))
	}
	if res.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("unexpected cache control: %q", res.Header().Get("Cache-Control"))
	}
}

func TestNotFoundAndBadFiles(t *testing.T) {
	root := t.TempDir()
	srv := New(root)
	assertStatusAndBody(t, srv, "/api/runs/missing", http.StatusNotFound, `"error"`)

	store := eval.NewOutputStore(root, "run-1")
	report := eval.BuildRunReport([]eval.CaseResult{{CaseID: "case-1", Passed: true, Events: nil}})
	if _, err := store.WriteRunReport(report); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	assertStatusAndBody(t, srv, "/api/runs/run-1/cases/missing/events", http.StatusNotFound, `"error"`)

	if err := os.WriteFile(filepath.Join(root, "run-1", "report.json"), []byte("{bad"), 0o644); err != nil {
		t.Fatalf("unexpected bad report write error: %v", err)
	}
	assertStatusAndBody(t, srv, "/api/runs/run-1", http.StatusInternalServerError, `"error"`)

	eventPath := filepath.Join(root, "run-1", "cases", "case-1", "events.jsonl")
	if err := os.MkdirAll(filepath.Dir(eventPath), 0o755); err != nil {
		t.Fatalf("unexpected mkdir error: %v", err)
	}
	if err := os.WriteFile(eventPath, []byte("{bad\n"), 0o644); err != nil {
		t.Fatalf("unexpected bad events write error: %v", err)
	}
	assertStatusAndBody(t, srv, "/api/runs/run-1/cases/case-1/events", http.StatusInternalServerError, `"error"`)
}

func TestRunsListRejectsBadReport(t *testing.T) {
	root := t.TempDir()
	goodStore := eval.NewOutputStore(root, "good-run")
	goodReport := eval.BuildRunReport([]eval.CaseResult{
		{CaseID: "case-1", Passed: true, StopReason: "finished", Iterations: 1, Check: eval.CheckResult{Checked: true, Passed: true}},
	})
	if _, err := goodStore.WriteRunReport(goodReport); err != nil {
		t.Fatalf("unexpected good report write error: %v", err)
	}

	runDir := filepath.Join(root, "bad-run")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("unexpected mkdir error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "report.json"), []byte("{bad"), 0o644); err != nil {
		t.Fatalf("unexpected bad report write error: %v", err)
	}

	srv := New(root)
	assertStatusAndBody(t, srv, "/api/runs", http.StatusOK, `"run_id":"good-run"`)
	assertStatusAndBody(t, srv, "/api/runs", http.StatusOK, `"run_id":"bad-run"`)
	assertStatusAndBody(t, srv, "/api/runs", http.StatusOK, `"skipped"`)
}

func TestRunsListFiltersByModeStatusAndLimit(t *testing.T) {
	root := t.TempDir()

	singleStore := eval.NewOutputStore(root, "single-run")
	singleReport := eval.BuildRunReport([]eval.CaseResult{
		{CaseID: "case-1", Passed: false, StopReason: "timed_out", Iterations: 1, Error: "timed out", ErrorClass: "timeout", Check: eval.CheckResult{Checked: true, Passed: false}},
	})
	singleReport.CreatedAt = "2026-04-19T10:00:00Z"
	singleReport.Metadata = map[string]string{"provider_mode": "stub", "model": "mock"}
	if _, err := singleStore.WriteRunReport(singleReport); err != nil {
		t.Fatalf("write single report: %v", err)
	}

	pairStore := eval.NewOutputStore(root, "pair-run")
	pairReport := eval.BuildPairReport([]eval.PairResult{
		{
			CaseID: "case-2",
			A:      eval.SingleRunResult{CaseResult: eval.CaseResult{Passed: true, Iterations: 1}},
			B:      eval.SingleRunResult{CaseResult: eval.CaseResult{Passed: true, Iterations: 1}},
			Score:  eval.ScoreResult{Reason: "not_scored"},
		},
	})
	pairReport.CreatedAt = "2026-04-20T10:00:00Z"
	pairReport.Metadata = map[string]string{"provider_mode": "openai", "model": "gpt-test"}
	if _, err := pairStore.WritePairReport(pairReport); err != nil {
		t.Fatalf("write pair report: %v", err)
	}

	if _, err := eval.RebuildAndWriteRunIndex(root); err != nil {
		t.Fatalf("rebuild index: %v", err)
	}

	srv := New(root)
	assertStatusAndBody(t, srv, "/api/runs?mode=pair", http.StatusOK, `"run_id":"pair-run"`)
	assertStatusAndBody(t, srv, "/api/runs?mode=pair", http.StatusOK, `"provider":"openai"`)
	assertStatusAndBody(t, srv, "/api/runs?status=timed_out", http.StatusOK, `"run_id":"single-run"`)
	assertStatusAndBody(t, srv, "/api/runs?limit=1", http.StatusOK, `"run_id":"pair-run"`)
}

func assertStatusAndBody(t *testing.T, srv *Server, target string, status int, fragment string) {
	t.Helper()
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, target, nil)

	srv.Handler().ServeHTTP(res, req)

	if res.Code != status {
		t.Fatalf("unexpected status for %s: got %d want %d body=%s", target, res.Code, status, res.Body.String())
	}
	if contentType := res.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", contentType)
	}
	if !strings.Contains(res.Body.String(), fragment) {
		t.Fatalf("expected body fragment %q for %s, got %s", fragment, target, res.Body.String())
	}

	var decoded any
	if err := json.Unmarshal(res.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("response is not json: %v", err)
	}
}

func assertStaticAsset(t *testing.T, srv *Server, target string, status int, contentType string, fragment string) {
	t.Helper()
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, target, nil)

	srv.Handler().ServeHTTP(res, req)

	if res.Code != status {
		t.Fatalf("unexpected status for %s: got %d want %d body=%s", target, res.Code, status, res.Body.String())
	}
	if got := res.Header().Get("Content-Type"); got != contentType {
		t.Fatalf("unexpected content type for %s: got %q want %q", target, got, contentType)
	}
	if !strings.Contains(res.Body.String(), fragment) {
		t.Fatalf("expected body fragment %q for %s", fragment, target)
	}
}
