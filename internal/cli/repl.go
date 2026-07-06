package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

type ChatRunner interface {
	Run(ctx context.Context, input string) (string, error)
	Clear()
}

// StartREPL runs the interactive chat loop.
func StartREPL(ctx context.Context, input io.Reader, output io.Writer, runner ChatRunner) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}
	if runner == nil {
		return fmt.Errorf("chat runner is required")
	}

	scanner := bufio.NewScanner(input)

	fmt.Fprintln(output, "Welcome to AI-agent chat.")
	fmt.Fprintln(output, `Type /help for commands or /exit to quit.`)

	for {
		fmt.Fprint(output, "> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
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
		default:
			response, err := runner.Run(ctx, line)
			if err != nil {
				fmt.Fprintf(output, "Error: %v\n", err)
				continue
			}
			fmt.Fprintf(output, "Agent: %s\n", response)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	fmt.Fprintln(output)
	return nil
}

func printREPLHelp(output io.Writer) {
	fmt.Fprintln(output, "Available commands:")
	fmt.Fprintln(output, "  /help  Show available commands")
	fmt.Fprintln(output, "  /clear Clear the conversation")
	fmt.Fprintln(output, "  /exit  Exit the REPL")
}
