package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// StartREPL runs the interactive chat loop.
func StartREPL(input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)

	fmt.Fprintln(output, "Welcome to ai-cli-agent chat.")
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
			fmt.Fprintln(output, "Conversation cleared.")
		default:
			fmt.Fprintf(output, "Agent: received %q\n", line)
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
