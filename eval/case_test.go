package eval

import (
	"errors"
	"testing"
)

func TestCaseValidateSuccess(t *testing.T) {
	c := Case{
		ID:     "case-1",
		Prompt: "summarize the file",
		Expected: Expected{
			FinalText: "done",
		},
		Checkers: []CheckerConfig{
			{Type: CheckerExactMatch, Config: map[string]any{"value": "done"}},
		},
		Metadata: map[string]string{"suite": "ab"},
		Tags:     []string{"smoke"},
	}

	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
}

func TestCaseValidateEmptyIDFails(t *testing.T) {
	err := (Case{Prompt: "do it"}).Validate()
	if !errors.Is(err, ErrCaseIDRequired) {
		t.Fatalf("expected ErrCaseIDRequired, got %v", err)
	}
}

func TestCaseValidateEmptyPromptAndTaskFail(t *testing.T) {
	err := (Case{ID: "case-1"}).Validate()
	if !errors.Is(err, ErrCaseTaskEmpty) {
		t.Fatalf("expected ErrCaseTaskEmpty, got %v", err)
	}
}
