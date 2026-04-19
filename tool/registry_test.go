package tool

import (
	"errors"
	"testing"
)

func TestRegistryRegisterLookupAndList(t *testing.T) {
	registry := NewRegistry()

	if err := registry.Register(FinishTool{}); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}
	if err := registry.Register(BashTool{}); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}

	found, err := registry.Lookup("finish")
	if err != nil {
		t.Fatalf("unexpected lookup error: %v", err)
	}
	if found.Spec().Name != "finish" {
		t.Fatalf("unexpected tool name: %q", found.Spec().Name)
	}

	got := registry.List()
	if len(got) != 2 || got[0].Spec().Name != "finish" || got[1].Spec().Name != "bash" {
		t.Fatalf("unexpected list order")
	}
}

func TestRegistryDuplicateRegisterFails(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(FinishTool{}); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}

	err := registry.Register(FinishTool{})
	if !errors.Is(err, ErrDuplicateTool) {
		t.Fatalf("expected ErrDuplicateTool, got %v", err)
	}
}

func TestRegistryLookupMissingToolFails(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Lookup("missing")
	if !errors.Is(err, ErrToolNotFound) {
		t.Fatalf("expected ErrToolNotFound, got %v", err)
	}
}
