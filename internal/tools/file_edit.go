package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type EditFileTool struct{}

func (EditFileTool) Name() string {
	return "edit_file"
}

func (EditFileTool) Description() string {
	return "Replaces the first matching text in a file inside the project root."
}

func (EditFileTool) Execute(ctx context.Context, input map[string]any) (string, error) {
	if err := contextError(ctx); err != nil {
		return "", err
	}

	path, err := requiredString(input, "path", "edit_file")
	if err != nil {
		return "", err
	}
	oldText, err := requiredString(input, "old", "edit_file")
	if err != nil {
		return "", err
	}
	newText, err := stringInput(input, "new", "edit_file")
	if err != nil {
		return "", err
	}

	safePath, err := safeProjectPath(path)
	if err != nil {
		return "", err
	}
	if err := ensureExistingTargetInsideProject(safePath); err != nil {
		return "", err
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return "", fmt.Errorf("read file %q: %w", path, err)
	}

	content := string(data)
	if !strings.Contains(content, oldText) {
		return "", fmt.Errorf("edit file %q: old text not found", path)
	}

	updated := strings.Replace(content, oldText, newText, 1)
	if err := os.WriteFile(safePath, []byte(updated), 0644); err != nil {
		return "", fmt.Errorf("write file %q: %w", path, err)
	}

	return "file edited", nil
}
