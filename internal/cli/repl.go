package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/MehulCodr/AI-agent/internal/agent"
)

type ChatRunner interface {
	Run(ctx context.Context, input string) (string, error)
	Clear()
}

type StreamingChatRunner interface {
	RunStream(ctx context.Context, input string, onEvent agent.EventHandler) (string, error)
}

type ToolLister interface {
	Tools() []string
}

type REPLConfig struct {
	Stream bool
}

// StartREPL runs the interactive chat loop.
func StartREPL(ctx context.Context, input io.Reader, output io.Writer, runner ChatRunner, config REPLConfig) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}
	if runner == nil {
		return fmt.Errorf("chat runner is required")
	}

	reader, ok := input.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(input)
	}

	fmt.Fprintln(output, "AI-agent chat")
	fmt.Fprintln(output, `Type /help for commands or /exit to quit.`)

	for {
		fmt.Fprint(output, "> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read input: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch line {
		case "/exit":
			fmt.Fprintln(output, "Goodbye.")
			return nil
		case "/help":
			printREPLHelp(output)
		case "/clear":
			runner.Clear()
			fmt.Fprintln(output, "Conversation cleared.")
		case "/tools":
			printRunnerTools(output, runner)
		default:
			if err := runREPLTurn(ctx, output, runner, line, config.Stream); err != nil {
				fmt.Fprintf(output, "Error: %v\n", err)
			}
		}
	}

	fmt.Fprintln(output)
	return nil
}

func runREPLTurn(ctx context.Context, output io.Writer, runner ChatRunner, line string, stream bool) error {
	if stream {
		if streaming, ok := runner.(StreamingChatRunner); ok {
			fmt.Fprint(output, "Agent: ")
			streamed := false
			response, err := streaming.RunStream(ctx, line, terminalEventWriter(output, &streamed))
			if err != nil {
				return err
			}
			if !streamed && response != "" {
				fmt.Fprint(output, response)
			}
			fmt.Fprintln(output)
			return nil
		}
	}

	response, err := runner.Run(ctx, line)
	if err != nil {
		return err
	}
	fmt.Fprintf(output, "Agent: %s\n", response)
	return nil
}

func printREPLHelp(output io.Writer) {
	fmt.Fprintln(output, "Available commands:")
	fmt.Fprintln(output, "  /help  Show available commands")
	fmt.Fprintln(output, "  /tools Show registered tools")
	fmt.Fprintln(output, "  /clear Clear the conversation")
	fmt.Fprintln(output, "  /exit  Exit the REPL")
}

func printRunnerTools(output io.Writer, runner ChatRunner) {
	lister, ok := runner.(ToolLister)
	if !ok {
		fmt.Fprintln(output, "Tool list unavailable.")
		return
	}
	for _, tool := range lister.Tools() {
		fmt.Fprintln(output, tool)
	}
}
