package cli

import (
	"os"
	"testing"

	"github.com/MehulCodr/AI-agent/internal/llm"
)

func TestLoadDotEnvSetsAgentConfig(t *testing.T) {
	chdirTest(t, t.TempDir())
	t.Setenv("AI_AGENT_PROVIDER", "")
	t.Setenv("AI_AGENT_MODEL", "")
	t.Setenv("AI_AGENT_API_KEY", "")
	t.Setenv("AI_AGENT_BASE_URL", "")

	data := []byte("AI_AGENT_PROVIDER=gemini\nAI_AGENT_MODEL=test-model\nAI_AGENT_API_KEY=test-key\nAI_AGENT_BASE_URL=https://example.test/v1beta\n")
	if err := os.WriteFile(".env", data, 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	if err := loadDotEnv(".env"); err != nil {
		t.Fatalf("loadDotEnv returned error: %v", err)
	}

	got := llm.LoadConfigFromEnv()
	if got.Provider != "gemini" {
		t.Fatalf("Provider = %q, want gemini", got.Provider)
	}
	if got.Model != "test-model" {
		t.Fatalf("Model = %q, want test-model", got.Model)
	}
	if got.APIKey != "test-key" {
		t.Fatalf("APIKey = %q, want test-key", got.APIKey)
	}
	if got.BaseURL != "https://example.test/v1beta" {
		t.Fatalf("BaseURL = %q, want https://example.test/v1beta", got.BaseURL)
	}
}

func TestLoadDotEnvDoesNotOverrideEnvironment(t *testing.T) {
	chdirTest(t, t.TempDir())
	t.Setenv("AI_AGENT_PROVIDER", "mock")
	t.Setenv("AI_AGENT_MODEL", "env-model")
	t.Setenv("AI_AGENT_API_KEY", "env-key")
	t.Setenv("AI_AGENT_BASE_URL", "")

	if err := os.WriteFile(".env", []byte("AI_AGENT_PROVIDER=gemini\nAI_AGENT_MODEL=file-model\nAI_AGENT_API_KEY=file-key\n"), 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	if err := loadDotEnv(".env"); err != nil {
		t.Fatalf("loadDotEnv returned error: %v", err)
	}

	got := llm.LoadConfigFromEnv()
	if got.Provider != "mock" {
		t.Fatalf("Provider = %q, want mock", got.Provider)
	}
	if got.Model != "env-model" {
		t.Fatalf("Model = %q, want env-model", got.Model)
	}
	if got.APIKey != "env-key" {
		t.Fatalf("APIKey = %q, want env-key", got.APIKey)
	}
}

func TestNewProviderUsesMockByDefault(t *testing.T) {
	chdirTest(t, t.TempDir())
	t.Setenv("AI_AGENT_PROVIDER", "")
	t.Setenv("AI_AGENT_MODEL", "")
	t.Setenv("AI_AGENT_API_KEY", "")
	t.Setenv("AI_AGENT_BASE_URL", "")

	provider, err := newProvider()
	if err != nil {
		t.Fatalf("newProvider returned error: %v", err)
	}
	if _, ok := provider.(llm.MockProvider); !ok {
		t.Fatalf("provider type = %T, want llm.MockProvider", provider)
	}
}

func TestNewProviderUsesGeminiWithAPIKey(t *testing.T) {
	chdirTest(t, t.TempDir())
	t.Setenv("AI_AGENT_PROVIDER", "")
	t.Setenv("AI_AGENT_MODEL", "")
	t.Setenv("AI_AGENT_API_KEY", "")
	t.Setenv("AI_AGENT_BASE_URL", "")

	if err := os.WriteFile(".env", []byte("AI_AGENT_PROVIDER=gemini\nAI_AGENT_API_KEY=test-key\n"), 0600); err != nil {
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

func chdirTest(t *testing.T, dir string) {
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
