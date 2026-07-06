package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ReadFileTool struct{}

func (ReadFileTool) Name() string {
	return "read_file"
}

func (ReadFileTool) Description() string {
	return "Reads a file inside the project root."
}

func (ReadFileTool) Execute(ctx context.Context, input map[string]any) (string, error) {
	if err := contextError(ctx); err != nil {
		return "", err
	}

	path, err := requiredString(input, "path", "read_file")
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

	return string(data), nil
}

func requiredString(input map[string]any, key, toolName string) (string, error) {
	text, err := stringInput(input, key, toolName)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("%s tool %s cannot be empty", toolName, key)
	}

	return text, nil
}

func stringInput(input map[string]any, key, toolName string) (string, error) {
	value, ok := input[key]
	if !ok {
		return "", fmt.Errorf("%s tool requires %s", toolName, key)
	}

	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s tool %s must be a string", toolName, key)
	}

	return text, nil
}

func safeProjectPath(path string) (string, error) {
	root, err := projectRoot()
	if err != nil {
		return "", err
	}

	cleaned := filepath.Clean(path)
	if filepath.IsAbs(cleaned) {
		return ensureInsideProject(root, cleaned)
	}

	return ensureInsideProject(root, filepath.Join(root, cleaned))
}

func projectRoot() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}

	root, err = filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}

	return root, nil
}

func ensureInsideProject(root, target string) (string, error) {
	target, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", fmt.Errorf("validate path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("unsafe path: path must stay inside project root")
	}

	return target, nil
}

func ensureExistingTargetInsideProject(path string) error {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("resolve path: %w", err)
	}

	root, err := projectRoot()
	if err != nil {
		return err
	}

	_, err = ensureInsideProject(root, resolved)
	return err
}

func safeWritableProjectPath(path string) (string, error) {
	safePath, err := safeProjectPath(path)
	if err != nil {
		return "", err
	}
	if err := ensureExistingTargetInsideProject(safePath); err != nil {
		return "", err
	}

	parent := filepath.Dir(safePath)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return "", fmt.Errorf("create parent directories for %q: %w", path, err)
	}

	resolvedParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", fmt.Errorf("resolve parent directory: %w", err)
	}

	root, err := projectRoot()
	if err != nil {
		return "", err
	}

	resolvedParent, err = ensureInsideProject(root, resolvedParent)
	if err != nil {
		return "", err
	}

	return filepath.Join(resolvedParent, filepath.Base(safePath)), nil
}
