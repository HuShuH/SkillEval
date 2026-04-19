package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-skill-eval-go/providers"
	"agent-skill-eval-go/skill"
	"agent-skill-eval-go/tool"
)

func TestOrchestratorAssistantMessageThenFinish(t *testing.T) {
	client := &providers.SequenceClient{
		Steps: []providers.FakeStep{
			{Response: providers.ChatResponse{Message: providers.Message{Role: "assistant", Content: "thinking"}}},
			{Response: providers.ChatResponse{Finish: &providers.FinishSignal{FinalAnswer: "done", Reason: "complete"}}},
		},
	}

	orch := Orchestrator{}
	runContext := &RunContext{RunID: "run-1"}
	s, err := skill.NewSkill("writer", "", "writes", nil, "", "follow repo")
	if err != nil {
		t.Fatalf("unexpected skill error: %v", err)
	}

	result, err := orch.Run(context.Background(), Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 4,
	}, s, runContext, "do the task")
	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if result.StopReason != StopReasonFinished {
		t.Fatalf("unexpected stop reason: %s", result.StopReason)
	}
	if result.FinalAnswer != "done" {
		t.Fatalf("unexpected final answer: %q", result.FinalAnswer)
	}
	if len(result.AssistantMessages) != 1 {
		t.Fatalf("unexpected assistant message count: %d", len(result.AssistantMessages))
	}
	if !hasEventType(runContext.Events, "run.started") || !hasEventType(runContext.Events, "run.finished") {
		t.Fatalf("expected run start/finish events: %+v", runContext.Events)
	}
}

func TestOrchestratorFinishToolEndsRun(t *testing.T) {
	client := &providers.SequenceClient{
		Steps: []providers.FakeStep{
			{Response: providers.ChatResponse{
				ToolCalls: []providers.ToolCall{
					{ToolName: "finish", Input: map[string]any{"final_answer": "all set", "reason": "done"}},
				},
			}},
		},
	}

	s, _ := skill.NewSkill("writer", "", "writes", []string{"finish"}, "", "follow repo")
	s, err := s.AttachResolvedTools([]tool.Tool{tool.FinishTool{}})
	if err != nil {
		t.Fatalf("unexpected attach error: %v", err)
	}

	result, err := Orchestrator{}.Run(context.Background(), Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 2,
	}, s, &RunContext{}, "complete")
	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if result.FinalAnswer != "all set" || result.StopReason != StopReasonFinished {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestOrchestratorFilesystemToolFlow(t *testing.T) {
	root := t.TempDir()
	client := &providers.SequenceClient{
		Steps: []providers.FakeStep{
			{Response: providers.ChatResponse{
				ToolCalls: []providers.ToolCall{
					{ToolName: "filesystem", Operation: tool.OpWriteFile, Input: map[string]any{"path": "notes.txt", "content": "hello"}},
				},
			}},
			{Response: providers.ChatResponse{
				ToolCalls: []providers.ToolCall{
					{ToolName: "filesystem", Operation: tool.OpReadFile, Input: map[string]any{"path": "notes.txt"}},
				},
			}},
			{Response: providers.ChatResponse{
				Finish: &providers.FinishSignal{FinalAnswer: "written and read", Reason: "complete"},
			}},
		},
	}

	s, _ := skill.NewSkill("writer", "", "writes", []string{"filesystem", "finish"}, "", "follow repo")
	s, err := s.AttachResolvedTools([]tool.Tool{
		tool.FilesystemTool{Config: tool.FilesystemConfig{WorkspaceRoot: root}},
		tool.FinishTool{},
	})
	if err != nil {
		t.Fatalf("unexpected attach error: %v", err)
	}

	result, err := Orchestrator{}.Run(context.Background(), Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 5,
	}, s, &RunContext{Workspace: root}, "write a file")
	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if len(result.ToolExecutions) != 2 {
		t.Fatalf("unexpected tool execution count: %d", len(result.ToolExecutions))
	}
	data, err := os.ReadFile(filepath.Join(root, "notes.txt"))
	if err != nil {
		t.Fatalf("unexpected readback error: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestOrchestratorToolNotFound(t *testing.T) {
	client := &providers.SequenceClient{
		Steps: []providers.FakeStep{
			{Response: providers.ChatResponse{
				ToolCalls: []providers.ToolCall{
					{ToolName: "missing", Operation: "do"},
				},
			}},
		},
	}

	s, _ := skill.NewSkill("writer", "", "writes", nil, "", "follow repo")
	result, err := Orchestrator{}.Run(context.Background(), Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 2,
	}, s, &RunContext{}, "do it")
	if err == nil {
		t.Fatalf("expected tool not found error")
	}
	if result.StopReason != StopReasonToolNotFound {
		t.Fatalf("unexpected stop reason: %s", result.StopReason)
	}
}

func TestOrchestratorProviderError(t *testing.T) {
	providerErr := errors.New("provider failed")
	client := &providers.SequenceClient{
		Steps: []providers.FakeStep{{Err: providerErr}},
	}

	s, _ := skill.NewSkill("writer", "", "writes", nil, "", "follow repo")
	result, err := Orchestrator{}.Run(context.Background(), Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 2,
	}, s, &RunContext{}, "do it")
	if !errors.Is(err, providerErr) {
		t.Fatalf("expected provider error, got %v", err)
	}
	if result.StopReason != StopReasonProviderError {
		t.Fatalf("unexpected stop reason: %s", result.StopReason)
	}
}

func TestOrchestratorMaxIterations(t *testing.T) {
	client := &providers.SequenceClient{
		Steps: []providers.FakeStep{
			{Response: providers.ChatResponse{Message: providers.Message{Role: "assistant", Content: "one"}}},
			{Response: providers.ChatResponse{Message: providers.Message{Role: "assistant", Content: "two"}}},
		},
	}

	s, _ := skill.NewSkill("writer", "", "writes", nil, "", "follow repo")
	result, err := Orchestrator{}.Run(context.Background(), Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 2,
	}, s, &RunContext{}, "do it")
	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if result.StopReason != StopReasonMaxIterations {
		t.Fatalf("unexpected stop reason: %s", result.StopReason)
	}
	if result.Iterations != 2 {
		t.Fatalf("unexpected iterations: %d", result.Iterations)
	}
}

func TestOrchestratorInvalidToolCallArgumentsFailBeforeExecution(t *testing.T) {
	client := &providers.SequenceClient{
		Steps: []providers.FakeStep{
			{Response: providers.ChatResponse{
				ToolCalls: []providers.ToolCall{
					{ToolName: "finish", Input: map[string]any{"reason": "done"}},
				},
			}},
		},
	}

	s, _ := skill.NewSkill("writer", "", "writes", []string{"finish"}, "", "follow repo")
	s, _ = s.AttachResolvedTools([]tool.Tool{tool.FinishTool{}})

	result, err := Orchestrator{}.Run(context.Background(), Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 2,
	}, s, &RunContext{}, "complete")
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if result.StopReason != StopReasonInvalidResponse {
		t.Fatalf("unexpected stop reason: %s", result.StopReason)
	}
	if !hasEventType((&RunContext{Events: []Event{}}).Events, "tool.validation.failed") {
	}
}

func TestOrchestratorEmitsToolValidationFailedEvent(t *testing.T) {
	client := &providers.SequenceClient{
		Steps: []providers.FakeStep{
			{Response: providers.ChatResponse{
				ToolCalls: []providers.ToolCall{
					{ToolName: "finish", Input: map[string]any{"reason": "done"}},
				},
			}},
		},
	}

	s, _ := skill.NewSkill("writer", "", "writes", []string{"finish"}, "", "follow repo")
	s, _ = s.AttachResolvedTools([]tool.Tool{tool.FinishTool{}})
	runContext := &RunContext{}

	_, err := Orchestrator{}.Run(context.Background(), Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 2,
	}, s, runContext, "complete")
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !hasEventType(runContext.Events, "tool.validation.failed") {
		t.Fatalf("expected tool.validation.failed event, got %+v", runContext.Events)
	}
}

func TestOrchestratorProviderTimeout(t *testing.T) {
	timeoutErr := &providers.ProviderError{Class: providers.ErrorClassTimeout, Message: "request timed out", Retryable: true}
	client := &providers.SequenceClient{Steps: []providers.FakeStep{{Err: timeoutErr}}}
	s, _ := skill.NewSkill("writer", "", "writes", nil, "", "follow repo")
	runContext := &RunContext{}

	result, err := Orchestrator{}.Run(context.Background(), Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 2,
	}, s, runContext, "do it")
	if !errors.Is(err, timeoutErr) {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if result.StopReason != StopReasonTimedOut {
		t.Fatalf("unexpected stop reason: %s", result.StopReason)
	}
	if result.ErrorClass != string(providers.ErrorClassTimeout) {
		t.Fatalf("unexpected error class: %q", result.ErrorClass)
	}
	if !hasEventType(runContext.Events, "provider.request.failed") {
		t.Fatalf("expected provider.request.failed event")
	}
}

func TestOrchestratorRetriesRetryableProviderError(t *testing.T) {
	client := &providers.SequenceClient{
		Steps: []providers.FakeStep{
			{Err: &providers.ProviderError{Class: providers.ErrorClassRateLimit, Message: "rate limited", Retryable: true, StatusCode: 429}},
			{Response: providers.ChatResponse{Finish: &providers.FinishSignal{FinalAnswer: "done", Reason: "complete"}}},
		},
	}
	s, _ := skill.NewSkill("writer", "", "writes", nil, "", "follow repo")
	runContext := &RunContext{}

	result, err := Orchestrator{}.Run(context.Background(), Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 2,
		ProviderConfig: providers.Config{
			MaxRetries:   1,
			RetryBackoff: time.Millisecond,
		},
	}, s, runContext, "do it")
	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if result.StopReason != StopReasonFinished {
		t.Fatalf("unexpected stop reason: %s", result.StopReason)
	}
	if !hasEventType(runContext.Events, "provider.request.retried") {
		t.Fatalf("expected provider.request.retried event")
	}
}

func TestOrchestratorCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &providers.SequenceClient{}
	s, _ := skill.NewSkill("writer", "", "writes", nil, "", "follow repo")
	runContext := &RunContext{}

	result, err := Orchestrator{}.Run(ctx, Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 2,
	}, s, runContext, "do it")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled error, got %v", err)
	}
	if result.StopReason != StopReasonCanceled {
		t.Fatalf("unexpected stop reason: %s", result.StopReason)
	}
	if !hasEventType(runContext.Events, "run.canceled") {
		t.Fatalf("expected run.canceled event")
	}
}

func hasEventType(events []Event, want string) bool {
	for _, event := range events {
		if event.Type == want {
			return true
		}
	}
	return false
}
