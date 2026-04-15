package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-skill-eval-go/internal/registry"
	"agent-skill-eval-go/internal/spec"
)

func TestSkillFilesInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	writeSkillFile(t, dir, "bad.json", `{invalid json}`)

	_, errors := SkillFiles(dir)
	if len(errors) == 0 {
		t.Fatal("expected invalid JSON skill file error")
	}
	if !containsAny(errors, "parse skill file") {
		t.Fatalf("expected parse skill file error, got: %v", errors)
	}
}

func TestSkillFilesEmptyName(t *testing.T) {
	dir := t.TempDir()
	writeSkillFile(t, dir, "empty.json", `{"name":"","description":"bad"}`)

	_, errors := SkillFiles(dir)
	if len(errors) == 0 {
		t.Fatal("expected empty skill name error")
	}
	if !containsAny(errors, "skill name must not be empty") {
		t.Fatalf("expected empty name error, got: %v", errors)
	}
}

func TestSkillFilesDuplicateName(t *testing.T) {
	dir := t.TempDir()
	writeSkillFile(t, dir, "one.json", `{"name":"dup","description":"one"}`)
	writeSkillFile(t, dir, "two.json", `{"name":"dup","description":"two"}`)

	_, errors := SkillFiles(dir)
	if len(errors) == 0 {
		t.Fatal("expected duplicate skill name error")
	}
	if !containsAny(errors, "duplicate skill name") {
		t.Fatalf("expected duplicate skill name error, got: %v", errors)
	}
}

func TestSkillFilesValid(t *testing.T) {
	dir := t.TempDir()
	writeSkillFile(t, dir, "hello.json", `{"name":"hello_world","description":"ok"}`)
	writeSkillFile(t, dir, "echo.json", `{"name":"echo","description":"ok"}`)

	skills, errors := SkillFiles(dir)
	if len(errors) != 0 {
		t.Fatalf("expected no skill file errors, got: %v", errors)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}
}

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

func writeSkillFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write skill file failed: %v", err)
	}
}

func containsAny(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
