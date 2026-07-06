package llm

import (
	"fmt"
	"os"
	"strings"
)

const (
	DefaultProvider = "mock"
	DefaultModel    = "gemini-3.5-flash"
)

type Config struct {
	Provider string
	Model    string
	APIKey   string
	BaseURL  string
}

func LoadConfigFromEnv() Config {
	provider := strings.TrimSpace(os.Getenv("AI_AGENT_PROVIDER"))
	if provider == "" {
		provider = DefaultProvider
		if firstEnv("AI_AGENT_API_KEY", "GEMINI_API_KEY") != "" {
			provider = "gemini"
		}
	}

	model := firstEnv("AI_AGENT_MODEL", "GEMINI_MODEL")
	if model == "" {
		model = DefaultModel
	}

	return Config{
		Provider: strings.ToLower(provider),
		Model:    model,
		APIKey:   firstEnv("AI_AGENT_API_KEY", "GEMINI_API_KEY"),
		BaseURL:  firstEnv("AI_AGENT_BASE_URL", "GEMINI_BASE_URL"),
	}
}

func NewProviderFromConfig(cfg Config) (Provider, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		provider = DefaultProvider
	}

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = DefaultModel
	}

	switch provider {
	case "mock":
		return MockProvider{}, nil
	case "gemini":
		if strings.TrimSpace(cfg.APIKey) == "" {
			return nil, fmt.Errorf("gemini provider requires AI_AGENT_API_KEY or GEMINI_API_KEY")
		}
		return NewGeminiProvider(GeminiConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   model,
		}), nil
	default:
		return nil, fmt.Errorf("unknown llm provider %q", cfg.Provider)
	}
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}
