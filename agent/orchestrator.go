// Package agent contains Phase 1 migration skeleton types for the new architecture.
// It currently provides a minimal in-memory orchestrator loop and structured events.
package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"agent-skill-eval-go/providers"
	"agent-skill-eval-go/skill"
	"agent-skill-eval-go/tool"
)

// StopReason describes why one orchestrator run ended.
type StopReason string

const (
	StopReasonFinished        StopReason = "finished"
	StopReasonMaxIterations   StopReason = "max_iterations"
	StopReasonToolError       StopReason = "tool_error"
	StopReasonProviderError   StopReason = "provider_error"
	StopReasonInvalidResponse StopReason = "invalid_response"
	StopReasonToolNotFound    StopReason = "tool_not_found"
	StopReasonCanceled        StopReason = "canceled"
	StopReasonTimedOut        StopReason = "timed_out"
)

// AssistantMessage records one assistant-visible text output.
type AssistantMessage struct {
	Iteration int    `json:"iteration"`
	Content   string `json:"content"`
}

// ToolExecutionRecord records one tool execution.
type ToolExecutionRecord struct {
	Iteration int            `json:"iteration"`
	ToolName  string         `json:"tool_name"`
	Operation string         `json:"operation,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	Result    tool.Result    `json:"result"`
	Error     string         `json:"error,omitempty"`
}

// RunResult is the final orchestrator result.
type RunResult struct {
	FinalAnswer       string                `json:"final_answer,omitempty"`
	StopReason        StopReason            `json:"stop_reason,omitempty"`
	Iterations        int                   `json:"iterations"`
	AssistantMessages []AssistantMessage    `json:"assistant_messages,omitempty"`
	ToolExecutions    []ToolExecutionRecord `json:"tool_executions,omitempty"`
	Error             string                `json:"error,omitempty"`
	ErrorClass        string                `json:"error_class,omitempty"`
	FailedIteration   int                   `json:"failed_iteration,omitempty"`
}

// Orchestrator runs the minimal provider -> tool -> event loop.
type Orchestrator struct{}

// Run executes one in-memory agent loop using the provided skill and run context.
func (o Orchestrator) Run(ctx context.Context, agentConfig Agent, selectedSkill skill.Skill, runContext *RunContext, prompt string) (RunResult, error) {
	if agentConfig.Provider == nil {
		err := errors.New("provider is required")
		return o.fail(runContext, 0, StopReasonProviderError, "run failed", err), err
	}
	if runContext == nil {
		runContext = &RunContext{}
	}

	runContext.AddMessage(Message{Role: "user", Content: prompt})
	runContext.EmitEvent("run.started", 0, "run started", map[string]any{
		"agent": agentConfig.Name,
		"skill": selectedSkill.Name,
	})

	availableTools := o.buildToolIndex(agentConfig, selectedSkill)
	availableToolList := o.buildToolList(agentConfig, selectedSkill)
	result := RunResult{}

	maxIterations := agentConfig.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 1
	}

	for iteration := 1; iteration <= maxIterations; iteration++ {
		if ctx.Err() != nil {
			return o.failWithContext(runContext, iteration, ctx.Err())
		}
		runContext.Iteration = iteration
		runContext.EmitEvent("iteration.started", iteration, "iteration started", nil)

		response, err := o.callProviderWithRetry(ctx, runContext, iteration, agentConfig, availableToolList)
		if err != nil {
			return o.failWithProvider(runContext, iteration, err), err
		}

		runContext.EmitEvent("provider.responded", iteration, "provider responded", map[string]any{
			"tool_calls": len(response.ToolCalls),
			"finish":     response.Finish != nil,
		})

		if response.Message.Content != "" {
			runContext.AddMessage(Message{Role: "assistant", Content: response.Message.Content})
			result.AssistantMessages = append(result.AssistantMessages, AssistantMessage{
				Iteration: iteration,
				Content:   response.Message.Content,
			})
		}

		if response.Finish != nil {
			result.FinalAnswer = response.Finish.FinalAnswer
			result.StopReason = StopReasonFinished
			result.Iterations = iteration
			runContext.EmitEvent("run.finished", iteration, "run finished", map[string]any{
				"reason":       response.Finish.Reason,
				"final_answer": response.Finish.FinalAnswer,
			})
			return result, nil
		}

		if len(response.ToolCalls) == 0 && response.Message.Content == "" {
			err := errors.New("provider returned no message, tool call, or finish signal")
			return o.fail(runContext, iteration, StopReasonInvalidResponse, "run failed", err), err
		}

		for _, toolCall := range response.ToolCalls {
			resolved, ok := availableTools[toolCall.ToolName]
			if !ok {
				err := fmt.Errorf("tool %q not found", toolCall.ToolName)
				return o.fail(runContext, iteration, StopReasonToolNotFound, "run failed", err), err
			}

			runContext.EmitEvent("tool.called", iteration, "tool called", map[string]any{
				"tool":      toolCall.ToolName,
				"operation": toolCall.Operation,
			})

			call := tool.Call{
				ToolName:  toolCall.ToolName,
				Operation: toolCall.Operation,
				Input:     toolCall.Input,
			}
			if err := tool.ValidateCall(resolved.Spec(), call); err != nil {
				runContext.EmitEvent("tool.validation.failed", iteration, "tool validation failed", map[string]any{
					"tool":        toolCall.ToolName,
					"operation":   toolCall.Operation,
					"error_class": "tool_validation_error",
					"error":       err.Error(),
				})
				record := ToolExecutionRecord{
					Iteration: iteration,
					ToolName:  toolCall.ToolName,
					Operation: toolCall.Operation,
					Input:     toolCall.Input,
					Error:     err.Error(),
				}
				result.ToolExecutions = append(result.ToolExecutions, record)
				return o.fail(runContext, iteration, StopReasonInvalidResponse, "run failed", err, result.ToolExecutions...), err
			}
			toolResult, err := resolved.Execute(ctx, call)
			record := ToolExecutionRecord{
				Iteration: iteration,
				ToolName:  toolCall.ToolName,
				Operation: toolCall.Operation,
				Input:     toolCall.Input,
				Result:    toolResult,
			}
			if err != nil {
				record.Error = err.Error()
				result.ToolExecutions = append(result.ToolExecutions, record)
				return o.fail(runContext, iteration, StopReasonToolError, "run failed", err, result.ToolExecutions...), err
			}
			result.ToolExecutions = append(result.ToolExecutions, record)
			runContext.EmitEvent("tool.completed", iteration, "tool completed", map[string]any{
				"tool":      toolCall.ToolName,
				"operation": toolCall.Operation,
				"status":    string(toolResult.Status),
				"final":     toolResult.Final,
			})

			runContext.AddMessage(Message{
				Role:    "tool",
				Content: toolResult.Message,
			})

			if toolResult.Final || toolResult.Finished {
				result.FinalAnswer = toolResult.Message
				result.StopReason = StopReasonFinished
				result.Iterations = iteration
				runContext.EmitEvent("run.finished", iteration, "run finished", map[string]any{
					"tool":         toolCall.ToolName,
					"final_answer": toolResult.Message,
				})
				return result, nil
			}
		}
	}

	result.StopReason = StopReasonMaxIterations
	result.Iterations = maxIterations
	runContext.EmitEvent("run.finished", maxIterations, "run finished", map[string]any{
		"reason": string(StopReasonMaxIterations),
	})
	return result, nil
}

func (o Orchestrator) callProviderWithRetry(ctx context.Context, runContext *RunContext, iteration int, agentConfig Agent, availableToolList []tool.Tool) (providers.ChatResponse, error) {
	maxAttempts := agentConfig.ProviderConfig.MaxRetries + 1
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	backoff := agentConfig.ProviderConfig.RetryBackoff
	if backoff <= 0 {
		backoff = 200 * time.Millisecond
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		runContext.EmitEvent("provider.request.started", iteration, "provider request started", map[string]any{
			"attempt": attempt,
		})
		response, err := agentConfig.Provider.ChatCompletion(ctx, providers.ChatRequest{
			Messages: toProviderMessages(runContext.Messages),
			Tools:    providers.ToolDefinitionsFromTools(availableToolList),
		})
		if err == nil {
			runContext.EmitEvent("provider.request.succeeded", iteration, "provider request succeeded", map[string]any{
				"attempt": attempt,
			})
			return response, nil
		}
		lastErr = err
		runContext.EmitEvent("provider.request.failed", iteration, "provider request failed", map[string]any{
			"attempt":     attempt,
			"error":       err.Error(),
			"error_class": providers.ErrorClassOf(err),
			"retryable":   providers.IsRetryable(err),
			"status_code": providers.StatusCodeOf(err),
		})
		if !providers.IsRetryable(err) || attempt == maxAttempts || ctx.Err() != nil {
			break
		}
		runContext.EmitEvent("provider.request.retried", iteration, "provider request retried", map[string]any{
			"attempt":      attempt,
			"next_attempt": attempt + 1,
			"backoff_ms":   backoff.Milliseconds(),
			"error_class":  providers.ErrorClassOf(err),
		})
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return providers.ChatResponse{}, ctx.Err()
		case <-timer.C:
		}
		backoff *= 2
	}
	return providers.ChatResponse{}, lastErr
}

func (o Orchestrator) buildToolIndex(agentConfig Agent, selectedSkill skill.Skill) map[string]tool.Tool {
	index := make(map[string]tool.Tool)
	for _, bound := range selectedSkill.BoundTools {
		index[bound.Spec().Name] = bound
	}
	for _, configured := range agentConfig.Tools {
		if _, ok := index[configured.Spec().Name]; ok {
			continue
		}
		index[configured.Spec().Name] = configured
	}
	return index
}

func (o Orchestrator) buildToolList(agentConfig Agent, selectedSkill skill.Skill) []tool.Tool {
	index := make(map[string]struct{})
	var tools []tool.Tool
	for _, bound := range selectedSkill.BoundTools {
		name := bound.Spec().Name
		if _, ok := index[name]; ok {
			continue
		}
		index[name] = struct{}{}
		tools = append(tools, bound)
	}
	for _, configured := range agentConfig.Tools {
		name := configured.Spec().Name
		if _, ok := index[name]; ok {
			continue
		}
		index[name] = struct{}{}
		tools = append(tools, configured)
	}
	return tools
}

func toProviderMessages(messages []Message) []providers.Message {
	converted := make([]providers.Message, 0, len(messages))
	for _, message := range messages {
		converted = append(converted, providers.Message{
			Role:    message.Role,
			Content: message.Content,
		})
	}
	return converted
}

func (o Orchestrator) fail(runContext *RunContext, iteration int, reason StopReason, message string, err error, records ...ToolExecutionRecord) RunResult {
	result := RunResult{
		StopReason:      reason,
		Iterations:      iteration,
		ToolExecutions:  append([]ToolExecutionRecord(nil), records...),
		Error:           err.Error(),
		ErrorClass:      providers.ErrorClassOf(err),
		FailedIteration: iteration,
	}
	if runContext != nil {
		runContext.EmitEvent("run.failed", iteration, message, map[string]any{
			"reason":           reason,
			"error":            err.Error(),
			"error_class":      providers.ErrorClassOf(err),
			"status_code":      providers.StatusCodeOf(err),
			"failed_iteration": iteration,
		})
	}
	return result
}

func (o Orchestrator) failWithProvider(runContext *RunContext, iteration int, err error) RunResult {
	switch {
	case errors.Is(err, context.Canceled):
		if runContext != nil {
			runContext.EmitEvent("run.canceled", iteration, "run canceled", map[string]any{"error_class": string(providers.ErrorClassCanceled)})
		}
		return o.fail(runContext, iteration, StopReasonCanceled, "run canceled", err)
	case errors.Is(err, context.DeadlineExceeded):
		if runContext != nil {
			runContext.EmitEvent("run.timed_out", iteration, "run timed out", map[string]any{"error_class": string(providers.ErrorClassTimeout)})
		}
		return o.fail(runContext, iteration, StopReasonTimedOut, "run timed out", err)
	case providers.ErrorClassOf(err) == string(providers.ErrorClassCanceled):
		if runContext != nil {
			runContext.EmitEvent("run.canceled", iteration, "run canceled", map[string]any{"error_class": string(providers.ErrorClassCanceled)})
		}
		return o.fail(runContext, iteration, StopReasonCanceled, "run canceled", err)
	case providers.ErrorClassOf(err) == string(providers.ErrorClassTimeout):
		if runContext != nil {
			runContext.EmitEvent("run.timed_out", iteration, "run timed out", map[string]any{"error_class": string(providers.ErrorClassTimeout)})
		}
		return o.fail(runContext, iteration, StopReasonTimedOut, "run timed out", err)
	default:
		return o.fail(runContext, iteration, StopReasonProviderError, "provider error", err)
	}
}

func (o Orchestrator) failWithContext(runContext *RunContext, iteration int, err error) (RunResult, error) {
	switch {
	case errors.Is(err, context.Canceled):
		runContext.EmitEvent("run.canceled", iteration, "run canceled", map[string]any{"error_class": string(providers.ErrorClassCanceled)})
		return o.fail(runContext, iteration, StopReasonCanceled, "run canceled", err), err
	case errors.Is(err, context.DeadlineExceeded):
		runContext.EmitEvent("run.timed_out", iteration, "run timed out", map[string]any{"error_class": string(providers.ErrorClassTimeout)})
		return o.fail(runContext, iteration, StopReasonTimedOut, "run timed out", err), err
	default:
		return o.fail(runContext, iteration, StopReasonProviderError, "run failed", err), err
	}
}
