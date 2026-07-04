package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAIConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

type OpenAIProvider struct {
	config OpenAIConfig
	client *http.Client
}

func NewOpenAIProvider(config OpenAIConfig) *OpenAIProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}

	return &OpenAIProvider{
		config: config,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message) (Message, error) {
	if p.config.APIKey == "" {
		return Message{}, fmt.Errorf("openai api key is required")
	}
	if p.config.Model == "" {
		return Message{}, fmt.Errorf("openai model is required")
	}

	payload, err := json.Marshal(chatRequest{
		Model:    p.config.Model,
		Messages: messages,
	})
	if err != nil {
		return Message{}, err
	}

	url := strings.TrimRight(p.config.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Message{}, fmt.Errorf("openai request failed: %s: %s", resp.Status, readSnippet(resp.Body))
	}

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Message{}, err
	}
	if len(result.Choices) == 0 {
		return Message{}, fmt.Errorf("openai response had no choices")
	}

	return result.Choices[0].Message, nil
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func readSnippet(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, 4096))
	if err != nil {
		return "could not read response body"
	}
	return string(data)
}
