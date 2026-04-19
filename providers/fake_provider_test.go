package providers

import (
	"context"
	"testing"
)

func TestSequenceClientReturnsStepsInOrder(t *testing.T) {
	client := &SequenceClient{
		Steps: []FakeStep{
			{Response: ChatResponse{Message: Message{Role: "assistant", Content: "one"}}},
			{Response: ChatResponse{Finish: &FinishSignal{FinalAnswer: "done"}}},
		},
	}

	first, err := client.ChatCompletion(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected first error: %v", err)
	}
	second, err := client.ChatCompletion(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected second error: %v", err)
	}
	if first.Message.Content != "one" {
		t.Fatalf("unexpected first content: %q", first.Message.Content)
	}
	if second.Finish == nil || second.Finish.FinalAnswer != "done" {
		t.Fatalf("unexpected second finish: %+v", second.Finish)
	}
}
