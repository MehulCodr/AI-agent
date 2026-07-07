package context

import (
	stdcontext "context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	apperrors "github.com/MehulCodr/AI-agent/internal/errors"
)

func TestScannerCountsFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md")
	writeFile(t, root, "internal/app.go")
	writeFile(t, root, "internal/app_test.go")
	writeFile(t, root, "package.json")

	summary, err := NewScanner(root).Scan(stdcontext.Background())
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if summary.TotalFiles != 4 {
		t.Fatalf("TotalFiles = %d, want 4", summary.TotalFiles)
	}
}

func TestScannerIgnoresGitAndVendor(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go")
	writeFile(t, root, ".git/config")
	writeFile(t, root, "vendor/example/vendor.go")
	writeFile(t, root, ".agent/config.json")
	writeFile(t, root, ".agents/state.json")
	writeFile(t, root, ".codex/config.json")
	writeFile(t, root, ".env")

	summary, err := NewScanner(root).Scan(stdcontext.Background())
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if summary.TotalFiles != 1 {
		t.Fatalf("TotalFiles = %d, want 1", summary.TotalFiles)
	}
	if strings.Contains(summary.Tree, ".git") || strings.Contains(summary.Tree, "vendor") || strings.Contains(summary.Tree, ".agent") || strings.Contains(summary.Tree, ".env") {
		t.Fatalf("Tree contains ignored directories: %q", summary.Tree)
	}
}

func TestScannerDetectsGoFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "cmd/main.go")
	writeFile(t, root, "internal/app.go")
	writeFile(t, root, "README.md")

	summary, err := NewScanner(root).Scan(stdcontext.Background())
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if summary.GoFiles != 2 {
		t.Fatalf("GoFiles = %d, want 2", summary.GoFiles)
	}
	if summary.Languages["Go"] != 2 {
		t.Fatalf("Languages[Go] = %d, want 2", summary.Languages["Go"])
	}
}

func TestScannerRespectsContextCancellation(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go")

	ctx, cancel := stdcontext.WithCancel(stdcontext.Background())
	cancel()

	_, err := NewScanner(root).Scan(ctx)
	if !errors.Is(err, stdcontext.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestScannerRequiresContext(t *testing.T) {
	_, err := NewScanner(t.TempDir()).Scan(nil)
	if !errors.Is(err, apperrors.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestScannerReturnsNonEmptyTree(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "cmd/main.go")

	summary, err := NewScanner(root).Scan(stdcontext.Background())
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if summary.Tree == "" {
		t.Fatal("Tree is empty")
	}
	if !strings.Contains(summary.Tree, "cmd/") || !strings.Contains(summary.Tree, "cmd/main.go") {
		t.Fatalf("Tree = %q, want directory and file", summary.Tree)
	}
}

func TestScannerTracksImportantDirectories(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "cmd/main.go")
	writeFile(t, root, "internal/app.go")
	writeFile(t, root, "README.md")

	summary, err := NewScanner(root).Scan(stdcontext.Background())
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	got := strings.Join(summary.ImportantDirs, ",")
	if got != "cmd,internal" {
		t.Fatalf("ImportantDirs = %#v, want cmd and internal", summary.ImportantDirs)
	}
}

func TestScannerRespectsContextBudget(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "very-long-file-name.go")
	writeFile(t, root, "another-long-file-name.go")

	scanner := NewScanner(root)
	scanner.MaxContextChars = 10

	summary, err := scanner.Scan(stdcontext.Background())
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if len(summary.Tree) > scanner.MaxContextChars {
		t.Fatalf("len(Tree) = %d, want <= %d", len(summary.Tree), scanner.MaxContextChars)
	}
	if summary.TotalFiles != 2 {
		t.Fatalf("TotalFiles = %d, want 2", summary.TotalFiles)
	}
}

func writeFile(t *testing.T, root, name string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("create parent directory: %v", err)
	}
	if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
		t.Fatalf("write file %s: %v", name, err)
	}
}
