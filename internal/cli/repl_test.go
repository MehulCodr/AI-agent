package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/MehulCodr/AI-agent/internal/llm"
)

func TestStartREPLRunsChatRunner(t *testing.T) {
	runner := &fakeChatRunner{response: "hello back"}
	input := strings.NewReader("hello\n/exit\n")
	var output bytes.Buffer

	if err := StartREPL(context.Background(), input, &output, runner); err != nil {
		t.Fatalf("StartREPL returned error: %v", err)
	}
	if runner.lastInput != "hello" {
		t.Fatalf("last input = %q, want hello", runner.lastInput)
	}
	if !strings.Contains(output.String(), "Agent: hello back") {
		t.Fatalf("output = %q, want agent response", output.String())
	}
}

func TestStartREPLClearsChatRunner(t *testing.T) {
	runner := &fakeChatRunner{}
	input := strings.NewReader("/clear\n/exit\n")
	var output bytes.Buffer

	if err := StartREPL(context.Background(), input, &output, runner); err != nil {
		t.Fatalf("StartREPL returned error: %v", err)
	}
	if !runner.cleared {
		t.Fatal("runner was not cleared")
	}
	if !runner.saved {
		t.Fatal("runner session was not saved")
	}
}

func TestStartREPLPrintsHistory(t *testing.T) {
	runner := &fakeChatRunner{
		messages: []llm.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
		},
	}
	input := strings.NewReader("/history\n/exit\n")
	var output bytes.Buffer

	if err := StartREPL(context.Background(), input, &output, runner); err != nil {
		t.Fatalf("StartREPL returned error: %v", err)
	}
	if !strings.Contains(output.String(), "user: hello") || !strings.Contains(output.String(), "assistant: hi") {
		t.Fatalf("output = %q, want history", output.String())
	}
}

type fakeChatRunner struct {
	response  string
	lastInput string
	cleared   bool
	saved     bool
	messages  []llm.Message
}

func (r *fakeChatRunner) Run(ctx context.Context, input string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	r.lastInput = input
	r.messages = append(r.messages, llm.Message{Role: "user", Content: input})
	r.messages = append(r.messages, llm.Message{Role: "assistant", Content: r.response})
	return r.response, nil
}

func (r *fakeChatRunner) Clear() {
	r.cleared = true
	r.messages = nil
}

func (r *fakeChatRunner) Messages() []llm.Message {
	messages := make([]llm.Message, len(r.messages))
	copy(messages, r.messages)
	return messages
}

func (r *fakeChatRunner) SaveSession() error {
	r.saved = true
	return nil
}
