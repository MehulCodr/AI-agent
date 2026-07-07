package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MehulCodr/AI-agent/internal/agent"
	projectcontext "github.com/MehulCodr/AI-agent/internal/context"
	apperrors "github.com/MehulCodr/AI-agent/internal/errors"
	"github.com/MehulCodr/AI-agent/internal/llm"
)

const (
	appName = "AI-agent"
	version = "v0.1.0"
)

// Run parses command-line arguments and routes to the requested command.
func Run(args []string) error {
	if len(args) < 2 {
		return usageError()
	}

	switch args[1] {
	case "version":
		return runVersion()
	case "init":
		return runInit()
	case "chat":
		stream, rest, err := parseStreamFlag(args[2:])
		if err != nil {
			return err
		}
		if len(rest) > 0 {
			return fmt.Errorf("%w: unexpected chat argument %q", apperrors.ErrInvalidInput, rest[0])
		}
		agent, err := newAgent()
		if err != nil {
			return err
		}
		if stream {
			ctx, stop := interruptContext()
			defer stop()
			return StartStreamingREPL(ctx, os.Stdin, os.Stdout, agent)
		}
		return StartREPL(context.Background(), os.Stdin, os.Stdout, agent)
	case "context":
		return runContext()
	case "run":
		return runTask(args[2:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("%w: unknown command %q\n\n%s", apperrors.ErrInvalidInput, args[1], usageText())
	}
}

func runVersion() error {
	fmt.Printf("%s %s\n", appName, version)
	return nil
}

func runInit() error {
	agentDir := ".agent"
	configPath := filepath.Join(agentDir, "config.json")

	if err := os.MkdirAll(agentDir, 0755); err != nil {
		return fmt.Errorf("create %s directory: %w", agentDir, err)
	}

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("%s already exists; leaving it unchanged\n", configPath)
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect %s: %w", configPath, err)
	}

	config := map[string]string{
		"version": version,
		"model":   llm.DefaultModel,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("create default config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", configPath, err)
	}

	fmt.Printf("Initialized %s\n", configPath)
	return nil
}

func runTask(parts []string) error {
	stream, parts, err := parseStreamFlag(parts)
	if err != nil {
		return err
	}

	task := strings.TrimSpace(strings.Join(parts, " "))
	if task == "" {
		return fmt.Errorf(`%w: missing task: usage: agent run "task"`, apperrors.ErrInvalidInput)
	}

	agent, err := newAgent()
	if err != nil {
		return err
	}

	if stream {
		ctx, stop := interruptContext()
		defer stop()

		chunks, err := agent.Stream(ctx, task)
		if err != nil {
			return err
		}

		fmt.Println("Thinking...")
		if err := printStream(ctx, os.Stdout, chunks); err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		if err := ctx.Err(); err != nil {
			return nil
		}
		return agent.SaveSession()
	}

	response, err := agent.Run(context.Background(), task)
	if err != nil {
		return err
	}
	if err := agent.SaveSession(); err != nil {
		return err
	}

	fmt.Println(response)
	return nil
}

func runContext() error {
	summary, err := projectcontext.NewScanner("").Scan(context.Background())
	if err != nil {
		return err
	}

	fmt.Printf("Root: %s\n", summary.Root)
	fmt.Printf("Total files: %d\n", summary.TotalFiles)
	fmt.Printf("Go files: %d\n", summary.GoFiles)
	fmt.Printf("Languages: %v\n", summary.Languages)
	fmt.Printf("Important directories: %s\n", strings.Join(summary.ImportantDirs, ", "))
	fmt.Println("Tree:")
	fmt.Println(summary.Tree)
	return nil
}

func newAgent() (*agent.Agent, error) {
	provider, err := newProvider()
	if err != nil {
		return nil, err
	}

	session, err := agent.LoadLatestSession()
	if err != nil {
		return nil, err
	}

	return agent.NewWithSession(provider, session), nil
}

func usageError() error {
	return fmt.Errorf("%w: missing command\n\n%s", apperrors.ErrInvalidInput, usageText())
}

func printUsage() {
	fmt.Println(usageText())
}

func parseStreamFlag(args []string) (bool, []string, error) {
	stream := false
	rest := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--stream" {
			stream = true
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return false, nil, fmt.Errorf("%w: unknown flag %q", apperrors.ErrInvalidInput, arg)
		}
		rest = append(rest, arg)
	}
	return stream, rest, nil
}

func usageText() string {
	return `Usage:
  agent version
  agent init
  agent context
  agent chat [--stream]
  agent run "task" [--stream]`
}
