package llm

import "fmt"

type ProviderConfig struct {
	APIKey    string
	BaseURL   string
	Model     string
	MaxTokens int
}

type ProviderFactory func(ProviderConfig) (Provider, error)

type Registry struct {
	factories map[string]ProviderFactory
}

func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]ProviderFactory)}
}

func DefaultRegistry() *Registry {
	registry := NewRegistry()
	_ = registry.Register("openai", func(config ProviderConfig) (Provider, error) {
		return NewOpenAIProvider(OpenAIConfig(config)), nil
	})
	_ = registry.Register("gemini", func(config ProviderConfig) (Provider, error) {
		return NewGeminiProvider(GeminiConfig(config)), nil
	})
	_ = registry.Register("anthropic", func(config ProviderConfig) (Provider, error) {
		return NewAnthropicProvider(AnthropicConfig(config)), nil
	})
	_ = registry.Register("ollama", func(config ProviderConfig) (Provider, error) {
		return NewOllamaProvider(OllamaConfig(config)), nil
	})
	return registry
}

func (r *Registry) Register(name string, factory ProviderFactory) error {
	if r == nil {
		return fmt.Errorf("provider registry is required")
	}
	if name == "" {
		return fmt.Errorf("provider name is required")
	}
	if factory == nil {
		return fmt.Errorf("provider factory for %q is required", name)
	}
	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("provider %q already registered", name)
	}
	r.factories[name] = factory
	return nil
}

func (r *Registry) New(name string, config ProviderConfig) (Provider, error) {
	if r == nil {
		return nil, fmt.Errorf("provider registry is required")
	}
	factory, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("provider %q is not registered", name)
	}
	return factory(config)
}
