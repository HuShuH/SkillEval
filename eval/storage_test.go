package eval

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-skill-eval-go/agent"
)

func TestOutputStoreWriteRunReport(t *testing.T) {
	root := t.TempDir()
	store := NewOutputStore(root, "run-1")
	report := BuildRunReport([]CaseResult{
		{
			CaseID:     "case/one",
			Passed:     true,
			StopReason: "finished",
			Iterations: 1,
			Check:      CheckResult{Checked: true, Passed: true},
			Events: []agent.Event{
				{Type: "run.started", Iteration: 0, Timestamp: time.Now().UTC(), Message: "started"},
			},
		},
	})

	reportPath, err := store.WriteRunReport(report)
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if reportPath != filepath.Join(root, "run-1", ReportFileName) {
		t.Fatalf("unexpected report path: %q", reportPath)
	}

	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("unexpected report read error: %v", err)
	}
	var saved RunReport
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unexpected report json error: %v", err)
	}
	if saved.ReportID != "run-1" || saved.TotalCases != 1 {
		t.Fatalf("unexpected saved report: %+v", saved)
	}

	eventPath := filepath.Join(root, "run-1", "cases", "case_one", EventsFileName)
	assertJSONLHasEvent(t, eventPath, "run.started")
}

func TestOutputStoreWritePairReport(t *testing.T) {
	root := t.TempDir()
	store := NewOutputStore(root, "pair-run")
	report := BuildPairReport([]PairResult{
		{
			CaseID: "case-1",
			A: SingleRunResult{CaseResult: CaseResult{
				Passed: true,
				Events: []agent.Event{{Type: "a.event", Timestamp: time.Now().UTC()}},
			}},
			B: SingleRunResult{CaseResult: CaseResult{
				Passed: false,
				Events: []agent.Event{{Type: "b.event", Timestamp: time.Now().UTC()}},
			}},
			Score: ScoreResult{Scored: false, Reason: "not_scored"},
		},
	})

	reportPath, err := store.WritePairReport(report)
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("expected report file: %v", err)
	}

	assertJSONLHasEvent(t, filepath.Join(root, "pair-run", "cases", "case-1", "a", EventsFileName), "a.event")
	assertJSONLHasEvent(t, filepath.Join(root, "pair-run", "cases", "case-1", "b", EventsFileName), "b.event")
}

func TestOutputStoreGeneratesRunIDAndSanitizesPaths(t *testing.T) {
	root := t.TempDir()
	store := NewOutputStore(root, "")

	eventPath, err := store.WriteCaseEvents("../bad/case", []agent.Event{{Type: "event", Timestamp: time.Now().UTC()}})
	if err != nil {
		t.Fatalf("unexpected event write error: %v", err)
	}
	if !strings.HasPrefix(eventPath, root) {
		t.Fatalf("event path escaped root: %q", eventPath)
	}
	if strings.Contains(eventPath, "..") {
		t.Fatalf("event path contains traversal: %q", eventPath)
	}
	if _, err := os.Stat(eventPath); err != nil {
		t.Fatalf("expected event file: %v", err)
	}
}

func assertJSONLHasEvent(t *testing.T, path string, eventType string) {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open events file %q: %v", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event agent.Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("parse event line: %v", err)
		}
		if event.Type == eventType {
			return
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan events file: %v", err)
	}
	t.Fatalf("event %q not found in %q", eventType, path)
}
