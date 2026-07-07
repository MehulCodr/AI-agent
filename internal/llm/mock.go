package llm

import (
	"context"
	"strings"
)

type MockProvider struct {
	Response string
}

func (p MockProvider) Chat(ctx context.Context, messages []Message) (Message, error) {
	if err := ctx.Err(); err != nil {
		return Message{}, err
	}

	content := p.Response
	if content == "" {
		if last := lastUserMessage(messages); strings.HasPrefix(last, "Tool results:") {
			return Message{Role: "assistant", Content: "mock response:\n" + last}, nil
		}
		if call, ok := mockToolCall(messages); ok {
			return Message{Role: "assistant", ToolCalls: []ToolCall{call}}, nil
		}

		content = "mock response"
		if last := lastUserMessage(messages); last != "" {
			content += ": " + last
		}
	}

	return Message{Role: "assistant", Content: content}, nil
}

func lastUserMessage(messages []Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return strings.TrimSpace(messages[i].Content)
		}
	}

	return ""
}

func mockToolCall(messages []Message) (ToolCall, bool) {
	last := strings.ToLower(lastUserMessage(messages))
	if strings.Contains(last, "list") && (strings.Contains(last, "file") || strings.Contains(last, "director")) {
		return ToolCall{
			Type: "function",
			Function: ToolCallFunction{
				Name:      "list_files",
				Arguments: `{"path":"."}`,
			},
		}, true
	}

	return ToolCall{}, false
}
