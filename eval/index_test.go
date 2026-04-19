package eval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildRunIndexFromMultipleRuns(t *testing.T) {
	root := t.TempDir()

	storeA := NewOutputStore(root, "run-a")
	reportA := BuildRunReport([]CaseResult{
		{CaseID: "case-1", Passed: true, StopReason: "finished", Iterations: 1, Check: CheckResult{Checked: true, Passed: true}},
	})
	reportA.CreatedAt = "2026-04-18T10:00:00Z"
	reportA.Metadata = map[string]string{
		"provider_mode": "stub",
		"model":         "mock-model",
		"skill_a":       "skills/a",
	}
	if _, err := storeA.WriteRunReport(reportA); err != nil {
		t.Fatalf("write run a report: %v", err)
	}
	if _, err := storeA.WriteRunReportHTML(reportA); err != nil {
		t.Fatalf("write run a html: %v", err)
	}

	storeB := NewOutputStore(root, "run-b")
	reportB := BuildRunReport([]CaseResult{
		{CaseID: "case-2", Passed: false, StopReason: "timed_out", Iterations: 2, Error: "timed out", ErrorClass: "timeout", Check: CheckResult{Checked: true, Passed: false}},
	})
	reportB.CreatedAt = "2026-04-19T10:00:00Z"
	reportB.Metadata = map[string]string{
		"provider_mode": "openai",
		"model":         "gpt-test",
		"skill_a":       "skills/b",
	}
	if _, err := storeB.WriteRunReport(reportB); err != nil {
		t.Fatalf("write run b report: %v", err)
	}

	index, err := BuildRunIndex(root)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}
	if len(index.Runs) != 2 {
		t.Fatalf("unexpected runs length: %d", len(index.Runs))
	}
	if index.Runs[0].RunID != "run-b" || index.Runs[1].RunID != "run-a" {
		t.Fatalf("unexpected order: %+v", index.Runs)
	}
	if !index.Runs[1].HasHTMLReport {
		t.Fatalf("expected html report flag for run-a")
	}
	if index.Runs[0].Provider != "openai" || index.Runs[0].Model != "gpt-test" {
		t.Fatalf("unexpected metadata extraction: %+v", index.Runs[0])
	}
	if index.Runs[0].TimedOutCount != 1 {
		t.Fatalf("expected timed out count: %+v", index.Runs[0])
	}
}

func TestBuildRunIndexSkipsBadReport(t *testing.T) {
	root := t.TempDir()
	good := NewOutputStore(root, "run-good")
	report := BuildRunReport([]CaseResult{
		{CaseID: "case-1", Passed: true, StopReason: "finished", Iterations: 1, Check: CheckResult{Checked: true, Passed: true}},
	})
	if _, err := good.WriteRunReport(report); err != nil {
		t.Fatalf("write good report: %v", err)
	}

	badDir := filepath.Join(root, "run-bad")
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatalf("mkdir bad run: %v", err)
	}
	if err := os.WriteFile(filepath.Join(badDir, ReportFileName), []byte("{bad"), 0o644); err != nil {
		t.Fatalf("write bad report: %v", err)
	}

	index, err := BuildRunIndex(root)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}
	if len(index.Runs) != 1 {
		t.Fatalf("expected one good run, got %d", len(index.Runs))
	}
	if len(index.Skipped) != 1 || index.Skipped[0].RunID != "run-bad" {
		t.Fatalf("expected skipped bad run, got %+v", index.Skipped)
	}
}

func TestWriteAndLoadRunIndex(t *testing.T) {
	root := t.TempDir()
	index := RunIndex{
		GeneratedAt: "2026-04-19T10:00:00Z",
		Runs: []RunIndexEntry{
			{RunID: "run-1", CreatedAt: "2026-04-19T09:00:00Z", Mode: "single", TotalCases: 1},
		},
	}
	path, err := WriteRunIndex(root, index)
	if err != nil {
		t.Fatalf("write index: %v", err)
	}
	if filepath.Base(path) != IndexFileName {
		t.Fatalf("unexpected index path: %s", path)
	}

	loaded, err := LoadRunIndex(root)
	if err != nil {
		t.Fatalf("load index: %v", err)
	}
	if len(loaded.Runs) != 1 || loaded.Runs[0].RunID != "run-1" {
		t.Fatalf("unexpected loaded index: %+v", loaded)
	}
}
