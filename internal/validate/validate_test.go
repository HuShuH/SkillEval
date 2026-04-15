package validate

import (
	"path/filepath"
	"testing"

	"agent-skill-eval-go/internal/registry"
	"agent-skill-eval-go/internal/spec"
)

func TestSkillsEmptyName(t *testing.T) {
	errors := Skills([]spec.SkillSpec{{Name: ""}})
	if len(errors) == 0 {
		t.Fatal("expected skill name validation error")
	}
}

func TestTestCasesEmptyCaseID(t *testing.T) {
	reg := mustLoadRegistry(t)
	errors := TestCases([]spec.TestCase{{Skill: spec.SkillRef{Name: "hello_world"}}}, reg)
	if len(errors) == 0 {
		t.Fatal("expected case_id validation error")
	}
}

func TestTestCasesDuplicateCaseID(t *testing.T) {
	reg := mustLoadRegistry(t)
	errors := TestCases([]spec.TestCase{
		{CaseID: "dup", Skill: spec.SkillRef{Name: "hello_world"}},
		{CaseID: "dup", Skill: spec.SkillRef{Name: "hello_world"}},
	}, reg)
	if len(errors) == 0 {
		t.Fatal("expected duplicate case_id validation error")
	}
}

func TestTestCasesMissingSkillName(t *testing.T) {
	reg := mustLoadRegistry(t)
	errors := TestCases([]spec.TestCase{{CaseID: "case1"}}, reg)
	if len(errors) == 0 {
		t.Fatal("expected missing skill.name validation error")
	}
}

func TestTestCasesSkillNotFound(t *testing.T) {
	reg := mustLoadRegistry(t)
	errors := TestCases([]spec.TestCase{{CaseID: "case1", Skill: spec.SkillRef{Name: "missing_skill"}}}, reg)
	if len(errors) == 0 {
		t.Fatal("expected referenced skill validation error")
	}
}

func TestTestCasesExpectedArgsWithoutToolName(t *testing.T) {
	reg := mustLoadRegistry(t)
	errors := TestCases([]spec.TestCase{{
		CaseID: "case1",
		Skill:  spec.SkillRef{Name: "hello_world"},
		HardChecks: spec.HardChecks{
			ExpectedArgs: map[string]interface{}{"value": "ok"},
		},
	}}, reg)
	if len(errors) == 0 {
		t.Fatal("expected expected_tool_name validation error")
	}
}

func TestAllValidReturnsNoErrors(t *testing.T) {
	reg := mustLoadRegistry(t)
	skills := reg.List()
	testCases := []spec.TestCase{{
		CaseID: "case1",
		Skill:  spec.SkillRef{Name: "hello_world"},
	}}

	errors := All(skills, testCases, reg)
	if len(errors) != 0 {
		t.Fatalf("expected no validation errors, got: %v", errors)
	}
}

func mustLoadRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	reg, err := registry.LoadSkills(filepath.Join("..", "..", "testdata", "skills"))
	if err != nil {
		t.Fatalf("LoadSkills returned error: %v", err)
	}
	return reg
}
