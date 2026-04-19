package eval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArchiveRunsDryRunAndReal(t *testing.T) {
	root := t.TempDir()
	makeRunForManage(t, root, "run-1", "2026-04-19T10:00:00Z", "finished")

	dry, err := ArchiveRuns(root, []string{"run-1"}, true)
	if err != nil {
		t.Fatalf("dry archive: %v", err)
	}
	if len(dry.Affected) != 1 || dry.Affected[0] != "run-1" {
		t.Fatalf("unexpected dry archive result: %+v", dry)
	}
	if _, err := os.Stat(filepath.Join(root, "run-1")); err != nil {
		t.Fatalf("run should still exist after dry-run: %v", err)
	}

	got, err := ArchiveRuns(root, []string{"run-1"}, false)
	if err != nil {
		t.Fatalf("real archive: %v", err)
	}
	if len(got.Affected) != 1 {
		t.Fatalf("unexpected archive result: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(root, ArchiveDirName, "run-1")); err != nil {
		t.Fatalf("archived run missing: %v", err)
	}
	index, err := BuildRunIndex(root)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}
	if len(index.Runs) != 0 {
		t.Fatalf("archived run should not remain active: %+v", index.Runs)
	}
}

func TestDeleteRunsDryRunAndReal(t *testing.T) {
	root := t.TempDir()
	makeRunForManage(t, root, "run-1", "2026-04-19T10:00:00Z", "finished")

	dry, err := DeleteRuns(root, []string{"run-1"}, true)
	if err != nil {
		t.Fatalf("dry delete: %v", err)
	}
	if len(dry.Affected) != 1 {
		t.Fatalf("unexpected dry delete result: %+v", dry)
	}
	if _, err := os.Stat(filepath.Join(root, "run-1")); err != nil {
		t.Fatalf("run should still exist after dry-run: %v", err)
	}

	got, err := DeleteRuns(root, []string{"run-1"}, false)
	if err != nil {
		t.Fatalf("real delete: %v", err)
	}
	if len(got.Affected) != 1 {
		t.Fatalf("unexpected delete result: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(root, "run-1")); !os.IsNotExist(err) {
		t.Fatalf("run should be deleted, got err=%v", err)
	}
}

func TestPruneRunsKeepN(t *testing.T) {
	root := t.TempDir()
	makeRunForManage(t, root, "run-1", "2026-04-19T10:00:00Z", "finished")
	makeRunForManage(t, root, "run-2", "2026-04-20T10:00:00Z", "finished")
	makeRunForManage(t, root, "run-3", "2026-04-21T10:00:00Z", "timed_out")

	result, err := PruneRuns(root, 1, "all", false)
	if err != nil {
		t.Fatalf("prune runs: %v", err)
	}
	if len(result.Affected) != 2 {
		t.Fatalf("unexpected prune result: %+v", result)
	}
	index, err := BuildRunIndex(root)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}
	if len(index.Runs) != 1 || index.Runs[0].RunID != "run-3" {
		t.Fatalf("unexpected remaining runs: %+v", index.Runs)
	}
}

func TestRebuildIndexAndArchiveIgnored(t *testing.T) {
	root := t.TempDir()
	makeRunForManage(t, root, "run-1", "2026-04-19T10:00:00Z", "finished")
	if err := os.MkdirAll(filepath.Join(root, ArchiveDirName, "archived-run"), 0o755); err != nil {
		t.Fatalf("mkdir archive: %v", err)
	}
	if err := RebuildIndex(root); err != nil {
		t.Fatalf("rebuild index: %v", err)
	}
	index, err := LoadRunIndex(root)
	if err != nil {
		t.Fatalf("load index: %v", err)
	}
	if len(index.Runs) != 1 || index.Runs[0].RunID != "run-1" {
		t.Fatalf("unexpected index runs: %+v", index.Runs)
	}
}

func TestManageRejectsUnsafeRunID(t *testing.T) {
	root := t.TempDir()
	result, err := DeleteRuns(root, []string{"../bad"}, true)
	if err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if result.Errors["../bad"] == "" {
		t.Fatalf("expected unsafe path error: %+v", result)
	}
}

func makeRunForManage(t *testing.T, root string, runID string, createdAt string, stopReason string) {
	t.Helper()
	store := NewOutputStore(root, runID)
	report := BuildRunReport([]CaseResult{
		{CaseID: "case-1", Passed: stopReason == "finished", StopReason: stopReason, Iterations: 1, Error: errorForStopReason(stopReason), ErrorClass: errorClassForStopReason(stopReason), Check: CheckResult{Checked: true, Passed: stopReason == "finished"}},
	})
	report.CreatedAt = createdAt
	if _, err := store.WriteRunReport(report); err != nil {
		t.Fatalf("write report %s: %v", runID, err)
	}
}

func errorForStopReason(reason string) string {
	switch reason {
	case "timed_out":
		return "timed out"
	case "finished":
		return ""
	default:
		return "failed"
	}
}

func errorClassForStopReason(reason string) string {
	switch reason {
	case "timed_out":
		return "timeout"
	case "finished":
		return ""
	default:
		return "error"
	}
}
