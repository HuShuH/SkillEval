// Package eval contains Phase 1 migration skeleton types for the new architecture.
// It currently provides minimal filesystem persistence for reports and events only.
package eval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-skill-eval-go/agent"
)

const (
	ReportFileName     = "report.json"
	ReportHTMLFileName = "report.html"
	EventsFileName     = "events.jsonl"
)

// OutputStore writes new-architecture reports and events to a stable directory layout.
type OutputStore struct {
	OutputRoot string
	RunID      string
}

// NewOutputStore creates an output store. If runID is empty, it is generated on first use.
func NewOutputStore(outputRoot string, runID string) *OutputStore {
	return &OutputStore{
		OutputRoot: outputRoot,
		RunID:      runID,
	}
}

// RunDir returns the concrete output directory for this run.
func (s *OutputStore) RunDir() (string, error) {
	if strings.TrimSpace(s.OutputRoot) == "" {
		return "", fmt.Errorf("output root is required")
	}
	runID := s.ensureRunID()
	return filepath.Join(s.OutputRoot, runID), nil
}

// WriteRunReport writes report.json and single-run case events.
func (s *OutputStore) WriteRunReport(report RunReport) (string, error) {
	runDir, err := s.RunDir()
	if err != nil {
		return "", err
	}
	report.ReportID = s.RunID
	report.OutputDir = runDir

	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return "", fmt.Errorf("create run output dir %q: %w", runDir, err)
	}
	reportPath := filepath.Join(runDir, ReportFileName)
	encoded, err := EncodeRunReport(report)
	if err != nil {
		return "", fmt.Errorf("encode run report: %w", err)
	}
	if err := os.WriteFile(reportPath, encoded, 0o644); err != nil {
		return "", fmt.Errorf("write run report %q: %w", reportPath, err)
	}

	for _, result := range report.Results {
		if _, err := s.WriteCaseEvents(result.CaseID, result.Events); err != nil {
			return "", err
		}
	}
	return reportPath, nil
}

// WritePairReport writes report.json and A/B case events.
func (s *OutputStore) WritePairReport(report PairReport) (string, error) {
	runDir, err := s.RunDir()
	if err != nil {
		return "", err
	}
	report.ReportID = s.RunID
	report.OutputDir = runDir

	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return "", fmt.Errorf("create pair output dir %q: %w", runDir, err)
	}
	reportPath := filepath.Join(runDir, ReportFileName)
	encoded, err := EncodePairReport(report)
	if err != nil {
		return "", fmt.Errorf("encode pair report: %w", err)
	}
	if err := os.WriteFile(reportPath, encoded, 0o644); err != nil {
		return "", fmt.Errorf("write pair report %q: %w", reportPath, err)
	}

	for _, result := range report.Results {
		if _, err := s.WritePairCaseEvents(result.CaseID, "a", result.A.CaseResult.Events); err != nil {
			return "", err
		}
		if _, err := s.WritePairCaseEvents(result.CaseID, "b", result.B.CaseResult.Events); err != nil {
			return "", err
		}
	}
	return reportPath, nil
}

// WriteRunReportHTML writes a static offline HTML report next to report.json.
func (s *OutputStore) WriteRunReportHTML(report RunReport) (string, error) {
	runDir, err := s.RunDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return "", fmt.Errorf("create run output dir %q: %w", runDir, err)
	}
	report.ReportID = s.RunID
	report.OutputDir = runDir
	rendered, err := RenderRunReportHTML(report)
	if err != nil {
		return "", fmt.Errorf("render run html report: %w", err)
	}
	reportPath := filepath.Join(runDir, ReportHTMLFileName)
	if err := os.WriteFile(reportPath, rendered, 0o644); err != nil {
		return "", fmt.Errorf("write run html report %q: %w", reportPath, err)
	}
	return reportPath, nil
}

// WritePairReportHTML writes a static offline pair HTML report next to report.json.
func (s *OutputStore) WritePairReportHTML(report PairReport) (string, error) {
	runDir, err := s.RunDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return "", fmt.Errorf("create pair output dir %q: %w", runDir, err)
	}
	report.ReportID = s.RunID
	report.OutputDir = runDir
	rendered, err := RenderPairReportHTML(report)
	if err != nil {
		return "", fmt.Errorf("render pair html report: %w", err)
	}
	reportPath := filepath.Join(runDir, ReportHTMLFileName)
	if err := os.WriteFile(reportPath, rendered, 0o644); err != nil {
		return "", fmt.Errorf("write pair html report %q: %w", reportPath, err)
	}
	return reportPath, nil
}

// WriteCaseEvents writes single-mode case events to cases/<case-id>/events.jsonl.
func (s *OutputStore) WriteCaseEvents(caseID string, events []agent.Event) (string, error) {
	runDir, err := s.RunDir()
	if err != nil {
		return "", err
	}
	eventPath := filepath.Join(runDir, "cases", sanitizePathPart(caseID), EventsFileName)
	if err := writeEventsJSONL(eventPath, events); err != nil {
		return "", err
	}
	return eventPath, nil
}

// WritePairCaseEvents writes pair-mode side events to cases/<case-id>/<side>/events.jsonl.
func (s *OutputStore) WritePairCaseEvents(caseID string, side string, events []agent.Event) (string, error) {
	runDir, err := s.RunDir()
	if err != nil {
		return "", err
	}
	safeSide := sanitizePathPart(side)
	if safeSide != "a" && safeSide != "b" {
		return "", fmt.Errorf("invalid pair side %q", side)
	}
	eventPath := filepath.Join(runDir, "cases", sanitizePathPart(caseID), safeSide, EventsFileName)
	if err := writeEventsJSONL(eventPath, events); err != nil {
		return "", err
	}
	return eventPath, nil
}

func (s *OutputStore) ensureRunID() string {
	if strings.TrimSpace(s.RunID) == "" {
		s.RunID = "run-" + time.Now().UTC().Format("20060102T150405.000000000Z")
	}
	s.RunID = sanitizePathPart(s.RunID)
	return s.RunID
}

func writeEventsJSONL(path string, events []agent.Event) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create events dir %q: %w", filepath.Dir(path), err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create events file %q: %w", path, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			return fmt.Errorf("write event to %q: %w", path, err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush events file %q: %w", path, err)
	}
	return nil
}

func sanitizePathPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}

	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-', r == '_', r == '.':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}

	cleaned := strings.Trim(builder.String(), ".")
	if cleaned == "" || cleaned == ".." {
		return "unknown"
	}
	return cleaned
}
