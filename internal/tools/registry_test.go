package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	apperrors "github.com/MehulCodr/AI-agent/internal/errors"
)

func TestRegistryRejectsInvalidTool(t *testing.T) {
	registry := NewRegistry()

	if err := registry.Register(nil); !errors.Is(err, apperrors.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
	if err := registry.Register(fakeTool{}); !errors.Is(err, apperrors.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestRegistryRegisterTool(t *testing.T) {
	registry := NewRegistry()
	tool := fakeTool{name: "test_tool"}

	if err := registry.Register(tool); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if len(registry.List()) != 1 {
		t.Fatalf("tool count = %d, want 1", len(registry.List()))
	}
}

func TestRegistryGetRegisteredTool(t *testing.T) {
	registry := NewRegistry()
	tool := fakeTool{name: "test_tool"}

	if err := registry.Register(tool); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	got, err := registry.Get("test_tool")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Name() != "test_tool" {
		t.Fatalf("tool name = %q, want test_tool", got.Name())
	}
}

func TestRegistryDuplicateRegistrationFails(t *testing.T) {
	registry := NewRegistry()
	tool := fakeTool{name: "test_tool"}

	if err := registry.Register(tool); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	err := registry.Register(tool)
	if !errors.Is(err, apperrors.ErrInvalidInput) || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("error = %v, want duplicate registration ErrInvalidInput", err)
	}
}

func TestRegistryMissingToolReturnsError(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Get("missing_tool")
	if !errors.Is(err, apperrors.ErrToolNotFound) || !strings.Contains(err.Error(), "missing_tool") {
		t.Fatalf("error = %v, want ErrToolNotFound for missing tool", err)
	}
}

func TestRegistryNamesReturnsRegisteredToolNames(t *testing.T) {
	registry := NewRegistry()

	for _, tool := range []Tool{
		fakeTool{name: "zeta"},
		fakeTool{name: "alpha"},
	} {
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Register returned error: %v", err)
		}
	}

	got := registry.Names()
	want := []string{"alpha", "zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("names = %#v, want %#v", got, want)
	}
}

func TestEchoToolReturnsText(t *testing.T) {
	got, err := EchoTool{}.Execute(context.Background(), map[string]any{"text": "hello"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("result = %q, want hello", got)
	}
}

func TestEchoToolRequiresText(t *testing.T) {
	_, err := EchoTool{}.Execute(context.Background(), nil)
	if !errors.Is(err, apperrors.ErrInvalidInput) || !strings.Contains(err.Error(), "text") {
		t.Fatalf("error = %v, want ErrInvalidInput text error", err)
	}
}

func TestCurrentDirectoryToolReturnsPath(t *testing.T) {
	got, err := CurrentDirectoryTool{}.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got == "" {
		t.Fatal("path is empty")
	}
}

func TestListFilesToolListsTemporaryDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "one.txt"), []byte("one"), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "folder"), 0755); err != nil {
		t.Fatalf("create test directory: %v", err)
	}

	got, err := ListFilesTool{}.Execute(context.Background(), map[string]any{"path": dir})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	lines := strings.Split(got, "\n")
	want := []string{"folder", "one.txt"}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("files = %#v, want %#v", lines, want)
	}
}

type fakeTool struct {
	name string
}

func (f fakeTool) Name() string {
	return f.name
}

func (f fakeTool) Description() string {
	return "fake tool"
}

func (f fakeTool) Execute(ctx context.Context, input map[string]any) (string, error) {
	_, _ = ctx, input
	return "", nil
}
