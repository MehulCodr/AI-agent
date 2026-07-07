package agent

import (
	"context"
	"fmt"
	"strings"

	apperrors "github.com/MehulCodr/AI-agent/internal/errors"
	"github.com/MehulCodr/AI-agent/internal/llm"
)

const toolCallsNotImplemented = "tool calls are not implemented yet"

func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	if a == nil {
		return "", fmt.Errorf("%w: agent is required", apperrors.ErrInvalidInput)
	}
	if ctx == nil {
		return "", fmt.Errorf("%w: context is required", apperrors.ErrInvalidInput)
	}
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("%w: input cannot be empty", apperrors.ErrInvalidInput)
	}
	if a.provider == nil {
		return "", fmt.Errorf("%w: llm provider is required", apperrors.ErrInvalidInput)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	a.memory.Add(llm.Message{
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

	a.memory.Add(response)
	a.syncSession()
	return response.Content, nil
}
