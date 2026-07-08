package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

const defaultShellTimeout = 30 * time.Second
const maxShellOutputBytes = 64 * 1024

type ApprovalFunc func(ctx context.Context, command string) (bool, error)

type ShellTool struct {
	Approve ApprovalFunc
}

func (ShellTool) Name() string {
	return "shell"
}

func (ShellTool) Description() string {
	return "Runs a shell command after explicit user approval."
}

func (ShellTool) Parameters() map[string]any {
	return objectSchema([]string{"command"}, map[string]any{
		"command": stringProperty("Shell command to run."),
		"cwd":     stringProperty("Optional relative working directory inside the project root."),
		"timeout_seconds": map[string]any{
			"type":        "number",
			"description": "Optional timeout in seconds. Defaults to 30.",
		},
	})
}

func (t ShellTool) Execute(ctx context.Context, input map[string]any) (string, error) {
	if err := contextError(ctx); err != nil {
		return "", err
	}
	if t.Approve == nil {
		return "", fmt.Errorf("shell command requires an approval handler")
	}

	command, err := requiredString(input, "command", "shell")
	if err != nil {
		return "", err
	}

	approved, err := t.Approve(ctx, command)
	if err != nil {
		return "", err
	}
	if !approved {
		return "shell command denied by user", nil
	}

	timeout := defaultShellTimeout
	if value, ok := input["timeout_seconds"]; ok {
		switch typed := value.(type) {
		case float64:
			if typed > 0 {
				timeout = time.Duration(typed) * time.Second
			}
		case int:
			if typed > 0 {
				timeout = time.Duration(typed) * time.Second
			}
		default:
			return "", fmt.Errorf("shell tool timeout_seconds must be a number")
		}
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := platformShellCommand(runCtx, command)
	if value, ok := input["cwd"]; ok {
		cwd, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("shell tool cwd must be a string")
		}
		if cwd != "" {
			safeCWD, err := safeProjectPath(cwd)
			if err != nil {
				return "", err
			}
			if err := ensureExistingTargetInsideProject(safeCWD); err != nil {
				return "", err
			}
			cmd.Dir = safeCWD
		}
	}

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err = cmd.Run()
	text := truncateShellOutput(output.String())
	if runCtx.Err() == context.DeadlineExceeded {
		return text, fmt.Errorf("shell command timed out after %s", timeout)
	}
	if err != nil {
		return text, fmt.Errorf("shell command failed: %w", err)
	}
	if text == "" {
		return "shell command completed with no output", nil
	}
	return text, nil
}

func platformShellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	}
	return exec.CommandContext(ctx, "sh", "-c", command)
}

func truncateShellOutput(output string) string {
	if len(output) <= maxShellOutputBytes {
		return output
	}
	return output[:maxShellOutputBytes] + "\n[output truncated]"
}
