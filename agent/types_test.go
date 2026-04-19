package agent

import (
	"testing"

	"agent-skill-eval-go/providers"
	"agent-skill-eval-go/tool"
)

func TestAgentConstructible(t *testing.T) {
	a := Agent{
		Name:           "agent-a",
		ProviderConfig: providers.Config{Model: "gpt-test"},
		Tools: []tool.Tool{
			tool.FinishTool{},
		},
		MaxIterations: 8,
		SystemPrompt:  "system",
		Instructions:  "follow the task",
	}

	if a.Name != "agent-a" {
		t.Fatalf("unexpected agent name: %q", a.Name)
	}
	if a.MaxIterations != 8 {
		t.Fatalf("unexpected max iterations: %d", a.MaxIterations)
	}
	if len(a.Tools) != 1 {
		t.Fatalf("unexpected tool count: %d", len(a.Tools))
	}
}
