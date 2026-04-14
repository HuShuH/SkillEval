package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"agent-skill-eval-go/internal/spec"
)

// Summarize aggregates run results into a report summary.
func Summarize(results []spec.RunResult) spec.ReportSummary {
	summary := spec.ReportSummary{
		Total:   len(results),
		Results: results,
	}

	for _, result := range results {
		if result.Passed {
			summary.Passed++
		} else {
			summary.Failed++
		}
	}

	return summary
}

// WriteJSON writes a formatted JSON report to the target path.
func WriteJSON(path string, summary spec.ReportSummary) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create report directory %s: %w", dir, err)
		}
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report summary: %w", err)
	}

	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write report file %s: %w", path, err)
	}

	return nil
}
