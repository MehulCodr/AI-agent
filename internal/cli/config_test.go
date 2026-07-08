package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MehulCodr/AI-agent/internal/llm"
)

func TestLoadRuntimeConfigReadsDotEnv(t *testing.T) {
	withWorkingDir(t, t.TempDir())
	clearConfigEnv(t)

	data := []byte("AI_AGENT_PROVIDER=gemini\nAI_AGENT_MODEL=test-model\nGEMINI_API_KEY=test-key\nGEMINI_BASE_URL=https://example.test/v1beta\n")
	if err := os.WriteFile(".env", data, 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	got, err := loadRuntimeConfig(commandOptions{})
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
	withWorkingDir(t, t.TempDir())
	clearConfigEnv(t)

	if err := os.Mkdir(".agent", 0755); err != nil {
		t.Fatalf("create .agent: %v", err)
	}
	configPath := filepath.Join(".agent", "config.json")
	if err := os.WriteFile(configPath, []byte(`{"provider":"ollama","model":"local-model"}`), 0644); err != nil {
		t.Fatalf("write local config: %v", err)
	}

	got, err := loadRuntimeConfig(commandOptions{})
	if err != nil {
		t.Fatalf("loadRuntimeConfig returned error: %v", err)
	}
	if got.Provider != "ollama" {
		t.Fatalf("Provider = %q, want ollama", got.Provider)
	}
	if got.Model != "local-model" {
		t.Fatalf("Model = %q, want local-model", got.Model)
	}
}

func TestLoadRuntimeConfigPrefersEnvironment(t *testing.T) {
	withWorkingDir(t, t.TempDir())
	clearConfigEnv(t)
	t.Setenv("AI_AGENT_PROVIDER", "openai")
	t.Setenv("AI_AGENT_MODEL", "env-model")
	t.Setenv("OPENAI_API_KEY", "env-key")

	if err := os.WriteFile(".env", []byte("AI_AGENT_PROVIDER=gemini\nAI_AGENT_MODEL=file-model\nOPENAI_API_KEY=file-key\n"), 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	got, err := loadRuntimeConfig(commandOptions{})
	if err != nil {
		t.Fatalf("loadRuntimeConfig returned error: %v", err)
	}
	if got.Provider != "openai" {
		t.Fatalf("Provider = %q, want openai", got.Provider)
	}
	if got.OpenAIAPIKey != "env-key" {
		t.Fatalf("OpenAIAPIKey = %q, want env-key", got.OpenAIAPIKey)
	}
	if got.Model != "env-model" {
		t.Fatalf("Model = %q, want env-model", got.Model)
	}
}

func TestNewProviderRequiresConfiguredKey(t *testing.T) {
	_, err := newProvider(runtimeConfig{Provider: "gemini", Model: "test-model"})
	if err == nil {
		t.Fatal("newProvider returned nil error, want missing key error")
	}
}

func TestNewProviderUsesGeminiWithAPIKey(t *testing.T) {
	provider, err := newProvider(runtimeConfig{
		Provider:     "gemini",
		Model:        "test-model",
		GeminiAPIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("newProvider returned error: %v", err)
	}
	if _, ok := provider.(*llm.GeminiProvider); !ok {
		t.Fatalf("provider type = %T, want *llm.GeminiProvider", provider)
	}
}

func TestNewProviderUsesOllamaWithoutAPIKey(t *testing.T) {
	provider, err := newProvider(runtimeConfig{
		Provider: "ollama",
		Model:    "llama3",
	})
	if err != nil {
		t.Fatalf("newProvider returned error: %v", err)
	}
	if _, ok := provider.(*llm.OllamaProvider); !ok {
		t.Fatalf("provider type = %T, want *llm.OllamaProvider", provider)
	}
}

func TestEnsureDotEnvAddsRequiredKeys(t *testing.T) {
	withWorkingDir(t, t.TempDir())
	if err := os.WriteFile(".env", []byte("GEMINI_API_KEY=keep\n"), 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	if err := ensureDotEnv(".env"); err != nil {
		t.Fatalf("ensureDotEnv returned error: %v", err)
	}
	data, err := os.ReadFile(".env")
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	content := string(data)
	for _, key := range []string{"AI_AGENT_PROVIDER=", "AI_AGENT_MODEL=", "OPENAI_API_KEY=", "ANTHROPIC_API_KEY=", "REDIS_ADDR="} {
		if !strings.Contains(content, key) {
			t.Fatalf(".env missing %s in %q", key, content)
		}
	}
	if !strings.Contains(content, "GEMINI_API_KEY=keep") {
		t.Fatalf(".env did not preserve existing key: %q", content)
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"AI_AGENT_PROVIDER",
		"AI_AGENT_MODEL",
		"AI_AGENT_SESSION",
		"AI_AGENT_STREAM",
		"AI_AGENT_MAX_STEPS",
		"OPENAI_API_KEY",
		"OPENAI_BASE_URL",
		"GEMINI_API_KEY",
		"GEMINI_BASE_URL",
		"ANTHROPIC_API_KEY",
		"ANTHROPIC_BASE_URL",
		"OLLAMA_BASE_URL",
		"REDIS_ADDR",
		"REDIS_PASSWORD",
		"REDIS_DB",
	} {
		t.Setenv(key, "")
	}
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
}
