// Package tool contains Phase 1 migration skeleton types for the new architecture.
// It currently provides only minimal tool contracts, registry, and controlled implementations.
package tool

import "context"

// FinishPayload is the explicit end-of-task payload emitted by an agent.
type FinishPayload struct {
	FinalAnswer string
	Reason      string
	Output      map[string]any
}

// FinishResult is the normalized finish tool result.
type FinishResult struct {
	Finished    bool
	FinalAnswer string
	Reason      string
	Output      map[string]any
}

// FinishTool marks an explicit end to one task run.
type FinishTool struct{}

func (f FinishTool) Spec() Spec {
	return Spec{
		Name:        "finish",
		Description: "explicitly finish the current run",
		Parameters: map[string]ParameterSpec{
			"final_answer": {
				Type:        TypeString,
				Description: "final answer for the task",
				Required:    true,
			},
			"reason": {
				Type:        TypeString,
				Description: "why the run is complete",
			},
			"output": {
				Type:        TypeObject,
				Description: "optional structured output payload",
			},
		},
		Required: []string{"final_answer"},
	}
}

func (f FinishTool) Execute(ctx context.Context, call Call) (Result, error) {
	payload := FinishPayload{}
	if answer, ok := call.Input["final_answer"].(string); ok {
		payload.FinalAnswer = answer
	}
	if reason, ok := call.Input["reason"].(string); ok {
		payload.Reason = reason
	}
	if output, ok := call.Input["output"].(map[string]any); ok {
		payload.Output = output
	}

	finished := f.Finish(payload)
	return Result{
		Status:   StatusOK,
		Output:   finished.Output,
		Message:  finished.FinalAnswer,
		Finished: finished.Finished,
		Final:    true,
		Structured: map[string]any{
			"reason": finished.Reason,
		},
	}, nil
}

// Finish converts a payload into the canonical finish result.
func (f FinishTool) Finish(payload FinishPayload) FinishResult {
	return FinishResult{
		Finished:    true,
		FinalAnswer: payload.FinalAnswer,
		Reason:      payload.Reason,
		Output:      payload.Output,
	}
}
