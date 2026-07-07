package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	apperrors "github.com/MehulCodr/AI-agent/internal/errors"
	"github.com/MehulCodr/AI-agent/internal/llm"
	"github.com/MehulCodr/AI-agent/internal/tools"
)

func TestAgentReturnsMockLLMResponse(t *testing.T) {
	provider := &fakeProvider{
		response: llm.Message{Role: "assistant", Content: "hi there"},
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
		response: llm.Message{Role: "assistant", Content: "hi there"},
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
	if !errors.Is(err, apperrors.ErrInvalidInput) || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error = %v, want ErrInvalidInput empty input error", err)
	}
}

func TestAgentRequiresProvider(t *testing.T) {
	agent := New(nil)

	_, err := agent.Run(context.Background(), "hello")
	if !errors.Is(err, apperrors.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestAgentClearRemovesHistory(t *testing.T) {
	agent := New(&fakeProvider{
		response: llm.Message{Role: "assistant", Content: "hi there"},
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
	want := []string{"current_directory", "echo", "edit_file", "list_files", "read_file", "run_shell", "write_file"}
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

func TestAgentExecutesToolCallsAndReturnsFinalResponse(t *testing.T) {
	chdirAgentTest(t, t.TempDir())
	if err := os.WriteFile("README.md", []byte("hello from readme"), 0644); err != nil {
		t.Fatalf("write README: %v", err)
	}

	provider := &fakeProvider{
		responses: []llm.Message{
			{
				Role: "assistant",
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: llm.ToolCallFunction{
							Name:      "read_file",
							Arguments: `{"path":"README.md"}`,
						},
					},
				},
			},
			{Role: "assistant", Content: "README says hello from readme"},
		},
	}
	agent := New(provider)

	got, err := agent.Run(context.Background(), "read README")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "README says hello from readme" {
		t.Fatalf("Run returned %q, want final response", got)
	}
	if provider.calls != 2 {
		t.Fatalf("provider calls = %d, want 2", provider.calls)
	}

	messages := agent.Messages()
	if len(messages) != 3 {
		t.Fatalf("len(Messages()) = %d, want user, tool result, assistant", len(messages))
	}
	if !strings.Contains(messages[1].Content, "hello from readme") {
		t.Fatalf("tool result = %q, want file content", messages[1].Content)
	}
}

func TestAgentParsesJSONToolCallsFromAssistantContent(t *testing.T) {
	chdirAgentTest(t, t.TempDir())
	if err := os.WriteFile("notes.txt", []byte("notes"), 0644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	provider := &fakeProvider{
		responses: []llm.Message{
			{Role: "assistant", Content: `{"tool_calls":[{"function":{"name":"list_files","arguments":{"path":"."}}}]}`},
			{Role: "assistant", Content: "I found notes.txt"},
		},
	}
	agent := New(provider)

	got, err := agent.Run(context.Background(), "what files are here?")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "I found notes.txt" {
		t.Fatalf("Run returned %q, want final response", got)
	}
	if !strings.Contains(agent.Messages()[1].Content, "notes.txt") {
		t.Fatalf("tool result = %q, want listed file", agent.Messages()[1].Content)
	}
}

func TestAgentFallsBackWhenProviderRepeatsToolCall(t *testing.T) {
	chdirAgentTest(t, t.TempDir())
	if err := os.WriteFile("notes.txt", []byte("notes"), 0644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	agent := New(&fakeProvider{
		response: llm.Message{
			Role: "assistant",
			ToolCalls: []llm.ToolCall{
				{
					Function: llm.ToolCallFunction{
						Name:      "list_files",
						Arguments: `{"path":"."}`,
					},
				},
			},
		},
	})

	got, err := agent.Run(context.Background(), "list files")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(got, "notes.txt") {
		t.Fatalf("Run returned %q, want latest tool result", got)
	}
	if !strings.Contains(got, "model kept requesting tools") {
		t.Fatalf("Run returned %q, want fallback explanation", got)
	}
}

func TestAgentPassesProjectContextToProvider(t *testing.T) {
	chdirAgentTest(t, t.TempDir())
	if err := os.WriteFile("README.md", []byte("# test"), 0644); err != nil {
		t.Fatalf("write README: %v", err)
	}

	provider := &fakeProvider{
		response: llm.Message{Role: "assistant", Content: "ok"},
	}
	agent := New(provider)

	if _, err := agent.Run(context.Background(), "summarize"); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(provider.seen) == 0 {
		t.Fatal("provider saw no messages")
	}
	if provider.seen[0].Role != "system" {
		t.Fatalf("first message role = %q, want system", provider.seen[0].Role)
	}
	if !strings.Contains(provider.seen[0].Content, "Project context:") {
		t.Fatalf("system prompt = %q, want project context", provider.seen[0].Content)
	}
	if !strings.Contains(provider.seen[0].Content, "README.md") {
		t.Fatalf("system prompt = %q, want README in tree", provider.seen[0].Content)
	}
}

type fakeProvider struct {
	response  llm.Message
	responses []llm.Message
	err       error
	calls     int
	seen      []llm.Message
}

func (p *fakeProvider) Chat(ctx context.Context, messages []llm.Message) (llm.Message, error) {
	if err := ctx.Err(); err != nil {
		return llm.Message{}, err
	}
	p.calls++
	p.seen = append([]llm.Message(nil), messages...)
	if p.err != nil {
		return llm.Message{}, p.err
	}
	if len(p.responses) > 0 {
		response := p.responses[0]
		p.responses = p.responses[1:]
		return response, nil
	}
	return p.response, nil
}

func chdirAgentTest(t *testing.T, dir string) {
	t.Helper()

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
}

func TestAgentCanEditFileThroughToolCall(t *testing.T) {
	root := t.TempDir()
	chdirAgentTest(t, root)
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("hello old world"), 0644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	provider := &fakeProvider{
		responses: []llm.Message{
			{
				Role: "assistant",
				ToolCalls: []llm.ToolCall{
					{
						Function: llm.ToolCallFunction{
							Name:      "edit_file",
							Arguments: `{"path":"notes.txt","old":"old","new":"new","apply":true}`,
						},
					},
				},
			},
			{Role: "assistant", Content: "Updated notes.txt"},
		},
	}
	agent := New(provider)

	got, err := agent.Run(context.Background(), "change old to new in notes.txt")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "Updated notes.txt" {
		t.Fatalf("Run returned %q, want update response", got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read notes: %v", err)
	}
	if string(data) != "hello new world" {
		t.Fatalf("content = %q, want edited file", string(data))
	}
}
