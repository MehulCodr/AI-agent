package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditFilePreviewDiffDoesNotModifyFile(t *testing.T) {
	withProjectRoot(t, func(root string) {
		path := filepath.Join(root, "notes.txt")
		if err := os.WriteFile(path, []byte("hello old world"), 0644); err != nil {
			t.Fatalf("write test file: %v", err)
		}

		got, err := EditFileTool{}.Execute(context.Background(), map[string]any{
			"path":  "notes.txt",
			"old":   "old",
			"new":   "new",
			"apply": false,
		})
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}
		if !strings.Contains(got, "-old") || !strings.Contains(got, "+new") {
			t.Fatalf("diff = %q, want removed and added lines", got)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read file: %v", err)
		}
		if string(data) != "hello old world" {
			t.Fatalf("content = %q, want preview to leave file unchanged", string(data))
		}
		if _, err := os.Stat(path + ".bak"); !os.IsNotExist(err) {
			t.Fatalf("backup err = %v, want no backup for preview", err)
		}
	})
}

func TestEditFileApplyModifiesFile(t *testing.T) {
	withProjectRoot(t, func(root string) {
		path := filepath.Join(root, "notes.txt")
		if err := os.WriteFile(path, []byte("one two two"), 0644); err != nil {
			t.Fatalf("write test file: %v", err)
		}

		got, err := EditFileTool{}.Execute(context.Background(), map[string]any{
			"path":  "notes.txt",
			"old":   "two",
			"new":   "three",
			"apply": true,
		})
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}
		if !strings.Contains(got, "-two") || !strings.Contains(got, "+three") {
			t.Fatalf("diff = %q, want removed and added lines", got)
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

func TestEditFileApplyCreatesBackup(t *testing.T) {
	withProjectRoot(t, func(root string) {
		path := filepath.Join(root, "notes.txt")
		if err := os.WriteFile(path, []byte("before"), 0644); err != nil {
			t.Fatalf("write test file: %v", err)
		}

		_, err := EditFileTool{}.Execute(context.Background(), map[string]any{
			"path":  "notes.txt",
			"old":   "before",
			"new":   "after",
			"apply": true,
		})
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}

		backup, err := os.ReadFile(path + ".bak")
		if err != nil {
			t.Fatalf("read backup: %v", err)
		}
		if string(backup) != "before" {
			t.Fatalf("backup = %q, want original content", string(backup))
		}
	})
}

func TestEditFileRejectsUnsafePath(t *testing.T) {
	withProjectRoot(t, func(root string) {
		_ = root

		_, err := EditFileTool{}.Execute(context.Background(), map[string]any{
			"path":  "../outside.txt",
			"old":   "old",
			"new":   "new",
			"apply": true,
		})
		if err == nil || !strings.Contains(err.Error(), "unsafe path") {
			t.Fatalf("error = %v, want unsafe path error", err)
		}
	})
}

func TestEditFileMissingOldTextReturnsError(t *testing.T) {
	withProjectRoot(t, func(root string) {
		if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("hello"), 0644); err != nil {
			t.Fatalf("write test file: %v", err)
		}

		_, err := EditFileTool{}.Execute(context.Background(), map[string]any{
			"path":  "notes.txt",
			"old":   "missing",
			"new":   "found",
			"apply": true,
		})
		if err == nil || !strings.Contains(err.Error(), "old text not found") {
			t.Fatalf("error = %v, want old text not found error", err)
		}
	})
}
