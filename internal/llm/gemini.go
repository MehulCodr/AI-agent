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
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

type GeminiProvider struct {
	config GeminiConfig
	client *http.Client
}

func NewGeminiProvider(config GeminiConfig) *GeminiProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	if config.Model == "" {
		config.Model = DefaultModel
	}

	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	return &GeminiProvider{
		config: config,
		client: client,
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

	message := result.Candidates[0].Content.toMessage()
	if message.Content == "" && len(message.ToolCalls) == 0 {
		return Message{}, fmt.Errorf("gemini response had no content")
	}
	return message, nil
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text         string              `json:"text,omitempty"`
	FunctionCall *geminiFunctionCall `json:"functionCall,omitempty"`
}

type geminiFunctionCall struct {
	Name string         `json:"name,omitempty"`
	Args map[string]any `json:"args,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func geminiContents(messages []Message) []geminiContent {
	contents := make([]geminiContent, 0, len(messages))
	for _, message := range messages {
		content := geminiContent{
			Role:  geminiRole(message.Role),
			Parts: geminiParts(message),
		}
		if len(content.Parts) == 0 {
			continue
		}
		contents = append(contents, content)
	}
	return contents
}

func geminiRole(role string) string {
	if role == "assistant" {
		return "model"
	}
	return "user"
}

func geminiParts(message Message) []geminiPart {
	parts := make([]geminiPart, 0, 1+len(message.ToolCalls))
	if text := strings.TrimSpace(message.Content); text != "" {
		parts = append(parts, geminiPart{Text: text})
	}
	for _, call := range message.ToolCalls {
		if call.Function.Name == "" {
			continue
		}
		parts = append(parts, geminiPart{
			FunctionCall: &geminiFunctionCall{
				Name: call.Function.Name,
				Args: parseToolArguments(call.Function.Arguments),
			},
		})
	}
	return parts
}

func (c geminiContent) toMessage() Message {
	message := Message{Role: "assistant"}
	var text strings.Builder
	for _, part := range c.Parts {
		text.WriteString(part.Text)
		if part.FunctionCall != nil {
			message.ToolCalls = append(message.ToolCalls, ToolCall{
				Type: "function",
				Function: ToolCallFunction{
					Name:      part.FunctionCall.Name,
					Arguments: encodeToolArguments(part.FunctionCall.Args),
				},
			})
		}
	}
	message.Content = text.String()
	return message
}

func parseToolArguments(arguments string) map[string]any {
	if strings.TrimSpace(arguments) == "" {
		return nil
	}
	var values map[string]any
	if err := json.Unmarshal([]byte(arguments), &values); err != nil {
		return map[string]any{"raw": arguments}
	}
	return values
}

func encodeToolArguments(arguments map[string]any) string {
	if len(arguments) == 0 {
		return "{}"
	}
	data, err := json.Marshal(arguments)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func readSnippet(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, 4096))
	if err != nil {
		return "could not read response body"
	}
	return string(data)
}
