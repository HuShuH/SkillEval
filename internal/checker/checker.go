package checker

import (
	"fmt"
	"reflect"

	"agent-skill-eval-go/internal/spec"
)

// Check applies minimal hard-check rules to one agent output.
func Check(tc spec.TestCase, out spec.AgentOutput) (bool, []string) {
	reasons := make([]string, 0)
	checks := tc.HardChecks
	hasChecks := false
	passed := true

	if checks.ExpectedOutput != "" {
		hasChecks = true
		if out.FinalOutput == checks.ExpectedOutput {
			reasons = append(reasons, fmt.Sprintf("expected_output matched: %q", checks.ExpectedOutput))
		} else {
			passed = false
			reasons = append(reasons, fmt.Sprintf("expected_output mismatch: got %q want %q", out.FinalOutput, checks.ExpectedOutput))
		}
	}

	var matchedTool *spec.ToolCall
	if checks.ExpectedToolName != "" {
		hasChecks = true
		for i := range out.ToolCalls {
			if out.ToolCalls[i].ToolName == checks.ExpectedToolName {
				matchedTool = &out.ToolCalls[i]
				break
			}
		}

		if matchedTool != nil {
			reasons = append(reasons, fmt.Sprintf("expected_tool_name matched: %q", checks.ExpectedToolName))
		} else {
			passed = false
			reasons = append(reasons, fmt.Sprintf("expected_tool_name not found: %q", checks.ExpectedToolName))
		}
	}

	if checks.ExpectedArgs != nil {
		hasChecks = true
		if checks.ExpectedToolName == "" {
			passed = false
			reasons = append(reasons, "expected_args provided but expected_tool_name is empty")
		} else if matchedTool == nil {
			passed = false
			reasons = append(reasons, fmt.Sprintf("expected_args not checked because tool %q was not found", checks.ExpectedToolName))
		} else if reflect.DeepEqual(matchedTool.Args, checks.ExpectedArgs) {
			reasons = append(reasons, fmt.Sprintf("expected_args matched for tool %q", checks.ExpectedToolName))
		} else {
			passed = false
			reasons = append(reasons, fmt.Sprintf("expected_args mismatch for tool %q: got %#v want %#v", checks.ExpectedToolName, matchedTool.Args, checks.ExpectedArgs))
		}
	}

	if !hasChecks {
		return true, []string{"no hard checks configured; treated as pass"}
	}

	return passed, reasons
}
