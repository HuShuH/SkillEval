// Package eval contains Phase 1 migration skeleton types for the new architecture.
// It currently provides minimal runnable reports and in-memory aggregation only.
package eval

import (
	"fmt"
	"sort"
	"strings"
)

// FormatRunSummary renders a CLI-friendly run report summary.
func FormatRunSummary(report RunReport) string {
	var builder strings.Builder

	builder.WriteString("run summary\n")
	builder.WriteString(fmt.Sprintf("total cases: %d\n", report.Summary.TotalCases))
	builder.WriteString(fmt.Sprintf("passed: %d\n", report.Summary.Passed))
	builder.WriteString(fmt.Sprintf("failed: %d\n", report.Summary.Failed))
	builder.WriteString(fmt.Sprintf("unchecked: %d\n", report.Summary.Unchecked))
	builder.WriteString(fmt.Sprintf("errored: %d\n", report.Summary.Errored))
	builder.WriteString(fmt.Sprintf("average iterations: %.2f\n", report.Summary.AverageIterations))
	builder.WriteString(fmt.Sprintf("total tool calls: %d\n", report.Summary.TotalToolCalls))
	if report.Summary.TimedOutCount > 0 {
		builder.WriteString(fmt.Sprintf("timed out: %d\n", report.Summary.TimedOutCount))
	}
	if report.Summary.CanceledCount > 0 {
		builder.WriteString(fmt.Sprintf("canceled: %d\n", report.Summary.CanceledCount))
	}
	builder.WriteString("stop reasons:\n")

	keys := sortedKeys(report.Summary.StopReasons)
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("  %s: %d\n", key, report.Summary.StopReasons[key]))
	}
	if len(report.Summary.ErrorClasses) > 0 {
		builder.WriteString("error classes:\n")
		for _, key := range sortedKeys(report.Summary.ErrorClasses) {
			builder.WriteString(fmt.Sprintf("  %s: %d\n", key, report.Summary.ErrorClasses[key]))
		}
	}

	return strings.TrimRight(builder.String(), "\n")
}

// FormatPairSummary renders a CLI-friendly pair report summary.
func FormatPairSummary(report PairReport) string {
	var builder strings.Builder

	builder.WriteString("pair summary\n")
	builder.WriteString(fmt.Sprintf("total pairs: %d\n", report.Summary.TotalPairs))
	builder.WriteString(fmt.Sprintf("both passed: %d\n", report.Summary.BothPassed))
	builder.WriteString(fmt.Sprintf("only A passed: %d\n", report.Summary.OnlyAPassed))
	builder.WriteString(fmt.Sprintf("only B passed: %d\n", report.Summary.OnlyBPassed))
	builder.WriteString(fmt.Sprintf("both failed: %d\n", report.Summary.BothFailed))
	builder.WriteString(fmt.Sprintf("errored pairs: %d\n", report.Summary.ErroredPairs))
	builder.WriteString(fmt.Sprintf("scorer pending: %d\n", report.Summary.ScorerPending))
	builder.WriteString("side A:\n")
	builder.WriteString(fmt.Sprintf("  passed: %d\n", report.Summary.A.Passed))
	builder.WriteString(fmt.Sprintf("  failed: %d\n", report.Summary.A.Failed))
	builder.WriteString(fmt.Sprintf("  errored: %d\n", report.Summary.A.Errored))
	builder.WriteString(fmt.Sprintf("  average iterations: %.2f\n", report.Summary.A.AverageIterations))
	builder.WriteString(fmt.Sprintf("  total tool calls: %d\n", report.Summary.A.TotalToolCalls))
	builder.WriteString("side B:\n")
	builder.WriteString(fmt.Sprintf("  passed: %d\n", report.Summary.B.Passed))
	builder.WriteString(fmt.Sprintf("  failed: %d\n", report.Summary.B.Failed))
	builder.WriteString(fmt.Sprintf("  errored: %d\n", report.Summary.B.Errored))
	builder.WriteString(fmt.Sprintf("  average iterations: %.2f\n", report.Summary.B.AverageIterations))
	builder.WriteString(fmt.Sprintf("  total tool calls: %d\n", report.Summary.B.TotalToolCalls))

	return strings.TrimRight(builder.String(), "\n")
}

func sortedKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
