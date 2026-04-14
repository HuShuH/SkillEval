package checker

import (
	"testing"

	"agent-skill-eval-go/internal/spec"
)

func TestCheckExpectedOutputMatched(t *testing.T) {
	tc := spec.TestCase{
		HardChecks: spec.HardChecks{ExpectedOutput: "hello world"},
	}
	out := spec.AgentOutput{FinalOutput: "hello world"}

	passed, reasons := Check(tc, out)
	if !passed {
		t.Fatalf("expected pass, got fail: %v", reasons)
	}
	if len(reasons) == 0 {
		t.Fatal("expected reasons to be populated")
	}
}

func TestCheckExpectedOutputMismatch(t *testing.T) {
	tc := spec.TestCase{
		HardChecks: spec.HardChecks{ExpectedOutput: "hello world"},
	}
	out := spec.AgentOutput{FinalOutput: "not hello"}

	passed, reasons := Check(tc, out)
	if passed {
		t.Fatalf("expected fail, got pass: %v", reasons)
	}
	if len(reasons) == 0 {
		t.Fatal("expected mismatch reason")
	}
}

func TestCheckExpectedToolAndArgsMatched(t *testing.T) {
	tc := spec.TestCase{
		HardChecks: spec.HardChecks{
			ExpectedToolName: "mock_tool",
			ExpectedArgs: map[string]interface{}{
				"value": "ok",
			},
		},
	}
	out := spec.AgentOutput{
		ToolCalls: []spec.ToolCall{
			{
				ToolName: "mock_tool",
				Args: map[string]interface{}{
					"value": "ok",
				},
			},
		},
	}

	passed, reasons := Check(tc, out)
	if !passed {
		t.Fatalf("expected pass, got fail: %v", reasons)
	}
}
