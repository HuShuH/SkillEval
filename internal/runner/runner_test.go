package runner

import (
	"context"
	"path/filepath"
	"testing"

	"agent-skill-eval-go/internal/adapters"
	"agent-skill-eval-go/internal/registry"
	"agent-skill-eval-go/internal/spec"
)

func TestLoadTestCasesSuccess(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "cases", "mvp.jsonl")

	testCases, err := LoadTestCases(path)
	if err != nil {
		t.Fatalf("LoadTestCases returned error: %v", err)
	}
	if len(testCases) != 3 {
		t.Fatalf("expected 3 testcases, got %d", len(testCases))
	}
}

func TestLoadTestCasesMissingFile(t *testing.T) {
	_, err := LoadTestCases(filepath.Join("..", "..", "testdata", "cases", "missing.jsonl"))
	if err == nil {
		t.Fatal("expected error for missing testcase file")
	}
}

func TestRunCasesSuccess(t *testing.T) {
	reg, err := registry.LoadSkills(filepath.Join("..", "..", "testdata", "skills"))
	if err != nil {
		t.Fatalf("LoadSkills returned error: %v", err)
	}

	testCases := []spec.TestCase{
		{
			CaseID: "case_hello_world",
			Skill:  spec.SkillRef{Name: "hello_world"},
			HardChecks: spec.HardChecks{
				ExpectedOutput: "hello world",
			},
			TimeoutSeconds: 1,
		},
	}

	results := RunCases(context.Background(), reg, adapters.MockAdapter{}, testCases)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Passed {
		t.Fatalf("expected run to pass, got fail: %+v", results[0])
	}
}

func TestRunCasesSkillNotFound(t *testing.T) {
	reg, err := registry.LoadSkills(filepath.Join("..", "..", "testdata", "skills"))
	if err != nil {
		t.Fatalf("LoadSkills returned error: %v", err)
	}

	testCases := []spec.TestCase{
		{
			CaseID: "case_missing_skill",
			Skill:  spec.SkillRef{Name: "does_not_exist"},
		},
	}

	results := RunCases(context.Background(), reg, adapters.MockAdapter{}, testCases)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Passed {
		t.Fatalf("expected missing skill result to fail: %+v", results[0])
	}
	if results[0].Error == "" {
		t.Fatal("expected missing skill error to be recorded")
	}
}
