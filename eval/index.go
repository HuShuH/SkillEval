// Package eval contains Phase 1 migration skeleton types for the new architecture.
// It currently provides minimal filesystem-backed run index building and caching.
package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const IndexFileName = "index.json"
const ArchiveDirName = "_archive"

// RunIndexEntry is one stable summary row for a persisted run.
type RunIndexEntry struct {
	RunID         string `json:"run_id"`
	CreatedAt     string `json:"created_at,omitempty"`
	Mode          string `json:"mode,omitempty"`
	TotalCases    int    `json:"total_cases"`
	Passed        int    `json:"passed"`
	Failed        int    `json:"failed"`
	Errored       int    `json:"errored"`
	TimedOutCount int    `json:"timed_out_count"`
	CanceledCount int    `json:"canceled_count"`
	Provider      string `json:"provider,omitempty"`
	Model         string `json:"model,omitempty"`
	SkillA        string `json:"skill_a,omitempty"`
	SkillB        string `json:"skill_b,omitempty"`
	OutputDir     string `json:"output_dir,omitempty"`
	Path          string `json:"path,omitempty"`
	HasHTMLReport bool   `json:"has_html_report"`
}

// RunIndexError records one skipped run during index building.
type RunIndexError struct {
	RunID string `json:"run_id"`
	Error string `json:"error"`
}

// RunIndex is the top-level persisted run index document.
type RunIndex struct {
	GeneratedAt string          `json:"generated_at"`
	Runs        []RunIndexEntry `json:"runs"`
	Skipped     []RunIndexError `json:"skipped,omitempty"`
}

// BuildRunIndex scans one output root and builds a stable run index.
func BuildRunIndex(outputRoot string) (RunIndex, error) {
	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		return RunIndex{}, fmt.Errorf("read output root %q: %w", outputRoot, err)
	}

	index := RunIndex{
		GeneratedAt: nowRFC3339(),
		Runs:        make([]RunIndexEntry, 0, len(entries)),
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		runID := entry.Name()
		if runID == ArchiveDirName {
			continue
		}
		reportPath := filepath.Join(outputRoot, runID, ReportFileName)
		data, err := os.ReadFile(reportPath)
		if err != nil {
			index.Skipped = append(index.Skipped, RunIndexError{
				RunID: runID,
				Error: fmt.Sprintf("read report: %v", err),
			})
			continue
		}
		current, err := parseRunIndexEntry(runID, filepath.Join(outputRoot, runID), data)
		if err != nil {
			index.Skipped = append(index.Skipped, RunIndexError{
				RunID: runID,
				Error: err.Error(),
			})
			continue
		}
		if _, err := os.Stat(filepath.Join(outputRoot, runID, ReportHTMLFileName)); err == nil {
			current.HasHTMLReport = true
		}
		index.Runs = append(index.Runs, current)
	}

	sort.SliceStable(index.Runs, func(i, j int) bool {
		if index.Runs[i].CreatedAt == index.Runs[j].CreatedAt {
			return index.Runs[i].RunID > index.Runs[j].RunID
		}
		return index.Runs[i].CreatedAt > index.Runs[j].CreatedAt
	})

	return index, nil
}

// LoadRunIndex reads one cached index.json file.
func LoadRunIndex(outputRoot string) (RunIndex, error) {
	data, err := os.ReadFile(filepath.Join(outputRoot, IndexFileName))
	if err != nil {
		return RunIndex{}, fmt.Errorf("read run index %q: %w", filepath.Join(outputRoot, IndexFileName), err)
	}
	var index RunIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return RunIndex{}, fmt.Errorf("parse run index %q: %w", filepath.Join(outputRoot, IndexFileName), err)
	}
	if index.Runs == nil {
		index.Runs = []RunIndexEntry{}
	}
	return index, nil
}

// LoadOrBuildRunIndex reads index.json when present, otherwise rebuilds it from reports.
func LoadOrBuildRunIndex(outputRoot string) (RunIndex, error) {
	index, err := LoadRunIndex(outputRoot)
	if err == nil {
		return index, nil
	}
	index, err = BuildRunIndex(outputRoot)
	if err != nil {
		return RunIndex{}, err
	}
	_, _ = WriteRunIndex(outputRoot, index)
	return index, nil
}

// WriteRunIndex writes one cached index.json file to the output root.
func WriteRunIndex(outputRoot string, index RunIndex) (string, error) {
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return "", fmt.Errorf("create output root %q: %w", outputRoot, err)
	}
	path := filepath.Join(outputRoot, IndexFileName)
	encoded, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode run index: %w", err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		return "", fmt.Errorf("write run index %q: %w", path, err)
	}
	return path, nil
}

// RebuildAndWriteRunIndex rebuilds the cached run index from report.json files.
func RebuildAndWriteRunIndex(outputRoot string) (RunIndex, error) {
	index, err := BuildRunIndex(outputRoot)
	if err != nil {
		return RunIndex{}, err
	}
	if _, err := WriteRunIndex(outputRoot, index); err != nil {
		return RunIndex{}, err
	}
	return index, nil
}

func parseRunIndexEntry(runID string, runDir string, data []byte) (RunIndexEntry, error) {
	var probe struct {
		CreatedAt  string            `json:"created_at"`
		TotalCases int               `json:"total_cases"`
		Summary    map[string]any    `json:"summary"`
		Metadata   map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return RunIndexEntry{}, fmt.Errorf("parse report.json for run %q: %w", runID, err)
	}

	mode := "single"
	if _, ok := probe.Summary["total_pairs"]; ok {
		mode = "pair"
	}

	entry := RunIndexEntry{
		RunID:         runID,
		CreatedAt:     probe.CreatedAt,
		Mode:          mode,
		TotalCases:    probe.TotalCases,
		OutputDir:     runDir,
		Path:          runDir,
		Provider:      probe.Metadata["provider_mode"],
		Model:         probe.Metadata["model"],
		SkillA:        probe.Metadata["skill_a"],
		SkillB:        probe.Metadata["skill_b"],
		Passed:        summaryInt(probe.Summary, "passed"),
		Failed:        summaryInt(probe.Summary, "failed"),
		Errored:       summaryInt(probe.Summary, "errored"),
		TimedOutCount: summaryInt(probe.Summary, "timed_out_count"),
		CanceledCount: summaryInt(probe.Summary, "canceled_count"),
	}

	return entry, nil
}

func summaryInt(values map[string]any, key string) int {
	if values == nil {
		return 0
	}
	switch value := values[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case int64:
		return int(value)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(value))
		return parsed
	default:
		return 0
	}
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
