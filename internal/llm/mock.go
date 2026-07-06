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
