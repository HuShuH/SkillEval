// Package eval contains Phase 1 migration skeleton types for the new architecture.
// It currently provides minimal runnable case definitions, validation, and runner types.
package eval

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidCase    = errors.New("invalid case")
	ErrCaseIDRequired = errors.New("case id is required")
	ErrCaseTaskEmpty  = errors.New("case prompt or task is required")
)

// CheckerConfig defines explicit evaluation settings for one case.
type CheckerConfig struct {
	Type   string
	Config map[string]any
}

// Expected defines minimal expected outputs for a case.
type Expected struct {
	OutputFiles []string
	FinalText   string
}

// Case defines one runnable evaluation case in the new architecture.
type Case struct {
	ID       string
	Prompt   string
	Task     string
	Expected Expected
	Checkers []CheckerConfig
	Metadata map[string]string
	Tags     []string
}

// NewCase constructs and validates a minimal case.
func NewCase(id, prompt, task string) (Case, error) {
	c := Case{
		ID:     strings.TrimSpace(id),
		Prompt: strings.TrimSpace(prompt),
		Task:   strings.TrimSpace(task),
	}
	if err := c.Validate(); err != nil {
		return Case{}, err
	}
	return c, nil
}

// Validate checks whether the case is usable for a run.
func (c Case) Validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return fmt.Errorf("%w: %w", ErrInvalidCase, ErrCaseIDRequired)
	}
	if strings.TrimSpace(c.Prompt) == "" && strings.TrimSpace(c.Task) == "" {
		return fmt.Errorf("%w: %w", ErrInvalidCase, ErrCaseTaskEmpty)
	}
	return nil
}
