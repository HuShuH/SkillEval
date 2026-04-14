package adapters

import (
	"context"
	"fmt"

	"agent-skill-eval-go/internal/spec"
)

// MockAdapter provides deterministic in-memory behavior for MVP evaluation.
type MockAdapter struct{}

// Run executes a minimal mock skill behavior without external dependencies.
func (m MockAdapter) Run(ctx context.Context, tc spec.TestCase, skill spec.SkillSpec) (spec.AgentOutput, error) {
	select {
	case <-ctx.Done():
		return spec.AgentOutput{}, ctx.Err()
	default:
	}

	switch skill.Name {
	case "hello_world":
		return spec.AgentOutput{
			FinalOutput: "hello world",
		}, nil
	case "echo":
		return spec.AgentOutput{
			FinalOutput: tc.Prompt,
		}, nil
	case "mock_tool_call":
		return spec.AgentOutput{
			ToolCalls: []spec.ToolCall{
				{
					ToolName: "mock_tool",
					Args: map[string]interface{}{
						"value": "ok",
					},
				},
			},
		}, nil
	default:
		return spec.AgentOutput{}, fmt.Errorf("unknown mock skill: %s", skill.Name)
	}
}
