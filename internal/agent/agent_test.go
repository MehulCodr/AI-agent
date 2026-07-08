package agent

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/MehulCodr/AI-agent/internal/llm"
	"github.com/MehulCodr/AI-agent/internal/rag"
	"github.com/MehulCodr/AI-agent/internal/tools"
)

func TestAgentReturnsLLMResponse(t *testing.T) {
	provider := &fakeProvider{
		responses: []llm.Message{{Role: "assistant", Content: "hi there"}},
	}
	agent := New(provider)

	got, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "hi there" {
		t.Fatalf("Run returned %q, want %q", got, "hi there")
	}
}

func TestAgentStoresUserAndAssistantMessages(t *testing.T) {
	provider := &fakeProvider{
		responses: []llm.Message{{Role: "assistant", Content: "hi there"}},
	}
	agent := New(provider)

	if _, err := agent.Run(context.Background(), "hello"); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	messages := agent.Messages()
	if len(messages) != 2 {
		t.Fatalf("len(Messages()) = %d, want 2", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "hello" {
		t.Fatalf("user message = %#v", messages[0])
	}
	if messages[1].Role != "assistant" || messages[1].Content != "hi there" {
		t.Fatalf("assistant message = %#v", messages[1])
	}

	messages[0].Content = "changed"
	if agent.Messages()[0].Content != "hello" {
		t.Fatalf("Messages returned internal slice directly")
	}
}

func TestAgentRejectsEmptyInput(t *testing.T) {
	agent := New(&fakeProvider{})

	_, err := agent.Run(context.Background(), "   ")
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error = %v, want empty input error", err)
	}
}

func TestAgentClearRemovesHistory(t *testing.T) {
	agent := New(&fakeProvider{
		responses: []llm.Message{{Role: "assistant", Content: "hi there"}},
	})

	if _, err := agent.Run(context.Background(), "hello"); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	agent.Clear()

	if len(agent.Messages()) != 0 {
		t.Fatalf("len(Messages()) = %d, want 0", len(agent.Messages()))
	}
}

func TestAgentHasDefaultSafeTools(t *testing.T) {
	agent := New(&fakeProvider{})

	got := agent.Tools()
	want := []string{"current_directory", "echo", "edit_file", "list_files", "read_file", "write_file"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Tools() = %#v, want %#v", got, want)
	}
}

func TestAgentAddsShellAndRepoToolsWhenConfigured(t *testing.T) {
	agent := New(
		&fakeProvider{},
		WithApproval(func(ctx context.Context, command string) (bool, error) {
			return true, ctx.Err()
		}),
		WithRepoSearcher(fakeSearcher{}),
	)

	got := agent.Tools()
	want := []string{"current_directory", "echo", "edit_file", "list_files", "read_file", "search_repo", "shell", "write_file"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Tools() = %#v, want %#v", got, want)
	}
}

func TestAgentCanReplaceTools(t *testing.T) {
	registry := tools.NewRegistry()
	if err := registry.Register(tools.EchoTool{}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	agent := New(&fakeProvider{})
	agent.SetTools(registry)

	got := agent.Tools()
	want := []string{"echo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Tools() = %#v, want %#v", got, want)
	}
}

func TestAgentRespectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	agent := New(&fakeProvider{})

	_, err := agent.Run(ctx, "hello")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if len(agent.Messages()) != 0 {
		t.Fatalf("len(Messages()) = %d, want 0", len(agent.Messages()))
	}
}

func TestAgentExecutesToolCalls(t *testing.T) {
	provider := &fakeProvider{
		responses: []llm.Message{
			{
				Role: "assistant",
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: llm.ToolCallFunction{
							Name:      "echo",
							Arguments: `{"text":"tool result"}`,
						},
					},
				},
			},
			{Role: "assistant", Content: "done"},
		},
	}
	agent := New(provider)

	got, err := agent.Run(context.Background(), "use a tool")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("Run returned %q, want done", got)
	}

	messages := agent.Messages()
	if len(messages) != 4 {
		t.Fatalf("len(Messages()) = %d, want 4", len(messages))
	}
	if messages[2].Role != llm.RoleTool || messages[2].Content != "tool result" {
		t.Fatalf("tool message = %#v", messages[2])
	}
}

type fakeProvider struct {
	responses []llm.Message
	err       error
	calls     int
	seen      []llm.Message
}

func (p *fakeProvider) Name() string {
	return "fake"
}

func (p *fakeProvider) Chat(ctx context.Context, request llm.ChatRequest) (llm.Message, error) {
	if err := ctx.Err(); err != nil {
		return llm.Message{}, err
	}
	p.calls++
	p.seen = append([]llm.Message(nil), request.Messages...)
	if p.err != nil {
		return llm.Message{}, p.err
	}
	if len(p.responses) == 0 {
		return llm.Message{Role: llm.RoleAssistant, Content: ""}, nil
	}
	response := p.responses[0]
	p.responses = p.responses[1:]
	return response, nil
}

func (p *fakeProvider) Stream(ctx context.Context, request llm.ChatRequest, onEvent llm.StreamHandler) (llm.Message, error) {
	message, err := p.Chat(ctx, request)
	if err != nil {
		return llm.Message{}, err
	}
	if onEvent != nil && message.Content != "" {
		if err := onEvent(llm.StreamEvent{Delta: message.Content}); err != nil {
			return llm.Message{}, err
		}
	}
	return message, nil
}

type fakeSearcher struct{}

func (fakeSearcher) Search(ctx context.Context, query string, limit int) ([]rag.SearchResult, error) {
	_, _, _ = ctx, query, limit
	return nil, nil
}
