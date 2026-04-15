package report

import (
	"testing"
	"time"

	"agent-skill-eval-go/internal/spec"
)

func TestSummarizeCounts(t *testing.T) {
	results := []spec.RunResult{
		{CaseID: "case1", Passed: true},
		{CaseID: "case2", Passed: false},
		{CaseID: "case3", Passed: true},
	}

	summary := Summarize(results)
	if summary.Total != 3 {
		t.Fatalf("expected total 3, got %d", summary.Total)
	}
	if summary.Passed != 2 {
		t.Fatalf("expected passed 2, got %d", summary.Passed)
	}
	if summary.Failed != 1 {
		t.Fatalf("expected failed 1, got %d", summary.Failed)
	}
	if len(summary.Results) != 3 {
		t.Fatalf("expected 3 results in summary, got %d", len(summary.Results))
	}
}

func TestSummarizePassRate(t *testing.T) {
	results := []spec.RunResult{
		{Passed: true},
		{Passed: true},
		{Passed: false},
		{Passed: false},
	}

	summary := Summarize(results)
	if summary.PassRate != 0.5 {
		t.Fatalf("expected pass_rate 0.5, got %v", summary.PassRate)
	}
}

func TestSummarizeEmptyResultsPassRateZero(t *testing.T) {
	summary := Summarize(nil)
	if summary.Total != 0 {
		t.Fatalf("expected total 0, got %d", summary.Total)
	}
	if summary.PassRate != 0 {
		t.Fatalf("expected pass_rate 0, got %v", summary.PassRate)
	}
}

func TestSummarizeGeneratedAtRFC3339(t *testing.T) {
	summary := Summarize([]spec.RunResult{{Passed: true}})
	if summary.GeneratedAt == "" {
		t.Fatal("expected generated_at to be non-empty")
	}
	if _, err := time.Parse(time.RFC3339, summary.GeneratedAt); err != nil {
		t.Fatalf("expected generated_at to be RFC3339, got %q: %v", summary.GeneratedAt, err)
	}
}
