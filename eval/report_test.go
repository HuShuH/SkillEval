package eval

import (
	"encoding/json"
	"strings"
	"testing"

	"agent-skill-eval-go/agent"
	"agent-skill-eval-go/tool"
)

func TestBuildRunReportAggregatesSummary(t *testing.T) {
	report := BuildRunReport([]CaseResult{
		{
			CaseID:         "case-1",
			Passed:         true,
			StopReason:     "finished",
			Iterations:     2,
			Check:          CheckResult{Checked: true, Passed: true},
			ToolExecutions: []agent.ToolExecutionRecord{{ToolName: "filesystem"}},
		},
		{
			CaseID:          "case-2",
			Passed:          false,
			StopReason:      "provider_error",
			Iterations:      1,
			Check:           CheckResult{Checked: false, Passed: true},
			Error:           "provider failed",
			ErrorClass:      "server_error",
			FailedIteration: 1,
		},
		{
			CaseID:     "case-3",
			Passed:     false,
			StopReason: "max_iterations",
			Iterations: 3,
			Check:      CheckResult{Checked: true, Passed: false},
			ToolExecutions: []agent.ToolExecutionRecord{
				{ToolName: "filesystem"},
				{ToolName: "finish", Result: tool.Result{Final: true}},
			},
		},
		{
			CaseID:          "case-4",
			Passed:          false,
			StopReason:      "timed_out",
			Iterations:      1,
			Check:           CheckResult{Checked: true, Passed: false},
			Error:           "request timed out",
			ErrorClass:      "timeout",
			FailedIteration: 1,
		},
		{
			CaseID:          "case-5",
			Passed:          false,
			StopReason:      "canceled",
			Iterations:      1,
			Check:           CheckResult{Checked: true, Passed: false},
			Error:           "request canceled",
			ErrorClass:      "canceled",
			FailedIteration: 1,
		},
	})

	if report.TotalCases != 5 || report.Summary.TotalCases != 5 {
		t.Fatalf("unexpected total cases: %+v", report.Summary)
	}
	if report.Summary.Passed != 1 || report.Summary.Failed != 4 {
		t.Fatalf("unexpected pass/fail counts: %+v", report.Summary)
	}
	if report.Summary.Unchecked != 1 || report.Summary.Errored != 3 {
		t.Fatalf("unexpected unchecked/errored counts: %+v", report.Summary)
	}
	if report.Summary.FinishedCount != 1 || report.Summary.MaxIterationsCount != 1 || report.Summary.ProviderErrorCount != 1 {
		t.Fatalf("unexpected stop reason counters: %+v", report.Summary)
	}
	if report.Summary.TimedOutCount != 1 || report.Summary.CanceledCount != 1 {
		t.Fatalf("unexpected timeout/canceled counters: %+v", report.Summary)
	}
	if report.Summary.ErrorClasses["server_error"] != 1 || report.Summary.ErrorClasses["timeout"] != 1 || report.Summary.ErrorClasses["canceled"] != 1 {
		t.Fatalf("unexpected error class counters: %+v", report.Summary.ErrorClasses)
	}
	if report.Summary.TotalToolCalls != 3 {
		t.Fatalf("unexpected total tool calls: %d", report.Summary.TotalToolCalls)
	}
	if report.Summary.AverageIterations != 1.6 {
		t.Fatalf("unexpected average iterations: %v", report.Summary.AverageIterations)
	}
}

func TestBuildPairReportAggregatesSummary(t *testing.T) {
	report := BuildPairReport([]PairResult{
		{
			CaseID: "case-1",
			A:      SingleRunResult{CaseResult: CaseResult{Passed: true, Iterations: 2, ToolExecutions: []agent.ToolExecutionRecord{{ToolName: "filesystem"}}}},
			B:      SingleRunResult{CaseResult: CaseResult{Passed: true, Iterations: 1}},
			Score:  ScoreResult{Scored: false, Reason: "not_scored"},
		},
		{
			CaseID: "case-2",
			A:      SingleRunResult{CaseResult: CaseResult{Passed: true, Iterations: 3}},
			B:      SingleRunResult{CaseResult: CaseResult{Passed: false, Iterations: 4, Error: "failed"}},
			Score:  ScoreResult{Scored: false},
			Error:  "run B: failed",
		},
		{
			CaseID: "case-3",
			A:      SingleRunResult{CaseResult: CaseResult{Passed: false, Iterations: 1}},
			B:      SingleRunResult{CaseResult: CaseResult{Passed: true, Iterations: 2, ToolExecutions: []agent.ToolExecutionRecord{{ToolName: "finish"}}}},
			Score:  ScoreResult{Scored: false, Reason: "not_scored"},
		},
	})

	if report.Summary.TotalPairs != 3 {
		t.Fatalf("unexpected total pairs: %+v", report.Summary)
	}
	if report.Summary.BothPassed != 1 || report.Summary.OnlyAPassed != 1 || report.Summary.OnlyBPassed != 1 {
		t.Fatalf("unexpected pass distribution: %+v", report.Summary)
	}
	if report.Summary.ErroredPairs != 1 {
		t.Fatalf("unexpected errored pairs: %+v", report.Summary)
	}
	if report.Summary.ScorerPending != 3 || report.Summary.ScorerMissing != 3 {
		t.Fatalf("unexpected scorer counters: %+v", report.Summary)
	}
	if report.Summary.A.TotalToolCalls != 1 || report.Summary.B.TotalToolCalls != 1 {
		t.Fatalf("unexpected side tool calls: %+v", report.Summary)
	}
	if report.Summary.A.AverageIterations != 2 || report.Summary.B.AverageIterations != float64(7)/3 {
		t.Fatalf("unexpected side averages: %+v", report.Summary)
	}
}

func TestReportsAreJSONSerializable(t *testing.T) {
	runReport := BuildRunReport([]CaseResult{
		{CaseID: "case-1", Passed: false, StopReason: "timed_out", Iterations: 1, Check: CheckResult{Checked: true, Passed: false}, Error: "timed out", ErrorClass: "timeout", FailedIteration: 1},
	})
	pairReport := BuildPairReport([]PairResult{
		{CaseID: "case-1", A: SingleRunResult{CaseResult: CaseResult{Passed: true}}, B: SingleRunResult{CaseResult: CaseResult{Passed: false}}, Score: ScoreResult{Scored: false, Reason: "not_scored"}},
	})

	if _, err := json.Marshal(runReport); err != nil {
		t.Fatalf("unexpected run report marshal error: %v", err)
	}
	if _, err := json.Marshal(pairReport); err != nil {
		t.Fatalf("unexpected pair report marshal error: %v", err)
	}
	data, err := json.Marshal(runReport)
	if err != nil {
		t.Fatalf("unexpected run report marshal error: %v", err)
	}
	for _, fragment := range []string{`"error_class":"timeout"`, `"failed_iteration":1`, `"timed_out_count":1`, `"error_classes":{"timeout":1}`} {
		if !strings.Contains(string(data), fragment) {
			t.Fatalf("expected fragment %s in %s", fragment, string(data))
		}
	}
}
