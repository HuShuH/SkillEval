package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCasesJSONArray(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "cases.json")
	content := `[{"id":"case-1","prompt":"do it"},{"id":"case-2","task":"write it"}]`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	cases, err := LoadCases(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if len(cases) != 2 {
		t.Fatalf("unexpected case count: %d", len(cases))
	}
}

func TestLoadCasesJSONL(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "cases.jsonl")
	content := strings.Join([]string{
		`{"id":"case-1","prompt":"do it"}`,
		`{"id":"case-2","task":"write it"}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	cases, err := LoadCases(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if len(cases) != 2 {
		t.Fatalf("unexpected case count: %d", len(cases))
	}
}

func TestLoadCasesInvalidCaseFails(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "cases.json")
	content := `[{"id":"","prompt":"do it"}]`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	_, err := LoadCases(path)
	if err == nil {
		t.Fatalf("expected invalid case error")
	}
}

func TestLoadCasesEmptyAndBadFormatFail(t *testing.T) {
	root := t.TempDir()
	emptyPath := filepath.Join(root, "empty.json")
	if err := os.WriteFile(emptyPath, []byte("   "), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if _, err := LoadCases(emptyPath); err == nil {
		t.Fatalf("expected empty file error")
	}

	badPath := filepath.Join(root, "bad.jsonl")
	if err := os.WriteFile(badPath, []byte(`{"id":"case-1"`), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if _, err := LoadCases(badPath); err == nil {
		t.Fatalf("expected bad format error")
	}
}

func TestExampleCasesLoad(t *testing.T) {
	for _, path := range []string{
		"../examples/cases/sample_single.json",
		"../examples/cases/sample_pair.json",
	} {
		cases, err := LoadCases(path)
		if err != nil {
			t.Fatalf("load example cases %s: %v", path, err)
		}
		if len(cases) == 0 {
			t.Fatalf("expected non-empty cases from %s", path)
		}
	}
}
