// Package agent contains Phase 1 migration skeleton types for the new architecture.
// It currently provides only minimal static agent configuration and no orchestrator.
package agent

import (
	"agent-skill-eval-go/providers"
	"agent-skill-eval-go/tool"
)

// Agent is the static configuration for one reusable agent instance.
type Agent struct {
	Name           string
	Provider       providers.ChatClient
	ProviderConfig providers.Config
	Tools          []tool.Tool
	MaxIterations  int
	SystemPrompt   string
	Instructions   string
}
