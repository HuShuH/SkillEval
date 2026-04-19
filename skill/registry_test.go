package skill

import (
	"errors"
	"testing"
)

func TestRegistryRegisterLookupAndList(t *testing.T) {
	registry := NewRegistry()
	writer, err := NewSkill("writer", " v1 ", "writes", []string{"finish"}, "", "follow repo")
	if err != nil {
		t.Fatalf("unexpected new skill error: %v", err)
	}
	reviewer, err := NewSkill("reviewer", "", "reviews", []string{"finish"}, "system", "")
	if err != nil {
		t.Fatalf("unexpected new skill error: %v", err)
	}

	if err := registry.Register(writer); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}
	if err := registry.Register(reviewer); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}

	found, err := registry.Lookup("writer")
	if err != nil {
		t.Fatalf("unexpected lookup error: %v", err)
	}
	if found.Version != "v1" {
		t.Fatalf("unexpected normalized version: %q", found.Version)
	}

	got := registry.List()
	if len(got) != 2 || got[0].Name != "writer" || got[1].Name != "reviewer" {
		t.Fatalf("unexpected list order: %+v", got)
	}
}

func TestRegistryDuplicateRegisterFails(t *testing.T) {
	registry := NewRegistry()
	s, _ := NewSkill("writer", "", "writes", []string{"finish"}, "", "follow repo")

	if err := registry.Register(s); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}
	err := registry.Register(s)
	if !errors.Is(err, ErrDuplicateSkill) {
		t.Fatalf("expected ErrDuplicateSkill, got %v", err)
	}
}

func TestRegistryLookupMissingSkillFails(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Lookup("missing")
	if !errors.Is(err, ErrSkillNotFound) {
		t.Fatalf("expected ErrSkillNotFound, got %v", err)
	}
}
