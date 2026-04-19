// Package eval contains Phase 1 migration skeleton types for the new architecture.
// It currently provides minimal runnable case definitions, validation, and runner types.
package eval

import (
	"fmt"
	"strings"
)

const (
	CheckerExactMatch = "exact_match"
	CheckerContains   = "contains"
	CheckerNonEmpty   = "non_empty"
)

// CheckResult is the normalized outcome of evaluating one case result.
type CheckResult struct {
	Type    string `json:"type"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
	Checked bool   `json:"checked"`
}

// EvaluateChecks applies the minimal checker rules against a final answer.
func EvaluateChecks(c Case, finalAnswer string) CheckResult {
	if len(c.Checkers) == 0 {
		if strings.TrimSpace(c.Expected.FinalText) != "" {
			return evaluateChecker(CheckerConfig{
				Type: CheckerExactMatch,
				Config: map[string]any{
					"value": c.Expected.FinalText,
				},
			}, finalAnswer)
		}
		return CheckResult{
			Type:    "unchecked",
			Passed:  true,
			Checked: false,
			Message: "no checker configured",
		}
	}

	return evaluateChecker(c.Checkers[0], finalAnswer)
}

func evaluateChecker(config CheckerConfig, finalAnswer string) CheckResult {
	switch config.Type {
	case "", CheckerExactMatch:
		expected, _ := config.Config["value"].(string)
		passed := finalAnswer == expected
		return CheckResult{
			Type:    CheckerExactMatch,
			Passed:  passed,
			Checked: true,
			Message: checkerMessage(passed, "exact match", expected, finalAnswer),
		}
	case CheckerContains:
		expected, _ := config.Config["value"].(string)
		passed := strings.Contains(finalAnswer, expected)
		return CheckResult{
			Type:    CheckerContains,
			Passed:  passed,
			Checked: true,
			Message: checkerMessage(passed, "contains", expected, finalAnswer),
		}
	case CheckerNonEmpty:
		passed := strings.TrimSpace(finalAnswer) != ""
		return CheckResult{
			Type:    CheckerNonEmpty,
			Passed:  passed,
			Checked: true,
			Message: checkerMessage(passed, "non-empty", "", finalAnswer),
		}
	default:
		return CheckResult{
			Type:    config.Type,
			Passed:  false,
			Checked: true,
			Message: fmt.Sprintf("unsupported checker type %q", config.Type),
		}
	}
}

func checkerMessage(passed bool, mode string, expected string, actual string) string {
	if passed {
		return fmt.Sprintf("%s checker passed", mode)
	}
	if expected == "" {
		return fmt.Sprintf("%s checker failed", mode)
	}
	return fmt.Sprintf("%s checker failed: expected %q got %q", mode, expected, actual)
}
