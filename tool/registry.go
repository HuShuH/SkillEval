// Package tool contains Phase 1 migration skeleton types for the new architecture.
// It currently provides only minimal tool contracts, registry, and controlled implementations.
package tool

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrDuplicateTool   = errors.New("tool already registered")
	ErrToolNotFound    = errors.New("tool not found")
	ErrInvalidTool     = errors.New("invalid tool")
	ErrToolNameMissing = errors.New("tool name is required")
)

// Registry stores tools by name for in-memory assembly.
type Registry struct {
	tools map[string]Tool
	order []string
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds one tool to the registry.
func (r *Registry) Register(t Tool) error {
	if r == nil {
		return fmt.Errorf("%w: registry is nil", ErrInvalidTool)
	}
	spec := t.Spec()
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return fmt.Errorf("%w: %w", ErrInvalidTool, ErrToolNameMissing)
	}
	if _, ok := r.tools[name]; ok {
		return fmt.Errorf("%w: %q", ErrDuplicateTool, name)
	}
	r.tools[name] = t
	r.order = append(r.order, name)
	return nil
}

// Lookup returns one tool by name.
func (r *Registry) Lookup(name string) (Tool, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: %q", ErrToolNotFound, name)
	}
	found, ok := r.tools[strings.TrimSpace(name)]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrToolNotFound, name)
	}
	return found, nil
}

// List returns registered tools in registration order.
func (r *Registry) List() []Tool {
	if r == nil {
		return nil
	}
	tools := make([]Tool, 0, len(r.order))
	for _, name := range r.order {
		tools = append(tools, r.tools[name])
	}
	return tools
}

// Names returns sorted tool names for inspection.
func (r *Registry) Names() []string {
	if r == nil {
		return nil
	}
	names := append([]string(nil), r.order...)
	sort.Strings(names)
	return names
}
