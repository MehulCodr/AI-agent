package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type ListFilesTool struct{}

func (ListFilesTool) Name() string {
	return "list_files"
}

func (ListFilesTool) Description() string {
	return "Lists files in a directory."
}

func (ListFilesTool) Execute(ctx context.Context, input map[string]any) (string, error) {
	if err := contextError(ctx); err != nil {
		return "", err
	}

	path := "."
	if value, ok := input["path"]; ok {
		text, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("list_files tool path must be a string")
		}
		if text != "" {
			path = text
		}
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("list files in %q: %w", path, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return strings.Join(names, "\n"), nil
}
