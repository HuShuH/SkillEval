package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"agent-skill-eval-go/internal/adapters"
	"agent-skill-eval-go/internal/checker"
	"agent-skill-eval-go/internal/registry"
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

// RunCases executes testcases sequentially using the registry, adapter, and checker.
func RunCases(
	ctx context.Context,
	reg *registry.Registry,
	adapter adapters.Adapter,
	testCases []spec.TestCase,
) []spec.RunResult {
	results := make([]spec.RunResult, 0, len(testCases))

	for _, tc := range testCases {
		startedAt := time.Now()
		result := spec.RunResult{
			CaseID: tc.CaseID,
			Skill:  tc.Skill,
		}

		skill, ok := reg.Get(tc.Skill.Name)
		if !ok {
			result.Passed = false
			result.Error = fmt.Sprintf("skill not found: %s", tc.Skill.Name)
			result.Reasons = []string{result.Error}
			result.DurationMS = time.Since(startedAt).Milliseconds()
			results = append(results, result)
			continue
		}

		caseCtx := ctx
		cancel := func() {}
		if tc.TimeoutSeconds > 0 {
			caseCtx, cancel = context.WithTimeout(ctx, time.Duration(tc.TimeoutSeconds)*time.Second)
		}

		output, err := adapter.Run(caseCtx, tc, skill)
		cancel()

		result.AgentOutput = output
		result.DurationMS = time.Since(startedAt).Milliseconds()

		if err != nil {
			result.Passed = false
			result.Error = err.Error()
			result.Reasons = []string{fmt.Sprintf("adapter run failed: %v", err)}
			results = append(results, result)
			continue
		}

		passed, reasons := checker.Check(tc, output)
		result.Passed = passed
		result.Reasons = reasons
		results = append(results, result)
	}

	return results
}
