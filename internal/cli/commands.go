package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MehulCodr/AI-agent/internal/agent"
	"github.com/MehulCodr/AI-agent/internal/rag"
	"github.com/MehulCodr/AI-agent/internal/session"
	"github.com/MehulCodr/AI-agent/internal/tools"
)

const (
	appName         = "AI-agent"
	version         = "v0.2.0"
	defaultMaxSteps = 8
)

// Run parses command-line arguments and routes to the requested command.
func Run(args []string) error {
	if len(args) < 2 {
		return usageError()
	}

	ctx := context.Background()
	reader := bufio.NewReader(os.Stdin)

	switch args[1] {
	case "version":
		return runVersion()
	case "init":
		return runInit()
	case "chat":
		options, rest, err := parseOptions(args[2:])
		if err != nil {
			return err
		}
		if len(rest) > 0 {
			return fmt.Errorf("chat does not accept positional arguments: %s", strings.Join(rest, " "))
		}
		runner, config, cleanup, err := newAgent(ctx, reader, os.Stdout, options)
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			return err
		}
		return StartREPL(ctx, reader, os.Stdout, runner, REPLConfig{Stream: config.Stream})
	case "run":
		return runTask(ctx, reader, os.Stdout, args[2:])
	case "index":
		return runIndex(ctx, args[2:])
	case "tools":
		return runTools(os.Stdout)
	case "config":
		return runConfig(os.Stdout, args[2:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[1], usageText())
	}
}

func runVersion() error {
	fmt.Printf("%s %s\n", appName, version)
	return nil
}

func runInit() error {
	if err := writeDefaultConfig(filepath.Join(".agent", "config.json")); err != nil {
		return err
	}
	if err := ensureDotEnv(".env"); err != nil {
		return err
	}

	fmt.Println("Initialized .agent/config.json and .env")
	fmt.Println("Fill AI_AGENT_MODEL and the API key for your selected provider before running the agent.")
	return nil
}

func runTask(ctx context.Context, input *bufio.Reader, output io.Writer, args []string) error {
	options, rest, err := parseOptions(args)
	if err != nil {
		return err
	}
	task := strings.TrimSpace(strings.Join(rest, " "))
	if task == "" {
		return errors.New(`missing task: usage: agent run [--provider name] [--model model] "task"`)
	}

	runner, config, cleanup, err := newAgent(ctx, input, output, options)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return err
	}

	if config.Stream {
		streamed := false
		response, err := runner.RunStream(ctx, task, terminalEventWriter(output, &streamed))
		if err != nil {
			return err
		}
		if !streamed && response != "" {
			fmt.Fprint(output, response)
		}
		fmt.Fprintln(output)
		return nil
	}

	response, err := runner.Run(ctx, task)
	if err != nil {
		return err
	}
	fmt.Fprintln(output, response)
	return nil
}

func runIndex(ctx context.Context, args []string) error {
	options, rest, err := parseOptions(args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("index does not accept positional arguments: %s", strings.Join(rest, " "))
	}

	config, err := loadRuntimeConfig(options)
	if err != nil {
		return err
	}
	index, err := newRedisIndex(config)
	if err != nil {
		return err
	}
	defer index.Close()

	count, err := index.Index(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Indexed %d files into Redis at %s\n", count, config.RedisAddr)
	return nil
}

func runTools(output io.Writer) error {
	for _, name := range []string{
		"current_directory",
		"edit_file",
		"list_files",
		"read_file",
		"search_repo",
		"shell",
		"write_file",
	} {
		fmt.Fprintln(output, name)
	}
	return nil
}

func runConfig(output io.Writer, args []string) error {
	options, rest, err := parseOptions(args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("config does not accept positional arguments: %s", strings.Join(rest, " "))
	}

	config, err := loadRuntimeConfig(options)
	if err != nil {
		return err
	}
	fmt.Fprintf(output, "provider: %s\n", config.Provider)
	fmt.Fprintf(output, "model: %s\n", config.Model)
	fmt.Fprintf(output, "session: %s\n", config.Session)
	fmt.Fprintf(output, "stream: %t\n", config.Stream)
	fmt.Fprintf(output, "max_steps: %d\n", config.MaxSteps)
	fmt.Fprintf(output, "redis_addr: %s\n", config.RedisAddr)
	fmt.Fprintf(output, "openai_api_key: %s\n", keyState(config.OpenAIAPIKey))
	fmt.Fprintf(output, "gemini_api_key: %s\n", keyState(config.GeminiAPIKey))
	fmt.Fprintf(output, "anthropic_api_key: %s\n", keyState(config.AnthropicAPIKey))
	return nil
}

func newAgent(ctx context.Context, input *bufio.Reader, output io.Writer, options commandOptions) (*agent.Agent, runtimeConfig, func() error, error) {
	config, err := loadRuntimeConfig(options)
	if err != nil {
		return nil, runtimeConfig{}, nil, err
	}

	provider, err := newProvider(config)
	if err != nil {
		return nil, runtimeConfig{}, nil, err
	}

	index, err := newRedisIndex(config)
	if err != nil {
		return nil, runtimeConfig{}, nil, err
	}

	runner := agent.New(
		provider,
		agent.WithMaxSteps(config.MaxSteps),
		agent.WithMemory(session.NewFileStore(sessionPath(config.Session))),
		agent.WithApproval(promptApproval(input, output)),
		agent.WithRepoSearcher(index),
	)
	if err := runner.LoadMemory(ctx); err != nil {
		_ = index.Close()
		return nil, runtimeConfig{}, nil, err
	}

	return runner, config, index.Close, nil
}

func newRedisIndex(config runtimeConfig) (*rag.RedisIndex, error) {
	return rag.NewRedisIndex(rag.RedisConfig{
		Addr:      config.RedisAddr,
		Password:  config.RedisPassword,
		DB:        config.RedisDB,
		Namespace: "ai-agent",
	})
}

func sessionPath(name string) string {
	return filepath.Join(".agent", "sessions", safeSessionName(name)+".json")
}

func promptApproval(input *bufio.Reader, output io.Writer) tools.ApprovalFunc {
	return func(ctx context.Context, command string) (bool, error) {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		fmt.Fprintf(output, "\nAllow shell command?\n%s\n[y/N] ", command)
		answer, err := input.ReadString('\n')
		if err != nil {
			return false, err
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		return answer == "y" || answer == "yes", nil
	}
}

func parseOptions(args []string) (commandOptions, []string, error) {
	var options commandOptions
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			rest = append(rest, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "--") {
			rest = append(rest, arg)
			continue
		}

		name, value, hasValue := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		switch name {
		case "provider":
			if !hasValue {
				i++
				if i >= len(args) {
					return options, nil, fmt.Errorf("--provider requires a value")
				}
				value = args[i]
			}
			options.Provider = value
		case "model":
			if !hasValue {
				i++
				if i >= len(args) {
					return options, nil, fmt.Errorf("--model requires a value")
				}
				value = args[i]
			}
			options.Model = value
		case "session":
			if !hasValue {
				i++
				if i >= len(args) {
					return options, nil, fmt.Errorf("--session requires a value")
				}
				value = args[i]
			}
			options.Session = value
		case "stream":
			valueBool := true
			if hasValue {
				parsed, err := strconv.ParseBool(value)
				if err != nil {
					return options, nil, fmt.Errorf("--stream requires a boolean value")
				}
				valueBool = parsed
			}
			options.Stream = &valueBool
		case "no-stream":
			valueBool := false
			options.Stream = &valueBool
		default:
			return options, nil, fmt.Errorf("unknown option --%s", name)
		}
	}
	return options, rest, nil
}

func keyState(value string) string {
	if strings.TrimSpace(value) == "" {
		return "missing"
	}
	return "set"
}

func usageError() error {
	return fmt.Errorf("missing command\n\n%s", usageText())
}

func printUsage() {
	fmt.Println(usageText())
}

func usageText() string {
	return `Usage:
  agent version
  agent init
  agent config [--provider name] [--model model]
  agent tools
  agent index
  agent chat [--provider name] [--model model] [--session name] [--no-stream]
  agent run [--provider name] [--model model] [--session name] [--no-stream] "task"`
}
