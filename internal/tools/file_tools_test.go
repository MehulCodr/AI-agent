package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileToolReadsFile(t *testing.T) {
	withProjectRoot(t, func(root string) {
		if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("hello"), 0644); err != nil {
			t.Fatalf("write test file: %v", err)
		}

		got, err := ReadFileTool{}.Execute(context.Background(), map[string]any{"path": "notes.txt"})
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}
		if got != "hello" {
			t.Fatalf("content = %q, want hello", got)
		}
	})
}

func TestWriteFileToolWritesFile(t *testing.T) {
	withProjectRoot(t, func(root string) {
		_, err := WriteFileTool{}.Execute(context.Background(), map[string]any{
			"path":    "nested/notes.txt",
			"content": "hello",
		})
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(root, "nested", "notes.txt"))
		if err != nil {
			t.Fatalf("read written file: %v", err)
		}
		if string(data) != "hello" {
			t.Fatalf("content = %q, want hello", string(data))
		}
	})
}

func TestEditFileToolEditsFirstMatch(t *testing.T) {
	withProjectRoot(t, func(root string) {
		path := filepath.Join(root, "notes.txt")
		if err := os.WriteFile(path, []byte("one two two"), 0644); err != nil {
			t.Fatalf("write test file: %v", err)
		}

		_, err := EditFileTool{}.Execute(context.Background(), map[string]any{
			"path": "notes.txt",
			"old":  "two",
			"new":  "three",
		})
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read edited file: %v", err)
		}
		if string(data) != "one three two" {
			t.Fatalf("content = %q, want first match edited", string(data))
		}
	})
}

func TestFileToolsRejectUnsafePath(t *testing.T) {
	withProjectRoot(t, func(root string) {
		_ = root

		_, err := ReadFileTool{}.Execute(context.Background(), map[string]any{"path": "../outside.txt"})
		if err == nil || !strings.Contains(err.Error(), "unsafe path") {
			t.Fatalf("error = %v, want unsafe path error", err)
		}
	})
}

func TestReadFileToolMissingFileReturnsError(t *testing.T) {
	withProjectRoot(t, func(root string) {
		_ = root

		_, err := ReadFileTool{}.Execute(context.Background(), map[string]any{"path": "missing.txt"})
		if err == nil || !strings.Contains(err.Error(), "read file") {
			t.Fatalf("error = %v, want missing file read error", err)
		}
	})
}

func TestEditFileToolErrorsWhenOldTextNotFound(t *testing.T) {
	withProjectRoot(t, func(root string) {
		if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("hello"), 0644); err != nil {
			t.Fatalf("write test file: %v", err)
		}

		_, err := EditFileTool{}.Execute(context.Background(), map[string]any{
			"path": "notes.txt",
			"old":  "missing",
			"new":  "found",
		})
		if err == nil || !strings.Contains(err.Error(), "old text not found") {
			t.Fatalf("error = %v, want old text not found error", err)
		}
	})
}

func TestFileToolsRejectAbsolutePathOutsideProject(t *testing.T) {
	withProjectRoot(t, func(root string) {
		outside := filepath.Join(filepath.Dir(root), "outside.txt")

		_, err := WriteFileTool{}.Execute(context.Background(), map[string]any{
			"path":    outside,
			"content": "nope",
		})
		if err == nil || !strings.Contains(err.Error(), "unsafe path") {
			t.Fatalf("error = %v, want unsafe path error", err)
		}
	})
}

func withProjectRoot(t *testing.T, fn func(root string)) {
	t.Helper()

	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	fn(root)
}
