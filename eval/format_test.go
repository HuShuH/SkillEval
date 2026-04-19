package eval

import (
	"strings"
	"testing"
)

func TestFormatRunSummaryIncludesKeyFields(t *testing.T) {
	report := BuildRunReport([]CaseResult{
		{CaseID: "case-1", Passed: true, StopReason: "finished", Iterations: 2, Check: CheckResult{Checked: true, Passed: true}},
		{CaseID: "case-2", Passed: false, StopReason: "provider_error", Iterations: 1, Check: CheckResult{Checked: false, Passed: true}, Error: "failed"},
	})

	output := FormatRunSummary(report)
	for _, fragment := range []string{"total cases: 2", "passed: 1", "failed: 1", "average iterations:", "stop reasons:"} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected fragment %q in output:\n%s", fragment, output)
		}
	}
}

func TestFormatPairSummaryIncludesKeyFields(t *testing.T) {
	report := BuildPairReport([]PairResult{
		{
			CaseID: "case-1",
			A:      SingleRunResult{CaseResult: CaseResult{Passed: true, Iterations: 2}},
			B:      SingleRunResult{CaseResult: CaseResult{Passed: false, Iterations: 1}},
			Score:  ScoreResult{Scored: false, Reason: "not_scored"},
		},
	})

	output := FormatPairSummary(report)
	for _, fragment := range []string{"total pairs: 1", "only A passed: 1", "side A:", "side B:"} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected fragment %q in output:\n%s", fragment, output)
		}
	}
}
