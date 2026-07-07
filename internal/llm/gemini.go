package llm

import (
	"bufio"
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
	return p.chat(ctx, messages, nil)
}

func (p *GeminiProvider) ChatWithTools(ctx context.Context, messages []Message, tools []ToolDefinition) (Message, error) {
	return p.chat(ctx, messages, tools)
}

func (p *GeminiProvider) Stream(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	if p.config.APIKey == "" {
		return nil, fmt.Errorf("gemini api key is required")
	}
	if p.config.Model == "" {
		return nil, fmt.Errorf("gemini model is required")
	}

	payload, err := json.Marshal(geminiRequest{
		Contents: geminiContents(messages),
	})
	if err != nil {
		return nil, err
	}

	model := strings.TrimPrefix(p.config.Model, "models/")
	endpoint := strings.TrimRight(p.config.BaseURL, "/") + "/models/" + url.PathEscape(model) + ":streamGenerateContent?alt=sse&key=" + url.QueryEscape(p.config.APIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, fmt.Errorf("gemini stream request failed: %s: %s", resp.Status, readSnippet(resp.Body))
	}

	chunks := make(chan StreamChunk)
	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		if err := streamGeminiResponse(ctx, resp.Body, chunks); err != nil {
			sendStreamChunk(ctx, chunks, StreamChunk{Error: err, Done: true})
		}
	}()

	return chunks, nil
}

func (p *GeminiProvider) chat(ctx context.Context, messages []Message, tools []ToolDefinition) (Message, error) {
	if p.config.APIKey == "" {
		return Message{}, fmt.Errorf("gemini api key is required")
	}
	if p.config.Model == "" {
		return Message{}, fmt.Errorf("gemini model is required")
	}

	payload, err := json.Marshal(geminiRequest{
		Contents: geminiContents(messages),
		Tools:    geminiTools(tools),
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
	Tools    []geminiTool    `json:"tools,omitempty"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

type geminiFunctionDeclaration struct {
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
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

func geminiTools(tools []ToolDefinition) []geminiTool {
	if len(tools) == 0 {
		return nil
	}

	declarations := make([]geminiFunctionDeclaration, 0, len(tools))
	for _, tool := range tools {
		if strings.TrimSpace(tool.Name) == "" {
			continue
		}
		declarations = append(declarations, geminiFunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		})
	}
	if len(declarations) == 0 {
		return nil
	}

	return []geminiTool{{FunctionDeclarations: declarations}}
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

func streamGeminiResponse(ctx context.Context, body io.Reader, chunks chan<- StreamChunk) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var data strings.Builder
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}

		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			done, err := dispatchGeminiSSE(ctx, data.String(), chunks)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
			data.Reset()
			continue
		}
		if strings.HasPrefix(line, "data:") {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	done, err := dispatchGeminiSSE(ctx, data.String(), chunks)
	if err != nil {
		return err
	}
	if !done {
		sendStreamChunk(ctx, chunks, StreamChunk{Done: true})
	}
	return nil
}

func dispatchGeminiSSE(ctx context.Context, data string, chunks chan<- StreamChunk) (bool, error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return false, nil
	}
	if data == "[DONE]" {
		sendStreamChunk(ctx, chunks, StreamChunk{Done: true})
		return true, nil
	}

	var result geminiResponse
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return false, err
	}
	if len(result.Candidates) == 0 {
		return false, nil
	}

	message := result.Candidates[0].Content.toMessage()
	if message.Content == "" {
		return false, nil
	}
	sendStreamChunk(ctx, chunks, StreamChunk{Content: message.Content})
	return false, nil
}
