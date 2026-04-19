// Package eval contains Phase 1 migration skeleton types for the new architecture.
// It currently provides minimal runnable case definitions, validation, and runner types.
package eval

import (
	"context"
	"fmt"

	"agent-skill-eval-go/agent"
	"agent-skill-eval-go/skill"
)

// RunConfig defines minimal runner settings.
type RunConfig struct {
	WorkspaceRoot string    `json:"workspace_root,omitempty"`
	EventSink     EventSink `json:"-"`
}

// EventSink receives per-case runtime events from the eval runner.
type EventSink func(caseID string, side string, event agent.Event)

// ScoreResult is a placeholder for future pair scoring or judge outputs.
type ScoreResult struct {
	Scored bool   `json:"scored"`
	Winner string `json:"winner,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// CaseResult is the normalized result for one case run.
type CaseResult struct {
	CaseID          string                      `json:"case_id"`
	AgentName       string                      `json:"agent_name,omitempty"`
	StopReason      string                      `json:"stop_reason,omitempty"`
	FinalAnswer     string                      `json:"final_answer,omitempty"`
	Passed          bool                        `json:"passed"`
	ToolExecutions  []agent.ToolExecutionRecord `json:"tool_executions,omitempty"`
	Iterations      int                         `json:"iterations"`
	Events          []agent.Event               `json:"events,omitempty"`
	Check           CheckResult                 `json:"check"`
	Error           string                      `json:"error,omitempty"`
	ErrorClass      string                      `json:"error_class,omitempty"`
	FailedIteration int                         `json:"failed_iteration,omitempty"`
}

// SingleRunResult wraps one case-level run result.
type SingleRunResult struct {
	CaseResult CaseResult `json:"case_result"`
}

// PairResult holds sequential A/B outputs for the same case.
type PairResult struct {
	CaseID string          `json:"case_id"`
	A      SingleRunResult `json:"a"`
	B      SingleRunResult `json:"b"`
	Score  ScoreResult     `json:"score"`
	Error  string          `json:"error,omitempty"`
}

// Runner executes single and pair runs through the orchestrator.
type Runner struct {
	Config       RunConfig
	Orchestrator agent.Orchestrator
}

// RunCases executes multiple cases sequentially and builds a run report.
func (r Runner) RunCases(ctx context.Context, a agent.Agent, s skill.Skill, cases []Case) (RunReport, error) {
	results := make([]CaseResult, 0, len(cases))
	var firstErr error

	for index, c := range cases {
		runResult, err := r.RunCase(ctx, fmt.Sprintf("%s/%d", c.ID, index), a, s, c)
		results = append(results, runResult.CaseResult)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	report := BuildRunReport(results)
	return report, firstErr
}

// RunCase executes one case for one agent/skill pair.
func (r Runner) RunCase(ctx context.Context, runID string, a agent.Agent, s skill.Skill, c Case) (SingleRunResult, error) {
	return r.runCaseWithSide(ctx, runID, a, s, c, "single")
}

func (r Runner) runCaseWithSide(ctx context.Context, runID string, a agent.Agent, s skill.Skill, c Case, side string) (SingleRunResult, error) {
	if err := c.Validate(); err != nil {
		return SingleRunResult{
			CaseResult: CaseResult{
				CaseID:    c.ID,
				AgentName: a.Name,
				Passed:    false,
				Error:     err.Error(),
			},
		}, err
	}

	runContext := &agent.RunContext{
		RunID:     runID,
		Workspace: r.Config.WorkspaceRoot,
	}
	if r.Config.EventSink != nil {
		runContext.EventSink = func(event agent.Event) {
			r.Config.EventSink(c.ID, side, event)
		}
	}

	prompt := c.Prompt
	if prompt == "" {
		prompt = c.Task
	}

	runResult, err := r.Orchestrator.Run(ctx, a, s, runContext, prompt)
	checkResult := EvaluateChecks(c, runResult.FinalAnswer)

	caseResult := CaseResult{
		CaseID:          c.ID,
		AgentName:       a.Name,
		StopReason:      string(runResult.StopReason),
		FinalAnswer:     runResult.FinalAnswer,
		Passed:          checkResult.Passed && err == nil,
		ToolExecutions:  append([]agent.ToolExecutionRecord(nil), runResult.ToolExecutions...),
		Iterations:      runResult.Iterations,
		Events:          append([]agent.Event(nil), runContext.Events...),
		Check:           checkResult,
		ErrorClass:      runResult.ErrorClass,
		FailedIteration: runResult.FailedIteration,
	}
	if err != nil {
		caseResult.Error = err.Error()
		caseResult.Passed = false
	}

	return SingleRunResult{CaseResult: caseResult}, err
}

// RunPair executes the same case twice, once with skill A and once with skill B.
func (r Runner) RunPair(ctx context.Context, c Case, agentA agent.Agent, skillA skill.Skill, agentB agent.Agent, skillB skill.Skill) (PairResult, error) {
	resultA, errA := r.runCaseWithSide(ctx, fmt.Sprintf("%s/a", c.ID), agentA, skillA, c, "a")
	resultB, errB := r.runCaseWithSide(ctx, fmt.Sprintf("%s/b", c.ID), agentB, skillB, c, "b")

	pair := PairResult{
		CaseID: c.ID,
		A:      resultA,
		B:      resultB,
		Score: ScoreResult{
			Scored: false,
			Reason: "not_scored",
		},
	}

	switch {
	case errA != nil && errB != nil:
		pair.Error = fmt.Sprintf("run A: %v; run B: %v", errA, errB)
		return pair, errA
	case errA != nil:
		pair.Error = fmt.Sprintf("run A: %v", errA)
		return pair, errA
	case errB != nil:
		pair.Error = fmt.Sprintf("run B: %v", errB)
		return pair, errB
	default:
		return pair, nil
	}
}

// RunCasePairs executes multiple pairs sequentially and builds a pair report.
func (r Runner) RunCasePairs(ctx context.Context, cases []Case, agentA agent.Agent, skillA skill.Skill, agentB agent.Agent, skillB skill.Skill) (PairReport, error) {
	results := make([]PairResult, 0, len(cases))
	var firstErr error

	for _, c := range cases {
		pairResult, err := r.RunPair(ctx, c, agentA, skillA, agentB, skillB)
		results = append(results, pairResult)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	report := BuildPairReport(results)
	return report, firstErr
}
