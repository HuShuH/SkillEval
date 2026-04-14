package runner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"agent-skill-eval-go/internal/spec"
)

// LoadTestCases loads testcase definitions from a JSONL file.
func LoadTestCases(path string) ([]spec.TestCase, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("testcase file does not exist: %s", path)
		}
		return nil, fmt.Errorf("open testcase file %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	testCases := make([]spec.TestCase, 0)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var testCase spec.TestCase
		if err := json.Unmarshal([]byte(line), &testCase); err != nil {
			return nil, fmt.Errorf("parse testcase file %s at line %d: %w", path, lineNumber, err)
		}

		testCases = append(testCases, testCase)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan testcase file %s: %w", path, err)
	}

	return testCases, nil
}
