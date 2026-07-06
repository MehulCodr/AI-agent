package llm

import (
	"strings"
	"testing"
)

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("AI_AGENT_PROVIDER", "gemini")
	t.Setenv("AI_AGENT_MODEL", "test-model")
	t.Setenv("AI_AGENT_API_KEY", "test-key")
	t.Setenv("AI_AGENT_BASE_URL", "https://example.test/v1beta")

	got := LoadConfigFromEnv()
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

func TestLoadConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("AI_AGENT_PROVIDER", "")
	t.Setenv("AI_AGENT_MODEL", "")
	t.Setenv("AI_AGENT_API_KEY", "")
	t.Setenv("AI_AGENT_BASE_URL", "")

	got := LoadConfigFromEnv()
	if got.Provider != DefaultProvider {
		t.Fatalf("Provider = %q, want %q", got.Provider, DefaultProvider)
	}
	if got.Model != DefaultModel {
		t.Fatalf("Model = %q, want %q", got.Model, DefaultModel)
	}
}

func TestLoadConfigFromEnvSupportsGeminiFallbackNames(t *testing.T) {
	t.Setenv("AI_AGENT_PROVIDER", "")
	t.Setenv("AI_AGENT_MODEL", "")
	t.Setenv("AI_AGENT_API_KEY", "")
	t.Setenv("AI_AGENT_BASE_URL", "")
	t.Setenv("GEMINI_API_KEY", "gemini-key")
	t.Setenv("GEMINI_MODEL", "gemini-model")
	t.Setenv("GEMINI_BASE_URL", "https://example.test/v1beta")

	got := LoadConfigFromEnv()
	if got.Provider != "gemini" {
		t.Fatalf("Provider = %q, want gemini", got.Provider)
	}
	if got.APIKey != "gemini-key" {
		t.Fatalf("APIKey = %q, want gemini-key", got.APIKey)
	}
	if got.Model != "gemini-model" {
		t.Fatalf("Model = %q, want gemini-model", got.Model)
	}
	if got.BaseURL != "https://example.test/v1beta" {
		t.Fatalf("BaseURL = %q, want fallback base URL", got.BaseURL)
	}
}

func TestNewProviderFromConfigSelectsMock(t *testing.T) {
	provider, err := NewProviderFromConfig(Config{Provider: "mock"})
	if err != nil {
		t.Fatalf("NewProviderFromConfig returned error: %v", err)
	}
	if _, ok := provider.(MockProvider); !ok {
		t.Fatalf("provider type = %T, want MockProvider", provider)
	}
}

func TestNewProviderFromConfigRequiresGeminiAPIKey(t *testing.T) {
	_, err := NewProviderFromConfig(Config{Provider: "gemini"})
	if err == nil || !strings.Contains(err.Error(), "API_KEY") {
		t.Fatalf("error = %v, want API key error", err)
	}
}

func TestNewProviderFromConfigSelectsGemini(t *testing.T) {
	provider, err := NewProviderFromConfig(Config{Provider: "gemini", APIKey: "test-key"})
	if err != nil {
		t.Fatalf("NewProviderFromConfig returned error: %v", err)
	}
	if _, ok := provider.(*GeminiProvider); !ok {
		t.Fatalf("provider type = %T, want *GeminiProvider", provider)
	}
}

func TestNewProviderFromConfigRejectsUnknownProvider(t *testing.T) {
	_, err := NewProviderFromConfig(Config{Provider: "unknown"})
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("error = %v, want unknown provider error", err)
	}
}
