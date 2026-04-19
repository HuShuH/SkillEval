package eval

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-skill-eval-go/agent"
	"agent-skill-eval-go/providers"
	"agent-skill-eval-go/skill"
	"agent-skill-eval-go/tool"
)

func TestRunnerRunCaseFinishAndCheckerPass(t *testing.T) {
	runner := Runner{}
	client := &providers.SequenceClient{
		Steps: []providers.FakeStep{
			{Response: providers.ChatResponse{Finish: &providers.FinishSignal{FinalAnswer: "done", Reason: "complete"}}},
		},
	}
	s, _ := skill.NewSkill("writer", "", "writes", nil, "", "follow repo")

	result, err := runner.RunCase(context.Background(), "run-1", agent.Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 3,
	}, s, Case{
		ID:       "case-1",
		Prompt:   "do it",
		Checkers: []CheckerConfig{{Type: CheckerExactMatch, Config: map[string]any{"value": "done"}}},
	})
	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if !result.CaseResult.Passed {
		t.Fatalf("expected passed result: %+v", result.CaseResult)
	}
	if result.CaseResult.StopReason != string(agent.StopReasonFinished) {
		t.Fatalf("unexpected stop reason: %s", result.CaseResult.StopReason)
	}
}

func TestRunnerRunCaseCheckerFail(t *testing.T) {
	runner := Runner{}
	client := &providers.SequenceClient{
		Steps: []providers.FakeStep{
			{Response: providers.ChatResponse{Finish: &providers.FinishSignal{FinalAnswer: "wrong", Reason: "complete"}}},
		},
	}
	s, _ := skill.NewSkill("writer", "", "writes", nil, "", "follow repo")

	result, err := runner.RunCase(context.Background(), "run-1", agent.Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 3,
	}, s, Case{
		ID:       "case-1",
		Prompt:   "do it",
		Checkers: []CheckerConfig{{Type: CheckerExactMatch, Config: map[string]any{"value": "done"}}},
	})
	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if result.CaseResult.Passed {
		t.Fatalf("expected failed result")
	}
}

func TestRunnerRunCaseToolExecutionsIncluded(t *testing.T) {
	root := t.TempDir()
	runner := Runner{Config: RunConfig{WorkspaceRoot: root}}
	client := &providers.SequenceClient{
		Steps: []providers.FakeStep{
			{Response: providers.ChatResponse{
				ToolCalls: []providers.ToolCall{
					{ToolName: "filesystem", Operation: tool.OpWriteFile, Input: map[string]any{"path": "notes.txt", "content": "hello"}},
				},
			}},
			{Response: providers.ChatResponse{Finish: &providers.FinishSignal{FinalAnswer: "done", Reason: "complete"}}},
		},
	}

	s, _ := skill.NewSkill("writer", "", "writes", []string{"filesystem"}, "", "follow repo")
	s, _ = s.AttachResolvedTools([]tool.Tool{
		tool.FilesystemTool{Config: tool.FilesystemConfig{WorkspaceRoot: root}},
	})

	result, err := runner.RunCase(context.Background(), "run-1", agent.Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 3,
	}, s, Case{
		ID:       "case-1",
		Prompt:   "write it",
		Checkers: []CheckerConfig{{Type: CheckerExactMatch, Config: map[string]any{"value": "done"}}},
	})
	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if len(result.CaseResult.ToolExecutions) != 1 {
		t.Fatalf("expected one tool execution, got %d", len(result.CaseResult.ToolExecutions))
	}
	data, err := os.ReadFile(filepath.Join(root, "notes.txt"))
	if err != nil {
		t.Fatalf("unexpected file read error: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestRunnerWrapsProviderError(t *testing.T) {
	providerErr := &providers.ProviderError{Class: providers.ErrorClassServer, Message: "provider failed", Retryable: true}
	runner := Runner{}
	client := &providers.SequenceClient{
		Steps: []providers.FakeStep{{Err: providerErr}},
	}
	s, _ := skill.NewSkill("writer", "", "writes", nil, "", "follow repo")

	result, err := runner.RunCase(context.Background(), "run-1", agent.Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 2,
	}, s, Case{ID: "case-1", Prompt: "do it"})
	if !errors.Is(err, providerErr) {
		t.Fatalf("expected provider error, got %v", err)
	}
	if result.CaseResult.Error == "" {
		t.Fatalf("expected wrapped error")
	}
	if result.CaseResult.ErrorClass != string(providers.ErrorClassServer) {
		t.Fatalf("expected error class, got %q", result.CaseResult.ErrorClass)
	}
}

func TestRunnerWrapsMaxIterations(t *testing.T) {
	runner := Runner{}
	client := &providers.SequenceClient{
		Steps: []providers.FakeStep{
			{Response: providers.ChatResponse{Message: providers.Message{Role: "assistant", Content: "one"}}},
			{Response: providers.ChatResponse{Message: providers.Message{Role: "assistant", Content: "two"}}},
		},
	}
	s, _ := skill.NewSkill("writer", "", "writes", nil, "", "follow repo")

	result, err := runner.RunCase(context.Background(), "run-1", agent.Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 2,
	}, s, Case{ID: "case-1", Prompt: "do it"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CaseResult.StopReason != string(agent.StopReasonMaxIterations) {
		t.Fatalf("unexpected stop reason: %s", result.CaseResult.StopReason)
	}
}

func TestRunnerRunPairReturnsBothSides(t *testing.T) {
	runner := Runner{}
	skillA, _ := skill.NewSkill("writer-a", "", "writes", nil, "", "follow repo")
	skillB, _ := skill.NewSkill("writer-b", "", "writes", nil, "", "follow repo")

	pair, err := runner.RunPair(context.Background(),
		Case{ID: "case-1", Prompt: "do it", Checkers: []CheckerConfig{{Type: CheckerNonEmpty}}},
		agent.Agent{
			Name:          "agent-a",
			Provider:      &providers.SequenceClient{Steps: []providers.FakeStep{{Response: providers.ChatResponse{Finish: &providers.FinishSignal{FinalAnswer: "A", Reason: "complete"}}}}},
			MaxIterations: 2,
		},
		skillA,
		agent.Agent{
			Name:          "agent-b",
			Provider:      &providers.SequenceClient{Steps: []providers.FakeStep{{Response: providers.ChatResponse{Finish: &providers.FinishSignal{FinalAnswer: "B", Reason: "complete"}}}}},
			MaxIterations: 2,
		},
		skillB,
	)
	if err != nil {
		t.Fatalf("unexpected pair error: %v", err)
	}
	if pair.A.CaseResult.FinalAnswer != "A" || pair.B.CaseResult.FinalAnswer != "B" {
		t.Fatalf("unexpected pair outputs: %+v", pair)
	}
	if pair.Score.Scored {
		t.Fatalf("expected unscored placeholder")
	}
}

func TestRunnerRunCaseTimeout(t *testing.T) {
	runner := Runner{}
	client := &providers.SequenceClient{
		Steps: []providers.FakeStep{{Err: &providers.ProviderError{Class: providers.ErrorClassTimeout, Message: "timed out", Retryable: true}}},
	}
	s, _ := skill.NewSkill("writer", "", "writes", nil, "", "follow repo")

	result, err := runner.RunCase(context.Background(), "run-1", agent.Agent{
		Name:          "agent-a",
		Provider:      client,
		MaxIterations: 2,
	}, s, Case{ID: "case-1", Prompt: "do it"})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if result.CaseResult.ErrorClass != string(providers.ErrorClassTimeout) {
		t.Fatalf("unexpected error class: %q", result.CaseResult.ErrorClass)
	}
	if result.CaseResult.FailedIteration != 1 {
		t.Fatalf("unexpected failed iteration: %d", result.CaseResult.FailedIteration)
	}
}

func TestRunnerRunPairTransientErrorOnOneSide(t *testing.T) {
	runner := Runner{}
	skillA, _ := skill.NewSkill("writer-a", "", "writes", nil, "", "follow repo")
	skillB, _ := skill.NewSkill("writer-b", "", "writes", nil, "", "follow repo")

	pair, err := runner.RunPair(context.Background(),
		Case{ID: "case-1", Prompt: "do it", Checkers: []CheckerConfig{{Type: CheckerNonEmpty}}},
		agent.Agent{
			Name:          "agent-a",
			Provider:      &providers.SequenceClient{Steps: []providers.FakeStep{{Err: &providers.ProviderError{Class: providers.ErrorClassRateLimit, Message: "rate limited", Retryable: true}}}},
			MaxIterations: 2,
			ProviderConfig: providers.Config{
				MaxRetries:   0,
				RetryBackoff: time.Millisecond,
			},
		},
		skillA,
		agent.Agent{
			Name:          "agent-b",
			Provider:      &providers.SequenceClient{Steps: []providers.FakeStep{{Response: providers.ChatResponse{Finish: &providers.FinishSignal{FinalAnswer: "B", Reason: "complete"}}}}},
			MaxIterations: 2,
		},
		skillB,
	)
	if err == nil {
		t.Fatalf("expected pair error")
	}
	if pair.A.CaseResult.ErrorClass != string(providers.ErrorClassRateLimit) {
		t.Fatalf("unexpected A error class: %q", pair.A.CaseResult.ErrorClass)
	}
	if pair.B.CaseResult.FinalAnswer != "B" {
		t.Fatalf("unexpected B result: %+v", pair.B.CaseResult)
	}
}
