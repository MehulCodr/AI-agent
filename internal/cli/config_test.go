package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MehulCodr/AI-agent/internal/llm"
)

func TestLoadRuntimeConfigReadsDotEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_MODEL", "")
	t.Setenv("GEMINI_BASE_URL", "")

	data := []byte("GEMINI_API_KEY=test-key\nGEMINI_MODEL=test-model\nGEMINI_BASE_URL=https://example.test/v1beta\n")
	if err := os.WriteFile(".env", data, 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	got, err := loadRuntimeConfig()
	if err != nil {
		t.Fatalf("loadRuntimeConfig returned error: %v", err)
	}
	if got.GeminiAPIKey != "test-key" {
		t.Fatalf("GeminiAPIKey = %q, want test-key", got.GeminiAPIKey)
	}
	if got.Model != "test-model" {
		t.Fatalf("Model = %q, want test-model", got.Model)
	}
	if got.GeminiBaseURL != "https://example.test/v1beta" {
		t.Fatalf("GeminiBaseURL = %q, want https://example.test/v1beta", got.GeminiBaseURL)
	}
}

func TestLoadRuntimeConfigUsesLocalModel(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_MODEL", "")
	t.Setenv("GEMINI_BASE_URL", "")

	if err := os.Mkdir(".agent", 0755); err != nil {
		t.Fatalf("create .agent: %v", err)
	}
	configPath := filepath.Join(".agent", "config.json")
	if err := os.WriteFile(configPath, []byte(`{"model":"local-model"}`), 0644); err != nil {
		t.Fatalf("write local config: %v", err)
	}

	got, err := loadRuntimeConfig()
	if err != nil {
		t.Fatalf("loadRuntimeConfig returned error: %v", err)
	}
	if got.Model != "local-model" {
		t.Fatalf("Model = %q, want local-model", got.Model)
	}
}

func TestLoadRuntimeConfigPrefersEnvironment(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GEMINI_API_KEY", "env-key")
	t.Setenv("GEMINI_MODEL", "env-model")
	t.Setenv("GEMINI_BASE_URL", "")

	if err := os.WriteFile(".env", []byte("GEMINI_API_KEY=file-key\nGEMINI_MODEL=file-model\n"), 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	got, err := loadRuntimeConfig()
	if err != nil {
		t.Fatalf("loadRuntimeConfig returned error: %v", err)
	}
	if got.GeminiAPIKey != "env-key" {
		t.Fatalf("GeminiAPIKey = %q, want env-key", got.GeminiAPIKey)
	}
	if got.Model != "env-model" {
		t.Fatalf("Model = %q, want env-model", got.Model)
	}
}

func TestNewProviderUsesMockWithoutAPIKey(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_MODEL", "")
	t.Setenv("GEMINI_BASE_URL", "")

	provider, err := newProvider()
	if err != nil {
		t.Fatalf("newProvider returned error: %v", err)
	}
	if _, ok := provider.(llm.MockProvider); !ok {
		t.Fatalf("provider type = %T, want llm.MockProvider", provider)
	}
}

func TestNewProviderUsesGeminiWithAPIKey(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_MODEL", "")
	t.Setenv("GEMINI_BASE_URL", "")

	if err := os.WriteFile(".env", []byte("GEMINI_API_KEY=test-key\nGEMINI_MODEL=test-model\n"), 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	provider, err := newProvider()
	if err != nil {
		t.Fatalf("newProvider returned error: %v", err)
	}
	if _, ok := provider.(*llm.GeminiProvider); !ok {
		t.Fatalf("provider type = %T, want *llm.GeminiProvider", provider)
	}
}
