// Package eval contains Phase 1 migration skeleton types for the new architecture.
// It currently provides minimal safe run management operations over the filesystem output root.
package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ListFilter applies minimal filtering to run listing.
type ListFilter struct {
	Status string
	Mode   string
}

// ManageResult describes one management operation outcome.
type ManageResult struct {
	Action   string            `json:"action"`
	DryRun   bool              `json:"dry_run"`
	Affected []string          `json:"affected,omitempty"`
	Skipped  []string          `json:"skipped,omitempty"`
	Errors   map[string]string `json:"errors,omitempty"`
}

// RebuildIndex rebuilds and persists the output-root run index.
func RebuildIndex(outputRoot string) error {
	_, err := RebuildAndWriteRunIndex(outputRoot)
	return err
}

// ListRuns returns active runs from the index, filtered and sorted by created_at descending.
func ListRuns(outputRoot string, filter ListFilter) ([]RunIndexEntry, error) {
	index, err := LoadOrBuildRunIndex(outputRoot)
	if err != nil {
		return nil, err
	}
	var runs []RunIndexEntry
	for _, run := range index.Runs {
		if filter.Mode != "" && run.Mode != filter.Mode {
			continue
		}
		if filter.Status != "" && !matchManageStatus(run, filter.Status) {
			continue
		}
		runs = append(runs, run)
	}
	return runs, nil
}

// ArchiveRuns moves explicit runs into <output-root>/_archive/<run-id>.
func ArchiveRuns(outputRoot string, runIDs []string, dryRun bool) (ManageResult, error) {
	result := ManageResult{Action: "archive", DryRun: dryRun, Errors: map[string]string{}}
	archiveRoot := filepath.Join(outputRoot, ArchiveDirName)
	for _, runID := range runIDs {
		runID = strings.TrimSpace(runID)
		if err := validateRunID(runID); err != nil {
			result.Errors[runID] = err.Error()
			continue
		}
		source := filepath.Join(outputRoot, runID)
		target := filepath.Join(archiveRoot, runID)
		if _, err := os.Stat(source); err != nil {
			result.Errors[runID] = fmt.Sprintf("stat run: %v", err)
			continue
		}
		if dryRun {
			result.Affected = append(result.Affected, runID)
			continue
		}
		if err := os.MkdirAll(archiveRoot, 0o755); err != nil {
			return result, fmt.Errorf("create archive root %q: %w", archiveRoot, err)
		}
		if err := os.Rename(source, target); err != nil {
			result.Errors[runID] = fmt.Sprintf("archive run: %v", err)
			continue
		}
		result.Affected = append(result.Affected, runID)
	}
	if !dryRun {
		if err := RebuildIndex(outputRoot); err != nil {
			return result, err
		}
	}
	if len(result.Errors) == 0 {
		result.Errors = nil
	}
	return result, nil
}

// DeleteRuns deletes explicit runs from the active output root.
func DeleteRuns(outputRoot string, runIDs []string, dryRun bool) (ManageResult, error) {
	result := ManageResult{Action: "delete", DryRun: dryRun, Errors: map[string]string{}}
	for _, runID := range runIDs {
		runID = strings.TrimSpace(runID)
		if err := validateRunID(runID); err != nil {
			result.Errors[runID] = err.Error()
			continue
		}
		target := filepath.Join(outputRoot, runID)
		if _, err := os.Stat(target); err != nil {
			result.Errors[runID] = fmt.Sprintf("stat run: %v", err)
			continue
		}
		if dryRun {
			result.Affected = append(result.Affected, runID)
			continue
		}
		if err := os.RemoveAll(target); err != nil {
			result.Errors[runID] = fmt.Sprintf("delete run: %v", err)
			continue
		}
		result.Affected = append(result.Affected, runID)
	}
	if !dryRun {
		if err := RebuildIndex(outputRoot); err != nil {
			return result, err
		}
	}
	if len(result.Errors) == 0 {
		result.Errors = nil
	}
	return result, nil
}

// PruneRuns deletes older runs after keeping the newest N within the optional status subset.
func PruneRuns(outputRoot string, keep int, status string, dryRun bool) (ManageResult, error) {
	result := ManageResult{Action: "prune", DryRun: dryRun, Errors: map[string]string{}}
	if keep < 0 {
		return result, fmt.Errorf("keep must be >= 0")
	}
	runs, err := ListRuns(outputRoot, ListFilter{Status: status})
	if err != nil {
		return result, err
	}
	if keep >= len(runs) {
		result.Skipped = append(result.Skipped, "nothing to prune")
		result.Errors = nil
		return result, nil
	}
	candidates := make([]string, 0, len(runs)-keep)
	for _, run := range runs[keep:] {
		candidates = append(candidates, run.RunID)
	}
	deleteResult, err := DeleteRuns(outputRoot, candidates, dryRun)
	if err != nil {
		return deleteResult, err
	}
	deleteResult.Action = "prune"
	return deleteResult, nil
}

func validateRunID(runID string) error {
	if strings.TrimSpace(runID) == "" {
		return fmt.Errorf("run id is required")
	}
	if runID == ArchiveDirName || strings.Contains(runID, "/") || strings.Contains(runID, "\\") || strings.Contains(runID, "..") {
		return fmt.Errorf("invalid run id %q", runID)
	}
	return nil
}

func matchManageStatus(run RunIndexEntry, status string) bool {
	switch status {
	case "", "all":
		return true
	case "failed":
		return run.Failed > 0
	case "errored":
		return run.Errored > 0
	case "timed_out":
		return run.TimedOutCount > 0
	case "passed":
		return run.TotalCases > 0 && run.Passed == run.TotalCases && run.Failed == 0 && run.Errored == 0 && run.TimedOutCount == 0 && run.CanceledCount == 0
	default:
		return false
	}
}

// sortByRecency is a helper for tests and pruning determinism.
func sortByRecency(runs []RunIndexEntry) {
	sort.SliceStable(runs, func(i, j int) bool {
		if runs[i].CreatedAt == runs[j].CreatedAt {
			return runs[i].RunID > runs[j].RunID
		}
		return runs[i].CreatedAt > runs[j].CreatedAt
	})
}
