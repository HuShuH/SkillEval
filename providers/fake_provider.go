// Package providers contains Phase 1 migration skeleton types for the new architecture.
// It currently provides only minimal provider contracts, response abstractions, and fake behavior.
package providers

import (
	"context"
	"fmt"
)

// FakeStep represents one deterministic provider step for tests.
type FakeStep struct {
	Response ChatResponse
	Err      error
}

// SequenceClient is a deterministic in-memory provider used by orchestrator tests.
type SequenceClient struct {
	Steps    []FakeStep
	Requests []ChatRequest
	index    int
}

func (c *SequenceClient) ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	c.Requests = append(c.Requests, req)
	if c.index >= len(c.Steps) {
		return ChatResponse{}, fmt.Errorf("fake provider exhausted at step %d", c.index)
	}
	step := c.Steps[c.index]
	c.index++
	if step.Err != nil {
		return ChatResponse{}, step.Err
	}
	return step.Response, nil
}
