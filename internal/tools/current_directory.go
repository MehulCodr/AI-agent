package tools

import (
	"context"
	"os"
)

type CurrentDirectoryTool struct{}

func (CurrentDirectoryTool) Name() string {
	return "current_directory"
}

func (CurrentDirectoryTool) Description() string {
	return "Returns the current working directory."
}

func (CurrentDirectoryTool) Execute(ctx context.Context, input map[string]any) (string, error) {
	_ = input

	if err := contextError(ctx); err != nil {
		return "", err
	}

	return os.Getwd()
}
