package llm

import (
	"context"
	"errors"
	"testing"
)

func TestMockProviderReturnsConfiguredResponse(t *testing.T) {
	provider := MockProvider{Response: "configured"}

	got, err := provider.Chat(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "hello"}}})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if got.Role != "assistant" || got.Content != "configured" {
		t.Fatalf("message = %#v, want configured assistant response", got)
	}
}

func TestMockProviderEchoesLastUserMessage(t *testing.T) {
	provider := MockProvider{}

	got, err := provider.Chat(context.Background(), ChatRequest{Messages: []Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "ok"},
		{Role: "user", Content: "second"},
	}})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if got.Content != "mock response: second" {
		t.Fatalf("content = %q, want last user message", got.Content)
	}
}

func TestMockProviderRespectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := MockProvider{}.Chat(ctx, ChatRequest{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
