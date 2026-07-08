package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MehulCodr/AI-agent/internal/llm"
)

const (
	defaultProvider = "gemini"
	defaultSession  = "default"
	defaultRedis    = "localhost:6379"
)

type runtimeConfig struct {
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	Session          string `json:"session"`
	MaxSteps         int    `json:"max_steps"`
	Stream           bool   `json:"stream"`
	OpenAIAPIKey     string `json:"-"`
	OpenAIBaseURL    string `json:"openai_base_url"`
	GeminiAPIKey     string `json:"-"`
	GeminiBaseURL    string `json:"gemini_base_url"`
	AnthropicAPIKey  string `json:"-"`
	AnthropicBaseURL string `json:"anthropic_base_url"`
	OllamaBaseURL    string `json:"ollama_base_url"`
	RedisAddr        string `json:"redis_addr"`
	RedisPassword    string `json:"-"`
	RedisDB          int    `json:"redis_db"`
}

type localConfig struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Session   string `json:"session"`
	MaxSteps  int    `json:"max_steps"`
	Stream    *bool  `json:"stream,omitempty"`
	RedisAddr string `json:"redis_addr"`
	RedisDB   int    `json:"redis_db"`
}

type commandOptions struct {
	Provider string
	Model    string
	Session  string
	Stream   *bool
}

func newProvider(config runtimeConfig) (llm.Provider, error) {
	providerName := strings.ToLower(strings.TrimSpace(config.Provider))
	if providerName == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model is required for provider %q", providerName)
	}

	providerConfig := llm.ProviderConfig{
		Model:     config.Model,
		MaxTokens: 4096,
	}

	switch providerName {
	case "openai":
		if strings.TrimSpace(config.OpenAIAPIKey) == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY is required for provider openai")
		}
		providerConfig.APIKey = config.OpenAIAPIKey
		providerConfig.BaseURL = config.OpenAIBaseURL
	case "gemini":
		if strings.TrimSpace(config.GeminiAPIKey) == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY is required for provider gemini")
		}
		providerConfig.APIKey = config.GeminiAPIKey
		providerConfig.BaseURL = config.GeminiBaseURL
	case "anthropic":
		if strings.TrimSpace(config.AnthropicAPIKey) == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY is required for provider anthropic")
		}
		providerConfig.APIKey = config.AnthropicAPIKey
		providerConfig.BaseURL = config.AnthropicBaseURL
	case "ollama":
		providerConfig.BaseURL = config.OllamaBaseURL
	default:
		return nil, fmt.Errorf("unknown provider %q", providerName)
	}

	return llm.DefaultRegistry().New(providerName, providerConfig)
}

func loadRuntimeConfig(options commandOptions) (runtimeConfig, error) {
	if err := loadDotEnv(".env"); err != nil {
		return runtimeConfig{}, err
	}

	local, err := loadLocalConfig(".agent/config.json")
	if err != nil {
		return runtimeConfig{}, err
	}

	config := runtimeConfig{
		Provider:         firstNonEmpty(os.Getenv("AI_AGENT_PROVIDER"), local.Provider, defaultProvider),
		Model:            firstNonEmpty(os.Getenv("AI_AGENT_MODEL"), local.Model),
		Session:          firstNonEmpty(os.Getenv("AI_AGENT_SESSION"), local.Session, defaultSession),
		MaxSteps:         firstPositiveInt(envInt("AI_AGENT_MAX_STEPS"), local.MaxSteps, defaultMaxSteps),
		Stream:           firstBool(envBool("AI_AGENT_STREAM"), local.Stream, true),
		OpenAIAPIKey:     strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		OpenAIBaseURL:    strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")),
		GeminiAPIKey:     strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
		GeminiBaseURL:    strings.TrimSpace(os.Getenv("GEMINI_BASE_URL")),
		AnthropicAPIKey:  strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")),
		AnthropicBaseURL: strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL")),
		OllamaBaseURL:    strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL")),
		RedisAddr:        firstNonEmpty(os.Getenv("REDIS_ADDR"), local.RedisAddr, defaultRedis),
		RedisPassword:    strings.TrimSpace(os.Getenv("REDIS_PASSWORD")),
		RedisDB:          firstPositiveOrZeroInt(envInt("REDIS_DB"), local.RedisDB, 0),
	}

	if options.Provider != "" {
		config.Provider = options.Provider
	}
	if options.Model != "" {
		config.Model = options.Model
	}
	if options.Session != "" {
		config.Session = options.Session
	}
	if options.Stream != nil {
		config.Stream = *options.Stream
	}

	config.Provider = strings.ToLower(strings.TrimSpace(config.Provider))
	config.Model = strings.TrimSpace(config.Model)
	config.Session = safeSessionName(config.Session)
	return config, nil
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("parse %s:%d: expected KEY=value", path, lineNumber)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("parse %s:%d: key is required", path, lineNumber)
		}

		value = trimEnvValue(value)
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set %s from %s: %w", key, path, err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	return nil
}

func trimEnvValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return value
	}

	first := value[0]
	last := value[len(value)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		return value[1 : len(value)-1]
	}

	return value
}

func loadLocalConfig(path string) (localConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return localConfig{}, nil
		}
		return localConfig{}, fmt.Errorf("read %s: %w", path, err)
	}

	var config localConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return localConfig{}, fmt.Errorf("parse %s: %w", path, err)
	}

	return config, nil
}

func writeDefaultConfig(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create %s directory: %w", filepath.Dir(path), err)
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect %s: %w", path, err)
	}

	stream := true
	config := localConfig{
		Provider:  defaultProvider,
		Model:     "",
		Session:   defaultSession,
		MaxSteps:  defaultMaxSteps,
		Stream:    &stream,
		RedisAddr: defaultRedis,
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("create default config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func ensureDotEnv(path string) error {
	existing := ""
	if data, err := os.ReadFile(path); err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}

	lines := []string{
		"AI_AGENT_PROVIDER=gemini",
		"AI_AGENT_MODEL=",
		"AI_AGENT_SESSION=default",
		"AI_AGENT_STREAM=true",
		"AI_AGENT_MAX_STEPS=8",
		"OPENAI_API_KEY=",
		"OPENAI_BASE_URL=https://api.openai.com/v1",
		"GEMINI_API_KEY=",
		"GEMINI_BASE_URL=https://generativelanguage.googleapis.com/v1beta",
		"ANTHROPIC_API_KEY=",
		"ANTHROPIC_BASE_URL=https://api.anthropic.com/v1",
		"OLLAMA_BASE_URL=http://localhost:11434",
		"REDIS_ADDR=localhost:6379",
		"REDIS_PASSWORD=",
		"REDIS_DB=0",
	}

	var builder strings.Builder
	builder.WriteString(existing)
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		builder.WriteString("\n")
	}
	for _, line := range lines {
		key, _, _ := strings.Cut(line, "=")
		if envFileHasKey(existing, key) {
			continue
		}
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	if builder.String() == existing {
		return nil
	}
	return os.WriteFile(path, []byte(builder.String()), 0600)
}

func envFileHasKey(content, key string) bool {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		left, _, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(left) == key {
			return true
		}
	}
	return false
}

func envInt(key string) *int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return nil
	}
	return &n
}

func envBool(key string) *bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil
	}
	return &parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstPositiveInt(values ...any) int {
	for _, value := range values {
		switch typed := value.(type) {
		case *int:
			if typed != nil && *typed > 0 {
				return *typed
			}
		case int:
			if typed > 0 {
				return typed
			}
		}
	}
	return 0
}

func firstPositiveOrZeroInt(values ...any) int {
	for _, value := range values {
		switch typed := value.(type) {
		case *int:
			if typed != nil && *typed >= 0 {
				return *typed
			}
		case int:
			if typed >= 0 {
				return typed
			}
		}
	}
	return 0
}

func firstBool(env *bool, local *bool, fallback bool) bool {
	if env != nil {
		return *env
	}
	if local != nil {
		return *local
	}
	return fallback
}

func safeSessionName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return defaultSession
	}
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "..", "_")
	return name
}
