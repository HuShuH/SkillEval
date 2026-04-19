// Package tool contains Phase 1 migration skeleton types for the new architecture.
// It currently provides only minimal tool contracts, registry, and controlled implementations.
package tool

import (
	"context"
	"errors"
)

var ErrNotImplemented = errors.New("tool execution not implemented")

// Status represents the outcome status of one tool invocation.
type Status string

const (
	StatusOK    Status = "ok"
	StatusError Status = "error"
)

// FieldType describes the minimal supported parameter types.
type FieldType string

const (
	TypeString  FieldType = "string"
	TypeInteger FieldType = "integer"
	TypeBoolean FieldType = "boolean"
	TypeObject  FieldType = "object"
	TypeArray   FieldType = "array"
)

// ParameterSpec describes one tool parameter for provider/tool alignment.
type ParameterSpec struct {
	Type        FieldType `json:"type"`
	Description string    `json:"description,omitempty"`
	Required    bool      `json:"required,omitempty"`
	Enum        []string  `json:"enum,omitempty"`
}

// Spec describes a tool's static metadata.
type Spec struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	Parameters  map[string]ParameterSpec `json:"parameters,omitempty"`
	Required    []string                 `json:"required,omitempty"`
}

// Call describes one tool invocation request.
type Call struct {
	ToolName  string         `json:"tool_name"`
	Operation string         `json:"operation,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
}

// Result describes one tool execution result.
type Result struct {
	Status     Status         `json:"status"`
	Output     map[string]any `json:"output,omitempty"`
	Message    string         `json:"message,omitempty"`
	Finished   bool           `json:"finished"`
	Final      bool           `json:"final"`
	Structured map[string]any `json:"structured,omitempty"`
}

// Tool defines the minimal execution contract for the new architecture.
type Tool interface {
	Spec() Spec
	Execute(ctx context.Context, call Call) (Result, error)
}

// StaticTool is a small helper implementation for tests and stubs.
type StaticTool struct {
	ToolSpec Spec
	Result   Result
	Err      error
}

func (t StaticTool) Spec() Spec {
	return t.ToolSpec
}

func (t StaticTool) Execute(ctx context.Context, call Call) (Result, error) {
	if t.Err != nil {
		return Result{}, t.Err
	}
	if t.Result.Status == "" {
		t.Result.Status = StatusOK
	}
	return t.Result, nil
}
