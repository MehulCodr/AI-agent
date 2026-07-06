package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const defaultShellTimeout = 30 * time.Second

type ShellTool struct {
	Timeout time.Duration
}

func (ShellTool) Name() string {
	return "run_shell"
}

func (ShellTool) Description() string {
	return "Runs a safe shell command with a timeout."
}

func (t ShellTool) Execute(ctx context.Context, input map[string]any) (string, error) {
	if err := contextError(ctx); err != nil {
		return "", err
	}

	command, err := requiredString(input, "command", "run_shell")
	if err != nil {
		return "", err
	}
	args, err := optionalStringSlice(input, "args", "run_shell")
	if err != nil {
		return "", err
	}
	if err := validateShellCommand(command, args); err != nil {
		return "", err
	}

	timeout := t.Timeout
	if timeout <= 0 {
		timeout = defaultShellTimeout
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, command, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err = cmd.Run()
	result := output.String()
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		return result, fmt.Errorf("run_shell command timed out after %s", timeout)
	}
	if errors.Is(runCtx.Err(), context.Canceled) {
		return result, runCtx.Err()
	}
	if err != nil {
		return result, fmt.Errorf("run_shell command failed: %w", err)
	}

	return result, nil
}

func optionalStringSlice(input map[string]any, key, toolName string) ([]string, error) {
	value, ok := input[key]
	if !ok || value == nil {
		return nil, nil
	}

	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), nil
	case []any:
		args := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%s tool %s must contain only strings", toolName, key)
			}
			args = append(args, text)
		}
		return args, nil
	default:
		return nil, fmt.Errorf("%s tool %s must be a string array", toolName, key)
	}
}

func validateShellCommand(command string, args []string) error {
	if containsDangerousShellPhrase(command, args) {
		return blockedShellCommandError(command)
	}

	name := normalizedCommandName(command)
	if name == "" {
		return fmt.Errorf("run_shell command cannot be empty")
	}

	switch {
	case name == "sudo":
		return blockedShellCommandError(command)
	case name == "shutdown":
		return blockedShellCommandError(command)
	case name == "reboot":
		return blockedShellCommandError(command)
	case name == "dd":
		return blockedShellCommandError(command)
	case strings.HasPrefix(name, "mkfs"):
		return blockedShellCommandError(command)
	case isDangerousRm(name, args):
		return blockedShellCommandError(command)
	case isDangerousChmod(name, args):
		return blockedShellCommandError(command)
	case isDangerousChown(name, args):
		return blockedShellCommandError(command)
	}

	return nil
}

func containsDangerousShellPhrase(command string, args []string) bool {
	parts := append([]string{command}, args...)
	fields := strings.Fields(strings.Join(parts, " "))
	if len(fields) == 0 {
		return false
	}

	name := normalizedCommandName(fields[0])
	commandArgs := fields[1:]

	return name == "sudo" ||
		name == "shutdown" ||
		name == "reboot" ||
		name == "dd" ||
		strings.HasPrefix(name, "mkfs") ||
		isDangerousRm(name, commandArgs) ||
		isDangerousChmod(name, commandArgs) ||
		isDangerousChown(name, commandArgs)
}

func normalizedCommandName(command string) string {
	name := strings.ToLower(filepath.Base(strings.TrimSpace(command)))
	if runtime.GOOS == "windows" {
		name = strings.TrimSuffix(name, ".exe")
		name = strings.TrimSuffix(name, ".cmd")
		name = strings.TrimSuffix(name, ".bat")
	}

	return name
}

func isDangerousRm(name string, args []string) bool {
	if name != "rm" {
		return false
	}

	return hasRecursiveArg(args) && hasForceArg(args) && hasRootTarget(args)
}

func isDangerousChmod(name string, args []string) bool {
	if name != "chmod" {
		return false
	}

	return hasRecursiveArg(args) && hasArg(args, "777") && hasRootTarget(args)
}

func isDangerousChown(name string, args []string) bool {
	return name == "chown" && hasRecursiveArg(args)
}

func hasRecursiveArg(args []string) bool {
	for _, arg := range args {
		if arg == "-R" || arg == "-r" {
			return true
		}
		if strings.HasPrefix(arg, "-") && strings.ContainsAny(arg[1:], "Rr") {
			return true
		}
	}

	return false
}

func hasForceArg(args []string) bool {
	for _, arg := range args {
		if arg == "-f" {
			return true
		}
		if strings.HasPrefix(arg, "-") && strings.Contains(arg[1:], "f") {
			return true
		}
	}

	return false
}

func hasRootTarget(args []string) bool {
	for _, arg := range args {
		if arg == "/" {
			return true
		}
	}

	return false
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}

	return false
}

func blockedShellCommandError(command string) error {
	return fmt.Errorf("run_shell blocked dangerous command %q", command)
}
