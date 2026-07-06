package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GeminiConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

type GeminiProvider struct {
	config GeminiConfig
	client *http.Client
}

func NewGeminiProvider(config GeminiConfig) *GeminiProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	return &GeminiProvider{
		config: config,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *GeminiProvider) Chat(ctx context.Context, messages []Message) (Message, error) {
	if p.config.APIKey == "" {
		return Message{}, fmt.Errorf("gemini api key is required")
	}
	if p.config.Model == "" {
		return Message{}, fmt.Errorf("gemini model is required")
	}

	payload, err := json.Marshal(geminiRequest{
		Contents: geminiContents(messages),
	})
	if err != nil {
		return Message{}, err
	}

	model := strings.TrimPrefix(p.config.Model, "models/")
	endpoint := strings.TrimRight(p.config.BaseURL, "/") + "/models/" + url.PathEscape(model) + ":generateContent?key=" + url.QueryEscape(p.config.APIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Message{}, fmt.Errorf("gemini request failed: %s: %s", resp.Status, readSnippet(resp.Body))
	}

	var result geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Message{}, err
	}
	if len(result.Candidates) == 0 {
		return Message{}, fmt.Errorf("gemini response had no candidates")
	}

	text := geminiText(result.Candidates[0].Content.Parts)
	if text == "" {
		return Message{}, fmt.Errorf("gemini response had no text")
	}

	return Message{Role: "assistant", Content: text}, nil
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func geminiContents(messages []Message) []geminiContent {
	contents := make([]geminiContent, 0, len(messages))
	for _, message := range messages {
		text := strings.TrimSpace(message.Content)
		if text == "" {
			continue
		}

		role := "user"
		if message.Role == "assistant" {
			role = "model"
		}

		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: text}},
		})
	}

	return contents
}

func geminiText(parts []geminiPart) string {
	var builder strings.Builder
	for _, part := range parts {
		builder.WriteString(part.Text)
	}

	return builder.String()
}

func readSnippet(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, 4096))
	if err != nil {
		return "could not read response body"
	}
	return string(data)
}
