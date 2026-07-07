package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	apperrors "github.com/MehulCodr/AI-agent/internal/errors"
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
	apply, err := optionalBool(input, "apply", "edit_file")
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
		return "", fmt.Errorf("%w: edit file %q: old text not found", apperrors.ErrInvalidInput, path)
	}

	diff := editDiff(path, oldText, newText)
	if !apply {
		return diff, nil
	}

	updated := strings.Replace(content, oldText, newText, 1)
	if err := writeBackup(safePath, data); err != nil {
		return "", err
	}
	if err := os.WriteFile(safePath, []byte(updated), 0644); err != nil {
		return "", fmt.Errorf("write file %q: %w", path, err)
	}

	return diff, nil
}

func optionalBool(input map[string]any, key, toolName string) (bool, error) {
	value, ok := input[key]
	if !ok || value == nil {
		return false, nil
	}

	typed, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%w: %s tool %s must be a boolean", apperrors.ErrInvalidInput, toolName, key)
	}

	return typed, nil
}

func writeBackup(path string, content []byte) error {
	backupPath := path + ".bak"
	if err := ensureExistingTargetInsideProject(backupPath); err != nil {
		return err
	}

	existing, err := os.ReadFile(backupPath)
	if err == nil && string(existing) == string(content) {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read backup %q: %w", backupPath, err)
	}

	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return fmt.Errorf("write backup %q: %w", backupPath, err)
	}

	return nil
}
