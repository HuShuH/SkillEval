package tool

import (
	"context"
	"testing"
)

func TestFinishToolFinish(t *testing.T) {
	finish := FinishTool{}

	got := finish.Finish(FinishPayload{
		FinalAnswer: "done",
		Reason:      "task completed",
		Output:      map[string]any{"file": "report.json"},
	})
	if !got.Finished {
		t.Fatalf("expected finished result")
	}
	if got.FinalAnswer != "done" {
		t.Fatalf("unexpected final answer: %q", got.FinalAnswer)
	}
}

func TestFinishToolExecute(t *testing.T) {
	finish := FinishTool{}

	got, err := finish.Execute(context.Background(), Call{
		ToolName: "finish",
		Input: map[string]any{
			"final_answer": "all set",
			"reason":       "completed",
			"output":       map[string]any{"ok": true},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Finished {
		t.Fatalf("expected finished result")
	}
	if got.Message != "all set" {
		t.Fatalf("unexpected message: %q", got.Message)
	}
	if !got.Final {
		t.Fatalf("expected final result")
	}
	if got.Structured["reason"].(string) != "completed" {
		t.Fatalf("unexpected reason: %+v", got.Structured)
	}
}
