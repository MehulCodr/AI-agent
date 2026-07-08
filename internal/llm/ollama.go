package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type OllamaConfig ProviderConfig

type OllamaProvider struct {
	config OllamaConfig
	client *http.Client
}

func NewOllamaProvider(config OllamaConfig) *OllamaProvider {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434"
	}

	return &OllamaProvider{
		config: config,
		client: &http.Client{Timeout: 0},
	}
}

func (p *OllamaProvider) Name() string {
	return "ollama"
}

func (p *OllamaProvider) Chat(ctx context.Context, request ChatRequest) (Message, error) {
	if err := p.validate(request); err != nil {
		return Message{}, err
	}

	payload := ollamaRequest{
		Model:    p.model(request),
		Messages: ollamaMessages(request.Messages),
		Tools:    ollamaTools(request.Tools),
		Stream:   false,
	}

	var result ollamaResponse
	if err := postJSON(ctx, p.client, p.endpoint(), nil, payload, &result, "ollama"); err != nil {
		return Message{}, err
	}
	return result.Message.toMessage(), nil
}

func (p *OllamaProvider) Stream(ctx context.Context, request ChatRequest, onEvent StreamHandler) (Message, error) {
	if err := p.validate(request); err != nil {
		return Message{}, err
	}

	payload := ollamaRequest{
		Model:    p.model(request),
		Messages: ollamaMessages(request.Messages),
		Tools:    ollamaTools(request.Tools),
		Stream:   true,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint(), bytes.NewReader(data))
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
		return Message{}, fmt.Errorf("ollama stream failed: %s: %s", resp.Status, readSnippet(resp.Body))
	}

	var content strings.Builder
	var calls []ToolCall
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var chunk ollamaResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return Message{}, err
		}
		message := chunk.Message.toMessage()
		if message.Content != "" {
			content.WriteString(message.Content)
			if err := emitDelta(onEvent, message.Content); err != nil {
				return Message{}, err
			}
		}
		calls = append(calls, message.ToolCalls...)
	}
	if err := scanner.Err(); err != nil {
		return Message{}, err
	}

	return Message{Role: RoleAssistant, Content: content.String(), ToolCalls: calls}, nil
}

func (p *OllamaProvider) validate(request ChatRequest) error {
	if request.Model == "" && p.config.Model == "" {
		return fmt.Errorf("ollama model is required")
	}
	return nil
}

func (p *OllamaProvider) model(request ChatRequest) string {
	if request.Model != "" {
		return request.Model
	}
	return p.config.Model
}

func (p *OllamaProvider) endpoint() string {
	return strings.TrimRight(p.config.BaseURL, "/") + "/api/chat"
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Tools    []ollamaTool    `json:"tools,omitempty"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content,omitempty"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

type ollamaToolCall struct {
	Function ollamaFunctionCall `json:"function"`
}

type ollamaFunctionCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type ollamaTool struct {
	Type     string         `json:"type"`
	Function ollamaFunction `json:"function"`
}

type ollamaFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
}

func ollamaMessages(messages []Message) []ollamaMessage {
	converted := make([]ollamaMessage, 0, len(messages))
	for _, message := range messages {
		role := message.Role
		content := message.Content
		if message.Role == RoleTool {
			role = RoleTool
			content = message.Content
		}
		converted = append(converted, ollamaMessage{
			Role:      role,
			Content:   content,
			ToolCalls: ollamaToolCalls(message.ToolCalls),
		})
	}
	return converted
}

func ollamaToolCalls(calls []ToolCall) []ollamaToolCall {
	if len(calls) == 0 {
		return nil
	}
	converted := make([]ollamaToolCall, 0, len(calls))
	for _, call := range calls {
		args := map[string]any{}
		if strings.TrimSpace(call.Function.Arguments) != "" {
			_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
		}
		converted = append(converted, ollamaToolCall{
			Function: ollamaFunctionCall{
				Name:      call.Function.Name,
				Arguments: args,
			},
		})
	}
	return converted
}

func ollamaTools(tools []ToolDefinition) []ollamaTool {
	if len(tools) == 0 {
		return nil
	}
	converted := make([]ollamaTool, 0, len(tools))
	for _, tool := range tools {
		converted = append(converted, ollamaTool{
			Type: "function",
			Function: ollamaFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}
	return converted
}

func (m ollamaMessage) toMessage() Message {
	return Message{
		Role:      RoleAssistant,
		Content:   m.Content,
		ToolCalls: ollamaMessageToolCalls(m.ToolCalls),
	}
}

func ollamaMessageToolCalls(calls []ollamaToolCall) []ToolCall {
	if len(calls) == 0 {
		return nil
	}
	converted := make([]ToolCall, 0, len(calls))
	for i, call := range calls {
		args, err := json.Marshal(call.Function.Arguments)
		if err != nil {
			args = []byte("{}")
		}
		converted = append(converted, ToolCall{
			ID:   fmt.Sprintf("ollama_call_%d", i),
			Type: "function",
			Function: ToolCallFunction{
				Name:      call.Function.Name,
				Arguments: string(args),
			},
		})
	}
	return converted
}
