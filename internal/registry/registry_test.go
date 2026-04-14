package registry

import (
	"path/filepath"
	"testing"
)

func TestLoadSkillsSuccess(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "skills")

	reg, err := LoadSkills(dir)
	if err != nil {
		t.Fatalf("LoadSkills returned error: %v", err)
	}

	skill, ok := reg.Get("hello_world")
	if !ok {
		t.Fatalf("expected hello_world skill to be loaded")
	}
	if skill.Name != "hello_world" {
		t.Fatalf("unexpected skill name: %q", skill.Name)
	}
}

func TestLoadSkillsMissingDir(t *testing.T) {
	_, err := LoadSkills(filepath.Join("..", "..", "testdata", "skills_missing"))
	if err == nil {
		t.Fatal("expected error for missing skills directory")
	}
}
