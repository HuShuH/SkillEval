// Package tool contains Phase 1 migration skeleton types for the new architecture.
// It currently provides only minimal tool contracts, registry, and controlled implementations.
package tool

import (
	"context"
	"errors"
	"time"
)

var ErrExecutionDisabled = errors.New("bash execution disabled")

// BashConfig describes future shell execution settings.
type BashConfig struct {
	WorkDir        string
	Shell          string
	Timeout        time.Duration
	AllowExecution bool
}

// BashRequest describes a future shell execution request.
type BashRequest struct {
	Command string
	Args    []string
}

// BashResult captures shell execution output.
type BashResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Status   Status
}

// BashTool is a placeholder shell tool.
type BashTool struct {
	Config BashConfig
}

func (b BashTool) Spec() Spec {
	return Spec{
		Name:        "bash",
		Description: "workspace shell execution",
		Parameters: map[string]ParameterSpec{
			"command": {
				Type:        TypeString,
				Description: "shell command to run",
				Required:    true,
			},
			"timeout_seconds": {
				Type:        TypeInteger,
				Description: "optional execution timeout in seconds",
			},
		},
		Required: []string{"command"},
	}
}

func (b BashTool) Execute(ctx context.Context, call Call) (Result, error) {
	req := BashRequest{}
	if command, ok := call.Input["command"].(string); ok {
		req.Command = command
	}
	if args, ok := call.Input["args"].([]string); ok {
		req.Args = args
	}
	runResult, err := b.Run(ctx, req)
	if err != nil {
		return Result{Status: StatusError}, err
	}
	return Result{
		Status: StatusOK,
		Output: map[string]any{
			"stdout":    runResult.Stdout,
			"stderr":    runResult.Stderr,
			"exit_code": runResult.ExitCode,
		},
	}, nil
}

// Run is a placeholder for future shell execution behavior.
func (b BashTool) Run(ctx context.Context, req BashRequest) (BashResult, error) {
	if !b.Config.AllowExecution {
		return BashResult{}, ErrExecutionDisabled
	}
	return BashResult{}, ErrNotImplemented
}
