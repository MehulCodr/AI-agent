package llm

import (
	"context"
	"errors"
	"testing"
)

func TestMockProviderReturnsConfiguredResponse(t *testing.T) {
	provider := MockProvider{Response: "configured"}

	got, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if got.Role != "assistant" || got.Content != "configured" {
		t.Fatalf("message = %#v, want configured assistant response", got)
	}
}

func TestMockProviderEchoesLastUserMessage(t *testing.T) {
	provider := MockProvider{}

	got, err := provider.Chat(context.Background(), []Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "ok"},
		{Role: "user", Content: "second"},
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if got.Content != "mock response: second" {
		t.Fatalf("content = %q, want last user message", got.Content)
	}
}

func TestMockProviderCallsListFilesForCommonFilePrompt(t *testing.T) {
	provider := MockProvider{}

	got, err := provider.Chat(context.Background(), []Message{
		{Role: "user", Content: "list current files in this directory"},
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if len(got.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(got.ToolCalls))
	}
	if got.ToolCalls[0].Function.Name != "list_files" {
		t.Fatalf("tool name = %q, want list_files", got.ToolCalls[0].Function.Name)
	}
}

func TestMockProviderReturnsToolResults(t *testing.T) {
	provider := MockProvider{}

	got, err := provider.Chat(context.Background(), []Message{
		{Role: "user", Content: "Tool results:\n\nlist_files:\nREADME.md\n"},
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if got.Content != "mock response:\nTool results:\n\nlist_files:\nREADME.md" {
		t.Fatalf("content = %q, want tool result response", got.Content)
	}
}

func TestMockProviderRespectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := MockProvider{}.Chat(ctx, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
