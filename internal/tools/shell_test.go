package tools

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestShellToolRunsSafeCommand(t *testing.T) {
	got, err := ShellTool{}.Execute(context.Background(), map[string]any{
		"command": os.Args[0],
		"args":    []string{"-test.run=TestShellHelperProcess", "--", "stdout"},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(got, "hello stdout") {
		t.Fatalf("output = %q, want stdout", got)
	}
}

func TestShellToolBlocksDangerousCommand(t *testing.T) {
	_, err := ShellTool{}.Execute(context.Background(), map[string]any{
		"command": "rm",
		"args":    []string{"-rf", "/"},
	})
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("error = %v, want blocked command error", err)
	}
}

func TestShellToolTimeout(t *testing.T) {
	_, err := ShellTool{Timeout: 10 * time.Millisecond}.Execute(context.Background(), map[string]any{
		"command": os.Args[0],
		"args":    []string{"-test.run=TestShellHelperProcess", "--", "sleep"},
	})
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("error = %v, want timeout error", err)
	}
}

func TestShellToolRespectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ShellTool{}.Execute(ctx, map[string]any{
		"command": os.Args[0],
		"args":    []string{"-test.run=TestShellHelperProcess", "--", "stdout"},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestShellToolCapturesStderr(t *testing.T) {
	got, err := ShellTool{}.Execute(context.Background(), map[string]any{
		"command": os.Args[0],
		"args":    []string{"-test.run=TestShellHelperProcess", "--", "stderr"},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(got, "hello stderr") {
		t.Fatalf("output = %q, want stderr", got)
	}
}

func TestShellHelperProcess(t *testing.T) {
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	if len(args) < 2 {
		return
	}

	switch args[1] {
	case "stdout":
		_, _ = os.Stdout.WriteString("hello stdout\n")
	case "stderr":
		_, _ = os.Stderr.WriteString("hello stderr\n")
	case "sleep":
		time.Sleep(time.Second)
	}

	os.Exit(0)
}
