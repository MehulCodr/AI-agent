package tools

import (
	"context"
	"fmt"
)

type EchoTool struct{}

func (EchoTool) Name() string {
	return "echo"
}

func (EchoTool) Description() string {
	return "Returns the provided text."
}

func (EchoTool) Execute(ctx context.Context, input map[string]any) (string, error) {
	if err := contextError(ctx); err != nil {
		return "", err
	}

	value, ok := input["text"]
	if !ok {
		return "", fmt.Errorf("echo tool requires text")
	}

	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("echo tool text must be a string")
	}

	return text, nil
}
