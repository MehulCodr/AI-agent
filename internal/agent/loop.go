package agent

import (
	"context"
	"errors"
	"strings"

	"github.com/MehulCodr/AI-agent/internal/llm"
)

const toolCallsNotImplemented = "tool calls are not implemented yet"

func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	if a == nil {
		return "", errors.New("agent is required")
	}
	if ctx == nil {
		return "", errors.New("context is required")
	}
	if strings.TrimSpace(input) == "" {
		return "", errors.New("input cannot be empty")
	}
	if a.provider == nil {
		return "", errors.New("llm provider is required")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	a.messages = append(a.messages, llm.Message{
		Role:    "user",
		Content: input,
	})

	response, err := a.provider.Chat(ctx, a.Messages())
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if len(response.ToolCalls) > 0 {
		response.Content = toolCallsNotImplemented
	}

	a.messages = append(a.messages, response)
	return response.Content, nil
}
