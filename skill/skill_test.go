package skill

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-skill-eval-go/tool"
)

func TestSkillConstructible(t *testing.T) {
	s, err := NewSkill("writer", " v2 ", "drafts content", []string{"filesystem", "finish"}, "system", "follow the repo skill")
	if err != nil {
		t.Fatalf("unexpected new skill error: %v", err)
	}

	if s.Name != "writer" {
		t.Fatalf("unexpected skill name: %q", s.Name)
	}
	if s.Version != "v2" {
		t.Fatalf("unexpected normalized version: %q", s.Version)
	}
	if len(s.Tools) != 2 {
		t.Fatalf("unexpected tools length: %d", len(s.Tools))
	}
}

func TestSkillValidateSuccess(t *testing.T) {
	s := Skill{
		Name:         "writer",
		Version:      " v1 ",
		Tools:        []string{"filesystem", "finish"},
		Instructions: "follow the task",
	}

	if err := s.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestSkillValidateEmptyNameFails(t *testing.T) {
	s := Skill{
		Instructions: "follow the task",
	}

	err := s.Validate()
	if !errors.Is(err, ErrSkillNameRequired) {
		t.Fatalf("expected ErrSkillNameRequired, got %v", err)
	}
}

func TestSkillValidateMissingInstructionsFails(t *testing.T) {
	s := Skill{
		Name: "writer",
	}

	err := s.Validate()
	if !errors.Is(err, ErrSkillInstructionsMissing) {
		t.Fatalf("expected ErrSkillInstructionsMissing, got %v", err)
	}
}

func TestSkillValidateDuplicateToolNamesFail(t *testing.T) {
	s := Skill{
		Name:         "writer",
		Instructions: "follow the task",
		Tools:        []string{"filesystem", "filesystem"},
	}

	err := s.Validate()
	if !errors.Is(err, ErrDuplicateToolName) {
		t.Fatalf("expected ErrDuplicateToolName, got %v", err)
	}
}

func TestSkillResolveTools(t *testing.T) {
	registry := tool.NewRegistry()
	if err := registry.Register(tool.FinishTool{}); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}

	s, err := NewSkill("writer", "", "writes", []string{"finish"}, "", "follow repo")
	if err != nil {
		t.Fatalf("unexpected new skill error: %v", err)
	}

	resolved, err := s.ResolveTools(registry)
	if err != nil {
		t.Fatalf("unexpected resolve error: %v", err)
	}
	if len(resolved) != 1 || resolved[0].Spec().Name != "finish" {
		t.Fatalf("unexpected resolved tools: %+v", resolved)
	}

	bound, err := s.AttachResolvedTools(resolved)
	if err != nil {
		t.Fatalf("unexpected attach error: %v", err)
	}
	if len(bound.BoundTools) != 1 {
		t.Fatalf("unexpected bound tool count: %d", len(bound.BoundTools))
	}
}

func TestSkillResolveMissingToolFails(t *testing.T) {
	registry := tool.NewRegistry()
	s, err := NewSkill("writer", "", "writes", []string{"missing"}, "", "follow repo")
	if err != nil {
		t.Fatalf("unexpected new skill error: %v", err)
	}

	_, err = s.ResolveTools(registry)
	if err == nil {
		t.Fatalf("expected resolve error")
	}
}

func TestSkillAttachResolvedToolsDuplicateFails(t *testing.T) {
	s, err := NewSkill("writer", "", "writes", []string{"finish"}, "", "follow repo")
	if err != nil {
		t.Fatalf("unexpected new skill error: %v", err)
	}

	_, err = s.AttachResolvedTools([]tool.Tool{tool.FinishTool{}, tool.FinishTool{}})
	if !errors.Is(err, ErrDuplicateToolName) {
		t.Fatalf("expected ErrDuplicateToolName, got %v", err)
	}
}

func TestParseSkillMinimalMarkdown(t *testing.T) {
	content := `
# filesystem_writer

Description: Writes files into the workspace.
Version: v1

## Instructions
Write requested files carefully.
Return a concise result.

## Tools
- filesystem
- finish
`

	got, err := ParseSkill(strings.NewReader(content))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if got.Name != "filesystem_writer" {
		t.Fatalf("unexpected skill name: %q", got.Name)
	}
	if got.Version != "v1" {
		t.Fatalf("unexpected version: %q", got.Version)
	}
	if got.Description != "Writes files into the workspace." {
		t.Fatalf("unexpected description: %q", got.Description)
	}
	if !strings.Contains(got.Instructions, "Write requested files carefully.") {
		t.Fatalf("unexpected instructions: %q", got.Instructions)
	}
	if len(got.Tools) != 2 || got.Tools[0] != "filesystem" || got.Tools[1] != "finish" {
		t.Fatalf("unexpected tools: %+v", got.Tools)
	}
}

func TestParseSkillMetadataNameFallback(t *testing.T) {
	content := `
## Metadata
Name: metadata_name
Description: Example skill.

## Instructions
Follow the task.
`

	got, err := ParseSkill(strings.NewReader(content))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if got.Name != "metadata_name" {
		t.Fatalf("unexpected metadata fallback name: %q", got.Name)
	}
}

func TestParseSkillMissingInstructionsFails(t *testing.T) {
	content := `
# no_instructions
Description: missing instructions
`
	_, err := ParseSkill(strings.NewReader(content))
	if !errors.Is(err, ErrSkillInstructionsMissing) {
		t.Fatalf("expected ErrSkillInstructionsMissing, got %v", err)
	}
}

func TestParseSkillDedupesTools(t *testing.T) {
	content := `
# dedupe_tools

## Instructions
Follow the task.

## Tools
- filesystem
- filesystem
- finish
`
	got, err := ParseSkill(strings.NewReader(content))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(got.Tools) != 2 {
		t.Fatalf("expected deduped tools, got %+v", got.Tools)
	}
}

func TestLoadSkillFromDirectoryAndFile(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "writer")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("unexpected mkdir error: %v", err)
	}
	content := "# writer\n\n## Instructions\nWrite.\n"
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte(content), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	fromDir, err := LoadSkill(skillDir)
	if err != nil {
		t.Fatalf("unexpected load from dir error: %v", err)
	}
	if fromDir.SourcePath != skillFile {
		t.Fatalf("unexpected source path from dir: %q", fromDir.SourcePath)
	}

	fromFile, err := LoadSkill(skillFile)
	if err != nil {
		t.Fatalf("unexpected load from file error: %v", err)
	}
	if fromFile.Directory != skillDir {
		t.Fatalf("unexpected directory: %q", fromFile.Directory)
	}
}

func TestLoadSkillMissingOrBadPathFails(t *testing.T) {
	_, err := LoadSkill("/definitely/missing/SKILL.md")
	if err == nil {
		t.Fatalf("expected missing path error")
	}

	root := t.TempDir()
	file := filepath.Join(root, "plain.md")
	if err := os.WriteFile(file, []byte("# plain"), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	_, err = LoadSkill(file)
	if err == nil {
		t.Fatalf("expected invalid file path error")
	}
}

func TestExampleSkillsLoad(t *testing.T) {
	for _, path := range []string{
		"../examples/skills/simple-writer",
		"../examples/skills/cautious-writer",
	} {
		got, err := LoadSkill(path)
		if err != nil {
			t.Fatalf("load example skill %s: %v", path, err)
		}
		if got.Name == "" || got.Instructions == "" || len(got.Tools) == 0 {
			t.Fatalf("unexpected example skill content: %+v", got)
		}
	}
}
