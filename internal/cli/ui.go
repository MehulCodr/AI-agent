package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/MehulCodr/AI-agent/internal/agent"
)

func terminalEventWriter(output io.Writer, streamed *bool) agent.EventHandler {
	color := colorEnabled()
	return func(event agent.Event) error {
		switch event.Type {
		case "delta":
			*streamed = true
			_, err := fmt.Fprint(output, event.Content)
			return err
		case "tool_start":
			_, err := fmt.Fprintf(output, "\n%s %s\n", paint(color, "36", "[tool]"), event.Tool)
			return err
		case "tool_error":
			_, err := fmt.Fprintf(output, "%s %s\n", paint(color, "31", "[tool error]"), event.Content)
			return err
		case "tool_result":
			summary := strings.TrimSpace(event.Content)
			if len(summary) > 240 {
				summary = summary[:240] + "..."
			}
			if summary == "" {
				summary = "completed"
			}
			_, err := fmt.Fprintf(output, "%s %s\n", paint(color, "32", "[tool result]"), summary)
			return err
		default:
			return nil
		}
	}
}

func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	term := strings.ToLower(os.Getenv("TERM"))
	return term != "" && term != "dumb"
}

func paint(enabled bool, code, text string) string {
	if !enabled {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}
