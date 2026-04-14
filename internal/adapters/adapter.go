package adapters

import (
	"context"

	"agent-skill-eval-go/internal/spec"
)

// Adapter defines the minimal execution contract for the MVP.
type Adapter interface {
	Run(ctx context.Context, tc spec.TestCase, skill spec.SkillSpec) (spec.AgentOutput, error)
}
