package report

import (
	"testing"

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
