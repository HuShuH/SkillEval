// Package eval contains Phase 1 migration skeleton types for the new architecture.
// It currently provides minimal runnable reports and in-memory aggregation only.
package eval

import "time"

// RunSummary contains aggregate statistics for single-run case results.
type RunSummary struct {
	TotalCases           int            `json:"total_cases"`
	Passed               int            `json:"passed"`
	Failed               int            `json:"failed"`
	Unchecked            int            `json:"unchecked"`
	Errored              int            `json:"errored"`
	FinishedCount        int            `json:"finished_count"`
	MaxIterationsCount   int            `json:"max_iterations_count"`
	TimedOutCount        int            `json:"timed_out_count"`
	CanceledCount        int            `json:"canceled_count"`
	ProviderErrorCount   int            `json:"provider_error_count"`
	ToolErrorCount       int            `json:"tool_error_count"`
	ToolNotFoundCount    int            `json:"tool_not_found_count"`
	InvalidResponseCount int            `json:"invalid_response_count"`
	AverageIterations    float64        `json:"average_iterations"`
	TotalToolCalls       int            `json:"total_tool_calls"`
	StopReasons          map[string]int `json:"stop_reasons,omitempty"`
	ErrorClasses         map[string]int `json:"error_classes,omitempty"`
}

// RunReport is the in-memory aggregate of multiple case runs.
type RunReport struct {
	ReportID   string            `json:"report_id,omitempty"`
	CreatedAt  string            `json:"created_at"`
	OutputDir  string            `json:"output_dir,omitempty"`
	TotalCases int               `json:"total_cases"`
	Results    []CaseResult      `json:"results"`
	Summary    RunSummary        `json:"summary"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Config     map[string]any    `json:"config,omitempty"`
}

// SideSummary contains side-specific pair statistics.
type SideSummary struct {
	Passed            int     `json:"passed"`
	Failed            int     `json:"failed"`
	Errored           int     `json:"errored"`
	AverageIterations float64 `json:"average_iterations"`
	TotalToolCalls    int     `json:"total_tool_calls"`
}

// PairSummary contains aggregate statistics for A/B pair results.
type PairSummary struct {
	TotalPairs    int         `json:"total_pairs"`
	BothPassed    int         `json:"both_passed"`
	OnlyAPassed   int         `json:"only_a_passed"`
	OnlyBPassed   int         `json:"only_b_passed"`
	BothFailed    int         `json:"both_failed"`
	ErroredPairs  int         `json:"errored_pairs"`
	ScorerPending int         `json:"scorer_pending"`
	ScorerMissing int         `json:"scorer_missing"`
	A             SideSummary `json:"a"`
	B             SideSummary `json:"b"`
}

// PairReport is the in-memory aggregate of multiple pair runs.
type PairReport struct {
	ReportID   string            `json:"report_id,omitempty"`
	CreatedAt  string            `json:"created_at"`
	OutputDir  string            `json:"output_dir,omitempty"`
	TotalCases int               `json:"total_cases"`
	Results    []PairResult      `json:"results"`
	Summary    PairSummary       `json:"summary"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Config     map[string]any    `json:"config,omitempty"`
}

// BuildRunReport aggregates multiple case results into a report.
func BuildRunReport(results []CaseResult) RunReport {
	report := RunReport{
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		TotalCases: len(results),
		Results:    append([]CaseResult(nil), results...),
		Summary: RunSummary{
			StopReasons:  make(map[string]int),
			ErrorClasses: make(map[string]int),
		},
	}

	var iterationSum int
	for _, result := range results {
		report.Summary.TotalCases++
		if result.Passed {
			report.Summary.Passed++
		} else {
			report.Summary.Failed++
		}
		if !result.Check.Checked {
			report.Summary.Unchecked++
		}
		if result.Error != "" {
			report.Summary.Errored++
		}

		report.Summary.TotalToolCalls += len(result.ToolExecutions)
		iterationSum += result.Iterations
		report.Summary.StopReasons[result.StopReason]++
		if result.ErrorClass != "" {
			report.Summary.ErrorClasses[result.ErrorClass]++
		}

		switch result.StopReason {
		case "finished":
			report.Summary.FinishedCount++
		case "max_iterations":
			report.Summary.MaxIterationsCount++
		case "timed_out":
			report.Summary.TimedOutCount++
		case "canceled":
			report.Summary.CanceledCount++
		case "provider_error":
			report.Summary.ProviderErrorCount++
		case "tool_error":
			report.Summary.ToolErrorCount++
		case "tool_not_found":
			report.Summary.ToolNotFoundCount++
		case "invalid_response":
			report.Summary.InvalidResponseCount++
		}
	}

	if len(results) > 0 {
		report.Summary.AverageIterations = float64(iterationSum) / float64(len(results))
	}
	return report
}

// BuildPairReport aggregates multiple pair results into a report.
func BuildPairReport(results []PairResult) PairReport {
	report := PairReport{
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		TotalCases: len(results),
		Results:    append([]PairResult(nil), results...),
	}

	var totalAIterations int
	var totalBIterations int

	for _, result := range results {
		report.Summary.TotalPairs++
		a := result.A.CaseResult
		b := result.B.CaseResult

		switch {
		case a.Passed && b.Passed:
			report.Summary.BothPassed++
		case a.Passed && !b.Passed:
			report.Summary.OnlyAPassed++
		case !a.Passed && b.Passed:
			report.Summary.OnlyBPassed++
		default:
			report.Summary.BothFailed++
		}

		if result.Error != "" || a.Error != "" || b.Error != "" {
			report.Summary.ErroredPairs++
		}
		if !result.Score.Scored {
			report.Summary.ScorerPending++
			if result.Score.Reason == "" || result.Score.Reason == "not_scored" {
				report.Summary.ScorerMissing++
			}
		}

		if a.Passed {
			report.Summary.A.Passed++
		} else {
			report.Summary.A.Failed++
		}
		if a.Error != "" {
			report.Summary.A.Errored++
		}
		report.Summary.A.TotalToolCalls += len(a.ToolExecutions)
		totalAIterations += a.Iterations

		if b.Passed {
			report.Summary.B.Passed++
		} else {
			report.Summary.B.Failed++
		}
		if b.Error != "" {
			report.Summary.B.Errored++
		}
		report.Summary.B.TotalToolCalls += len(b.ToolExecutions)
		totalBIterations += b.Iterations
	}

	if len(results) > 0 {
		report.Summary.A.AverageIterations = float64(totalAIterations) / float64(len(results))
		report.Summary.B.AverageIterations = float64(totalBIterations) / float64(len(results))
	}
	return report
}
