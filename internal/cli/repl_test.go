package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
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
}

type fakeChatRunner struct {
	response  string
	lastInput string
	cleared   bool
}

func (r *fakeChatRunner) Run(ctx context.Context, input string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	r.lastInput = input
	return r.response, nil
}

func (r *fakeChatRunner) Clear() {
	r.cleared = true
}
