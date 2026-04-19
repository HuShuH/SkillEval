package eval

import (
	"strings"
	"testing"

	"agent-skill-eval-go/agent"
)

func TestRenderRunReportHTML(t *testing.T) {
	report := BuildRunReport([]CaseResult{
		{
			CaseID:          "case-1",
			AgentName:       "agent-a",
			Passed:          false,
			StopReason:      "timed_out",
			FinalAnswer:     "partial answer",
			Iterations:      2,
			Error:           "request timed out",
			ErrorClass:      "timeout",
			FailedIteration: 2,
			Check:           CheckResult{Checked: true, Passed: false},
			Events: []agent.Event{
				{Type: "provider.request.failed", Iteration: 2, Message: "provider request failed"},
				{Type: "provider.request.retried", Iteration: 2, Message: "provider request retried"},
				{Type: "run.timed_out", Iteration: 2, Message: "run timed out"},
			},
		},
	})
	report.ReportID = "run-1"

	got, err := RenderRunReportHTML(report)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	html := string(got)
	for _, fragment := range []string{
		"<!doctype html>",
		"Run Report",
		"run-1",
		"Timed Out",
		"Error Classes",
		"case-1",
		"request timed out",
		"provider.request.failed",
		"provider.request.retried",
		"run.timed_out",
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("expected fragment %q in html:\n%s", fragment, html)
		}
	}
}

func TestRenderPairReportHTML(t *testing.T) {
	report := BuildPairReport([]PairResult{
		{
			CaseID: "case-1",
			A: SingleRunResult{CaseResult: CaseResult{
				CaseID:      "case-1",
				AgentName:   "agent-a",
				Passed:      true,
				StopReason:  "finished",
				FinalAnswer: "answer a",
				Iterations:  1,
				Events:      []agent.Event{{Type: "provider.request.retried", Iteration: 1}},
			}},
			B: SingleRunResult{CaseResult: CaseResult{
				CaseID:          "case-1",
				AgentName:       "agent-b",
				Passed:          false,
				StopReason:      "canceled",
				FinalAnswer:     "answer b",
				Iterations:      2,
				Error:           "request canceled",
				ErrorClass:      "canceled",
				FailedIteration: 2,
				Events:          []agent.Event{{Type: "run.canceled", Iteration: 2}},
			}},
			Score: ScoreResult{Scored: false, Reason: "not_scored"},
		},
	})
	report.ReportID = "pair-1"

	got, err := RenderPairReportHTML(report)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	html := string(got)
	for _, fragment := range []string{
		"<!doctype html>",
		"Pair Report",
		"pair-1",
		"Pair Summary",
		"Side A",
		"Side B",
		"answer a",
		"request canceled",
		"run.canceled",
		"not_scored",
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("expected fragment %q in html:\n%s", fragment, html)
		}
	}
}
