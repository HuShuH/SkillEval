// Package server contains the Phase 1 read-only HTTP API for persisted eval outputs.
// It exposes report and event browsing only; it does not start runs or stream events.
package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"agent-skill-eval-go/agent"
	"agent-skill-eval-go/eval"
)

func (s *Server) handleWeb(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var filePath string
	switch r.URL.Path {
	case "/", "/index.html":
		filePath = "../web/index.html"
	case "/app.js":
		filePath = "../web/app.js"
	case "/styles.css":
		filePath = "../web/styles.css"
	default:
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	data, err := os.ReadFile(webAssetPath(filePath))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	switch {
	case strings.HasSuffix(filePath, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(filePath, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case strings.HasSuffix(filePath, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func webAssetPath(path string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return path
	}
	return filepath.Join(filepath.Dir(filename), path)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if r.URL.Path != "/api/runs" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	index, err := eval.LoadOrBuildRunIndex(s.OutputRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	runs := filterRuns(index.Runs, r.URL.Query().Get("mode"), r.URL.Query().Get("status"), r.URL.Query().Get("limit"))
	writeJSON(w, http.StatusOK, RunsResponse{Runs: runs, Skipped: index.Skipped})
}

func (s *Server) handleRunSubtree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/runs/")
	parts := splitPath(path)
	if len(parts) == 1 {
		s.handleRunReport(w, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "summary" {
		s.handleRunSummary(w, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "stream" {
		s.handleRunStream(w, r, parts[0])
		return
	}
	if len(parts) == 4 && parts[1] == "cases" && parts[3] == "events" {
		s.handleCaseEvents(w, parts[0], parts[2], r.URL.Query().Get("side"))
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}

func (s *Server) handleRunStream(w http.ResponseWriter, r *http.Request, runID string) {
	events, cancel, err := s.Hub.Subscribe(runID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cancel()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, _ := w.(http.Flusher)
	if flusher != nil {
		flusher.Flush()
	}

	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			data, err := encodeSSEData(event)
			if err != nil {
				return
			}
			if _, err := fmt.Fprintf(w, "event: message\ndata: %s\n\n", data); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleRunReport(w http.ResponseWriter, runID string) {
	data, err := s.readReport(runID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("parse report.json for run %q: %v", runID, err))
		return
	}
	writeJSON(w, http.StatusOK, raw)
}

func (s *Server) handleRunSummary(w http.ResponseWriter, runID string) {
	data, err := s.readReport(runID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	var probe struct {
		ReportID   string         `json:"report_id,omitempty"`
		CreatedAt  string         `json:"created_at"`
		TotalCases int            `json:"total_cases"`
		Summary    map[string]any `json:"summary"`
		Metadata   map[string]any `json:"metadata,omitempty"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("parse report.json for run %q: %v", runID, err))
		return
	}

	mode := "single"
	if _, ok := probe.Summary["total_pairs"]; ok {
		mode = "pair"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"run_id":      runID,
		"report_id":   probe.ReportID,
		"created_at":  probe.CreatedAt,
		"total_cases": probe.TotalCases,
		"mode":        mode,
		"summary":     probe.Summary,
		"metadata":    probe.Metadata,
	})
}

func (s *Server) handleCaseEvents(w http.ResponseWriter, runID string, caseID string, side string) {
	events, err := s.readEvents(runID, caseID, side)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, EventsResponse{Events: events})
}

func (s *Server) readReport(runID string) ([]byte, error) {
	path := filepath.Join(s.OutputRoot, safeSegment(runID), eval.ReportFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read report for run %q: %w", runID, err)
	}
	return data, nil
}

func (s *Server) readEvents(runID string, caseID string, side string) ([]agent.Event, error) {
	base := filepath.Join(s.OutputRoot, safeSegment(runID), "cases", safeSegment(caseID))
	var path string
	switch side {
	case "":
		path = filepath.Join(base, eval.EventsFileName)
	case "a", "b":
		path = filepath.Join(base, side, eval.EventsFileName)
	default:
		return nil, fmt.Errorf("invalid side %q", side)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read events for run %q case %q: %w", runID, caseID, err)
	}
	defer file.Close()

	var events []agent.Event
	scanner := bufio.NewScanner(file)
	line := 0
	for scanner.Scan() {
		line++
		var event agent.Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("parse events for run %q case %q line %d: %w", runID, caseID, line, err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan events for run %q case %q: %w", runID, caseID, err)
	}
	normalizeEventTimestamps(events)
	sortEvents(events)
	return events, nil
}

func sortEvents(events []agent.Event) {
	sort.SliceStable(events, func(i, j int) bool {
		left := events[i].Timestamp
		right := events[j].Timestamp
		switch {
		case left.Equal(right):
			return events[i].Iteration < events[j].Iteration
		case left.IsZero():
			return false
		case right.IsZero():
			return true
		default:
			return left.Before(right)
		}
	})
}

func normalizeEventTimestamps(events []agent.Event) {
	for index := range events {
		if events[index].Timestamp.IsZero() {
			events[index].Timestamp = time.Unix(0, 0).UTC()
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

func splitPath(path string) []string {
	raw := strings.Split(strings.Trim(path, "/"), "/")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func safeSegment(value string) string {
	return strings.NewReplacer("/", "_", "\\", "_", "..", "_").Replace(strings.TrimSpace(value))
}

func filterRuns(runs []eval.RunIndexEntry, mode string, status string, limitRaw string) []eval.RunIndexEntry {
	filtered := make([]eval.RunIndexEntry, 0, len(runs))
	mode = strings.TrimSpace(mode)
	status = strings.TrimSpace(status)
	for _, run := range runs {
		if mode != "" && run.Mode != mode {
			continue
		}
		if status != "" && !matchesRunStatus(run, status) {
			continue
		}
		filtered = append(filtered, run)
	}
	if strings.TrimSpace(limitRaw) == "" {
		return filtered
	}
	limit, err := strconv.Atoi(limitRaw)
	if err != nil || limit <= 0 || limit >= len(filtered) {
		return filtered
	}
	return filtered[:limit]
}

func matchesRunStatus(run eval.RunIndexEntry, status string) bool {
	switch status {
	case "failed":
		return run.Failed > 0
	case "errored":
		return run.Errored > 0
	case "timed_out":
		return run.TimedOutCount > 0
	case "passed":
		return run.TotalCases > 0 && run.Passed == run.TotalCases && run.Failed == 0 && run.Errored == 0 && run.TimedOutCount == 0 && run.CanceledCount == 0
	default:
		return true
	}
}
