package tools

import (
	"context"
	"strings"
	"testing"
)

func TestShellToolRequiresApprovalHandler(t *testing.T) {
	_, err := ShellTool{}.Execute(context.Background(), map[string]any{"command": "echo hello"})
	if err == nil || !strings.Contains(err.Error(), "approval") {
		t.Fatalf("error = %v, want approval error", err)
	}
}

func TestShellToolDeniesCommand(t *testing.T) {
	tool := ShellTool{Approve: func(ctx context.Context, command string) (bool, error) {
		return false, ctx.Err()
	}}

	got, err := tool.Execute(context.Background(), map[string]any{"command": "echo hello"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got != "shell command denied by user" {
		t.Fatalf("result = %q, want denial", got)
	}
}
