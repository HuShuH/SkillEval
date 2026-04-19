// Package tool contains Phase 1 migration skeleton types for the new architecture.
// It currently provides only minimal tool contracts, registry, and controlled implementations.
package tool

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrPathNotAllowed = errors.New("path is outside allowed workspace scope")

const (
	OpReadFile  = "read_file"
	OpWriteFile = "write_file"
	OpListDir   = "list_dir"
)

// FilesystemConfig defines workspace-scoped filesystem access.
type FilesystemConfig struct {
	WorkspaceRoot string
	AllowedPaths  []string
	ReadOnly      bool
}

// FilesystemTool is a workspace-bound filesystem tool skeleton.
type FilesystemTool struct {
	Config FilesystemConfig
}

func (f FilesystemTool) Spec() Spec {
	return Spec{
		Name:        "filesystem",
		Description: "workspace-bound file access",
		Parameters: map[string]ParameterSpec{
			"operation": {
				Type:        TypeString,
				Description: "filesystem action",
				Required:    true,
				Enum:        []string{OpReadFile, OpWriteFile, OpListDir},
			},
			"path": {
				Type:        TypeString,
				Description: "path inside the workspace",
				Required:    true,
			},
			"content": {
				Type:        TypeString,
				Description: "file contents for write_file",
			},
		},
		Required: []string{"operation", "path"},
	}
}

func (f FilesystemTool) Execute(ctx context.Context, call Call) (Result, error) {
	path, _ := call.Input["path"].(string)
	switch call.Operation {
	case OpReadFile:
		content, err := f.ReadFile(path)
		if err != nil {
			return Result{Status: StatusError}, err
		}
		return Result{
			Status: StatusOK,
			Output: map[string]any{
				"path":    path,
				"content": string(content),
			},
		}, nil
	case OpWriteFile:
		data, err := toBytes(call.Input["content"])
		if err != nil {
			return Result{Status: StatusError}, err
		}
		if err := f.WriteFile(path, data); err != nil {
			return Result{Status: StatusError}, err
		}
		return Result{
			Status:  StatusOK,
			Message: "file written",
			Output:  map[string]any{"path": path},
		}, nil
	case OpListDir:
		entries, err := f.ListDir(path)
		if err != nil {
			return Result{Status: StatusError}, err
		}
		return Result{
			Status: StatusOK,
			Output: map[string]any{
				"path":    path,
				"entries": entries,
			},
		}, nil
	default:
		return Result{Status: StatusError}, fmt.Errorf("%w: unsupported filesystem operation %q", ErrInvalidTool, call.Operation)
	}
}

// ResolvePath resolves a relative or absolute path inside the allowed workspace scope.
func (f FilesystemTool) ResolvePath(path string) (string, error) {
	root := filepath.Clean(f.Config.WorkspaceRoot)
	if root == "." || root == "" {
		return "", ErrPathNotAllowed
	}

	var candidate string
	if filepath.IsAbs(path) {
		candidate = filepath.Clean(path)
	} else {
		candidate = filepath.Join(root, path)
	}

	allowedRoots := f.allowedRoots(root)
	for _, allowed := range allowedRoots {
		if isWithinBase(candidate, allowed) {
			return candidate, nil
		}
	}

	return "", ErrPathNotAllowed
}

// ReadFile reads a file from the allowed workspace scope.
func (f FilesystemTool) ReadFile(path string) ([]byte, error) {
	resolved, err := f.ResolvePath(path)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	return content, nil
}

// WriteFile writes a file inside the allowed workspace scope.
func (f FilesystemTool) WriteFile(path string, data []byte) error {
	if f.Config.ReadOnly {
		return os.ErrPermission
	}

	resolved, err := f.ResolvePath(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(resolved, data, 0o644); err != nil {
		return fmt.Errorf("write file %q: %w", path, err)
	}
	return nil
}

// ListDir lists one directory inside the allowed workspace scope.
func (f FilesystemTool) ListDir(path string) ([]string, error) {
	if path == "" {
		path = "."
	}
	resolved, err := f.ResolvePath(path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, fmt.Errorf("list dir %q: %w", path, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
}

func (f FilesystemTool) allowedRoots(workspaceRoot string) []string {
	if len(f.Config.AllowedPaths) == 0 {
		return []string{workspaceRoot}
	}

	allowed := make([]string, 0, len(f.Config.AllowedPaths))
	for _, path := range f.Config.AllowedPaths {
		if filepath.IsAbs(path) {
			allowed = append(allowed, filepath.Clean(path))
			continue
		}
		allowed = append(allowed, filepath.Join(workspaceRoot, path))
	}
	return allowed
}

func isWithinBase(path string, base string) bool {
	path = filepath.Clean(path)
	base = filepath.Clean(base)
	if path == base {
		return true
	}
	separator := string(filepath.Separator)
	return strings.HasPrefix(path, base+separator)
}

func toBytes(value any) ([]byte, error) {
	switch data := value.(type) {
	case string:
		return []byte(data), nil
	case []byte:
		return data, nil
	default:
		return nil, fmt.Errorf("%w: content must be string or []byte", ErrInvalidTool)
	}
}
