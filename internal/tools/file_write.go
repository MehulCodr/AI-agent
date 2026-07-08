package tools

import (
	"context"
	"fmt"
	"os"
)

type WriteFileTool struct{}

func (WriteFileTool) Name() string {
	return "write_file"
}

func (WriteFileTool) Description() string {
	return "Writes content to a file inside the project root."
}

func (WriteFileTool) Parameters() map[string]any {
	return objectSchema([]string{"path", "content"}, map[string]any{
		"path":    stringProperty("Relative path to write."),
		"content": stringProperty("Complete file contents to write."),
	})
}

func (WriteFileTool) Execute(ctx context.Context, input map[string]any) (string, error) {
	if err := contextError(ctx); err != nil {
		return "", err
	}

	path, err := requiredString(input, "path", "write_file")
	if err != nil {
		return "", err
	}
	content, err := stringInput(input, "content", "write_file")
	if err != nil {
		return "", err
	}

	safePath, err := safeWritableProjectPath(path)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(safePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write file %q: %w", path, err)
	}

	return "file written", nil
}
