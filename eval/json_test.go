package eval

import (
	"strings"
	"testing"
)

func TestEncodeRunReport(t *testing.T) {
	report := BuildRunReport([]CaseResult{
		{CaseID: "case-1", Passed: true, StopReason: "finished", Iterations: 1, Check: CheckResult{Checked: true, Passed: true}},
	})

	encoded, err := EncodeRunReport(report)
	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}
	output := string(encoded)
	for _, fragment := range []string{`"results"`, `"summary"`, `"case_id": "case-1"`} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected fragment %q in output:\n%s", fragment, output)
		}
	}
}

func TestEncodePairReport(t *testing.T) {
	report := BuildPairReport([]PairResult{
		{
			CaseID: "case-1",
			A:      SingleRunResult{CaseResult: CaseResult{Passed: true}},
			B:      SingleRunResult{CaseResult: CaseResult{Passed: false}},
			Score:  ScoreResult{Scored: false, Reason: "not_scored"},
		},
	})

	encoded, err := EncodePairReport(report)
	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}
	output := string(encoded)
	for _, fragment := range []string{`"results"`, `"summary"`, `"case_id": "case-1"`} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected fragment %q in output:\n%s", fragment, output)
		}
	}
}
