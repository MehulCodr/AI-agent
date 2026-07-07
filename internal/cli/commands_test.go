package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	apperrors "github.com/MehulCodr/AI-agent/internal/errors"
)

func TestRunRequiresCommand(t *testing.T) {
	err := Run([]string{"agent"})
	if !errors.Is(err, apperrors.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	err := Run([]string{"agent", "nope"})
	if !errors.Is(err, apperrors.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestRunRejectsMissingTask(t *testing.T) {
	err := Run([]string{"agent", "run"})
	if !errors.Is(err, apperrors.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestRunInitCreatesConfig(t *testing.T) {
	dir := t.TempDir()
	chdirTest(t, dir)

	if err := Run([]string{"agent", "init"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	path := filepath.Join(dir, ".agent", "config.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat config: %v", err)
	}
}

func TestLoadDotEnvRejectsMalformedLine(t *testing.T) {
	dir := t.TempDir()
	chdirTest(t, dir)

	if err := os.WriteFile(".env", []byte("BROKEN\n"), 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	err := loadDotEnv(".env")
	if !errors.Is(err, apperrors.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}
