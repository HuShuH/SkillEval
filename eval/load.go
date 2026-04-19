// Package eval contains Phase 1 migration skeleton types for the new architecture.
// It currently provides minimal runnable reports and in-memory aggregation only.
package eval

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// LoadCases loads cases from either a JSON array file or a JSONL file.
func LoadCases(path string) ([]Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cases file %q: %w", path, err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("cases file %q is empty", path)
	}

	trimmed := bytes.TrimSpace(data)
	if trimmed[0] == '[' {
		return loadJSONArrayCases(trimmed)
	}
	return loadJSONLCases(trimmed)
}

func loadJSONArrayCases(data []byte) ([]Case, error) {
	var cases []Case
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, fmt.Errorf("parse cases json array: %w", err)
	}
	for index, c := range cases {
		if err := c.Validate(); err != nil {
			return nil, fmt.Errorf("invalid case at index %d: %w", index, err)
		}
	}
	return cases, nil
}

func loadJSONLCases(data []byte) ([]Case, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var cases []Case
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var c Case
		if err := json.Unmarshal(line, &c); err != nil {
			return nil, fmt.Errorf("parse cases jsonl line %d: %w", lineNumber, err)
		}
		if err := c.Validate(); err != nil {
			return nil, fmt.Errorf("invalid case at line %d: %w", lineNumber, err)
		}
		cases = append(cases, c)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan cases jsonl: %w", err)
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("cases file contains no cases")
	}
	return cases, nil
}
