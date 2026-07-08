package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/MehulCodr/AI-agent/internal/agent"
)

func TestStartREPLRunsChatRunner(t *testing.T) {
	runner := &fakeChatRunner{response: "hello back"}
	input := strings.NewReader("hello\n/exit\n")
	var output bytes.Buffer

	if err := StartREPL(context.Background(), input, &output, runner, REPLConfig{}); err != nil {
		t.Fatalf("StartREPL returned error: %v", err)
	}
	if runner.lastInput != "hello" {
		t.Fatalf("last input = %q, want hello", runner.lastInput)
	}
	if !strings.Contains(output.String(), "Agent: hello back") {
		t.Fatalf("output = %q, want agent response", output.String())
	}
}

func TestStartREPLStreamsWhenEnabled(t *testing.T) {
	runner := &fakeChatRunner{response: "hello back"}
	input := strings.NewReader("hello\n/exit\n")
	var output bytes.Buffer

	if err := StartREPL(context.Background(), input, &output, runner, REPLConfig{Stream: true}); err != nil {
		t.Fatalf("StartREPL returned error: %v", err)
	}
	if !runner.streamed {
		t.Fatal("runner did not stream")
	}
	if !strings.Contains(output.String(), "Agent: hello back") {
		t.Fatalf("output = %q, want streamed response", output.String())
	}
}

func TestStartREPLClearsChatRunner(t *testing.T) {
	runner := &fakeChatRunner{}
	input := strings.NewReader("/clear\n/exit\n")
	var output bytes.Buffer

	if err := StartREPL(context.Background(), input, &output, runner, REPLConfig{}); err != nil {
		t.Fatalf("StartREPL returned error: %v", err)
	}
	if !runner.cleared {
		t.Fatal("runner was not cleared")
	}
}

type fakeChatRunner struct {
	response  string
	lastInput string
	cleared   bool
	streamed  bool
}

func (r *fakeChatRunner) Run(ctx context.Context, input string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	r.lastInput = input
	return r.response, nil
}

func (r *fakeChatRunner) RunStream(ctx context.Context, input string, onEvent agent.EventHandler) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	r.lastInput = input
	r.streamed = true
	if onEvent != nil {
		if err := onEvent(agent.Event{Type: "delta", Content: r.response}); err != nil {
			return "", err
		}
	}
	return r.response, nil
}

func (r *fakeChatRunner) Clear() {
	r.cleared = true
}
