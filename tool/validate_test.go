package tool

import (
	"errors"
	"testing"
)

func TestValidateCallRequiredFieldsAndTypes(t *testing.T) {
	spec := FinishTool{}.Spec()

	err := ValidateCall(spec, Call{ToolName: "finish", Input: map[string]any{}})
	if !errors.Is(err, ErrInvalidTool) {
		t.Fatalf("expected invalid tool error, got %v", err)
	}

	err = ValidateCall(spec, Call{
		ToolName: "finish",
		Input:    map[string]any{"final_answer": 123},
	})
	if !errors.Is(err, ErrInvalidTool) {
		t.Fatalf("expected invalid type error, got %v", err)
	}
}

func TestValidateCallFilesystemOperationAndContent(t *testing.T) {
	spec := FilesystemTool{}.Spec()

	err := ValidateCall(spec, Call{
		ToolName:  "filesystem",
		Operation: "bad_op",
		Input:     map[string]any{"path": "a.txt"},
	})
	if !errors.Is(err, ErrInvalidTool) {
		t.Fatalf("expected invalid operation error, got %v", err)
	}

	err = ValidateCall(spec, Call{
		ToolName:  "filesystem",
		Operation: OpWriteFile,
		Input:     map[string]any{"path": "a.txt"},
	})
	if !errors.Is(err, ErrInvalidTool) {
		t.Fatalf("expected missing content error, got %v", err)
	}

	err = ValidateCall(spec, Call{
		ToolName:  "filesystem",
		Operation: OpReadFile,
		Input:     map[string]any{"path": "a.txt"},
	})
	if err != nil {
		t.Fatalf("unexpected valid filesystem call error: %v", err)
	}
}

func TestValidateCallFinishRequiresFinalAnswer(t *testing.T) {
	err := ValidateCall(FinishTool{}.Spec(), Call{
		ToolName: "finish",
		Input:    map[string]any{"reason": "done"},
	})
	if !errors.Is(err, ErrInvalidTool) {
		t.Fatalf("expected invalid tool error, got %v", err)
	}
}
