package tool

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBashToolConstructible(t *testing.T) {
	bash := BashTool{
		Config: BashConfig{
			WorkDir: "/tmp/work",
			Shell:   "/bin/bash",
			Timeout: 3 * time.Second,
		},
	}

	if bash.Spec().Name != "bash" {
		t.Fatalf("unexpected tool name: %q", bash.Spec().Name)
	}
}

func TestBashToolPlaceholderMethods(t *testing.T) {
	bash := BashTool{}

	_, err := bash.Run(context.Background(), BashRequest{Command: "pwd"})
	if !errors.Is(err, ErrExecutionDisabled) {
		t.Fatalf("expected ErrExecutionDisabled, got %v", err)
	}

	_, err = bash.Execute(context.Background(), Call{ToolName: "bash", Input: map[string]any{"command": "pwd"}})
	if !errors.Is(err, ErrExecutionDisabled) {
		t.Fatalf("expected ErrExecutionDisabled, got %v", err)
	}
}

func TestBashToolAllowsConfiguredPlaceholderExecution(t *testing.T) {
	bash := BashTool{
		Config: BashConfig{
			WorkDir:        "/tmp/work",
			Shell:          "/bin/bash",
			Timeout:        5 * time.Second,
			AllowExecution: true,
		},
	}

	if bash.Config.WorkDir != "/tmp/work" || bash.Config.Timeout != 5*time.Second {
		t.Fatalf("unexpected config: %+v", bash.Config)
	}

	_, err := bash.Run(context.Background(), BashRequest{Command: "pwd"})
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}
