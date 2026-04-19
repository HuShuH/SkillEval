package tool

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesystemToolResolvePathWithinWorkspace(t *testing.T) {
	root := t.TempDir()
	fs := FilesystemTool{Config: FilesystemConfig{WorkspaceRoot: root}}

	got, err := fs.ResolvePath("notes/output.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "notes", "output.txt")
	if got != want {
		t.Fatalf("unexpected resolved path: got %q want %q", got, want)
	}
}

func TestFilesystemToolResolvePathRejectsEscape(t *testing.T) {
	root := t.TempDir()
	fs := FilesystemTool{Config: FilesystemConfig{WorkspaceRoot: root}}

	_, err := fs.ResolvePath("../outside.txt")
	if !errors.Is(err, ErrPathNotAllowed) {
		t.Fatalf("expected ErrPathNotAllowed, got %v", err)
	}
}

func TestFilesystemToolReadWriteHelpers(t *testing.T) {
	root := t.TempDir()
	fs := FilesystemTool{Config: FilesystemConfig{WorkspaceRoot: root}}

	if err := fs.WriteFile("data/result.txt", []byte("ok")); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	got, err := fs.ReadFile("data/result.txt")
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if string(got) != "ok" {
		t.Fatalf("unexpected file contents: %q", string(got))
	}

	onDisk, err := os.ReadFile(filepath.Join(root, "data", "result.txt"))
	if err != nil {
		t.Fatalf("unexpected disk read error: %v", err)
	}
	if string(onDisk) != "ok" {
		t.Fatalf("unexpected disk contents: %q", string(onDisk))
	}
}

func TestFilesystemToolReadOnlyWriteRejected(t *testing.T) {
	root := t.TempDir()
	fs := FilesystemTool{Config: FilesystemConfig{WorkspaceRoot: root, ReadOnly: true}}

	err := fs.WriteFile("result.txt", []byte("nope"))
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("expected os.ErrPermission, got %v", err)
	}
}

func TestFilesystemToolListDir(t *testing.T) {
	root := t.TempDir()
	fs := FilesystemTool{Config: FilesystemConfig{WorkspaceRoot: root}}

	if err := fs.WriteFile("dir/a.txt", []byte("a")); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if err := fs.WriteFile("dir/b.txt", []byte("b")); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	entries, err := fs.ListDir("dir")
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("unexpected entry count: %d", len(entries))
	}
}

func TestFilesystemToolExecuteSupportsReadWriteAndList(t *testing.T) {
	root := t.TempDir()
	fs := FilesystemTool{Config: FilesystemConfig{WorkspaceRoot: root}}

	writeResult, err := fs.Execute(nil, Call{
		ToolName:  "filesystem",
		Operation: OpWriteFile,
		Input: map[string]any{
			"path":    "docs/out.txt",
			"content": "hello",
		},
	})
	if err != nil {
		t.Fatalf("unexpected write execute error: %v", err)
	}
	if writeResult.Status != StatusOK {
		t.Fatalf("unexpected write status: %s", writeResult.Status)
	}

	readResult, err := fs.Execute(nil, Call{
		ToolName:  "filesystem",
		Operation: OpReadFile,
		Input:     map[string]any{"path": "docs/out.txt"},
	})
	if err != nil {
		t.Fatalf("unexpected read execute error: %v", err)
	}
	if got := readResult.Output["content"].(string); got != "hello" {
		t.Fatalf("unexpected read content: %q", got)
	}

	listResult, err := fs.Execute(nil, Call{
		ToolName:  "filesystem",
		Operation: OpListDir,
		Input:     map[string]any{"path": "docs"},
	})
	if err != nil {
		t.Fatalf("unexpected list execute error: %v", err)
	}
	if len(listResult.Output["entries"].([]string)) != 1 {
		t.Fatalf("unexpected listed entries: %+v", listResult.Output["entries"])
	}
}

func TestFilesystemToolMissingFileReturnsClearError(t *testing.T) {
	root := t.TempDir()
	fs := FilesystemTool{Config: FilesystemConfig{WorkspaceRoot: root}}

	_, err := fs.ReadFile("missing.txt")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
	if !strings.Contains(err.Error(), "missing.txt") {
		t.Fatalf("expected file path in error, got %v", err)
	}
}
