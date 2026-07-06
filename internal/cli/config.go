package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/MehulCodr/AI-agent/internal/llm"
)

const defaultGeminiModel = "gemini-3.5-flash"

type runtimeConfig struct {
	GeminiAPIKey  string
	GeminiBaseURL string
	Model         string
}

type localConfig struct {
	Model string `json:"model"`
}

func newProvider() (llm.Provider, error) {
	config, err := loadRuntimeConfig()
	if err != nil {
		return nil, err
	}

	if config.GeminiAPIKey == "" {
		return llm.MockProvider{}, nil
	}

	return llm.NewGeminiProvider(llm.GeminiConfig{
		APIKey:  config.GeminiAPIKey,
		BaseURL: config.GeminiBaseURL,
		Model:   config.Model,
	}), nil
}

func loadRuntimeConfig() (runtimeConfig, error) {
	if err := loadDotEnv(".env"); err != nil {
		return runtimeConfig{}, err
	}

	local, err := loadLocalConfig(".agent/config.json")
	if err != nil {
		return runtimeConfig{}, err
	}

	model := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	if model == "" {
		model = strings.TrimSpace(local.Model)
	}
	if model == "" {
		model = defaultGeminiModel
	}

	return runtimeConfig{
		GeminiAPIKey:  strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
		GeminiBaseURL: strings.TrimSpace(os.Getenv("GEMINI_BASE_URL")),
		Model:         model,
	}, nil
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
