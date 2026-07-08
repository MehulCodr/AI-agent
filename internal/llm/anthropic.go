package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultAnthropicMaxTokens = 4096

type AnthropicConfig ProviderConfig

type AnthropicProvider struct {
	config AnthropicConfig
	client *http.Client
}

func NewAnthropicProvider(config AnthropicConfig) *AnthropicProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com/v1"
	}
	if config.MaxTokens <= 0 {
		config.MaxTokens = defaultAnthropicMaxTokens
	}

	return &AnthropicProvider{
		config: config,
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) Chat(ctx context.Context, request ChatRequest) (Message, error) {
	if err := p.validate(request); err != nil {
		return Message{}, err
	}

	payload := p.requestPayload(request, false)
	var result anthropicResponse
	if err := postJSON(ctx, p.client, p.endpoint(), p.headers(), payload, &result, "anthropic"); err != nil {
		return Message{}, err
	}
	return anthropicResponseMessage(result)
}

func (p *AnthropicProvider) Stream(ctx context.Context, request ChatRequest, onEvent StreamHandler) (Message, error) {
	if err := p.validate(request); err != nil {
		return Message{}, err
	}

	payload := p.requestPayload(request, true)
	var content strings.Builder
	blocks := map[int]*anthropicStreamBlock{}

	err := postStream(ctx, p.client, p.endpoint(), p.headers(), payload, "anthropic", func(data string) error {
		var event anthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return err
		}
		switch event.Type {
		case "content_block_start":
			block := &anthropicStreamBlock{Kind: event.ContentBlock.Type}
			if event.ContentBlock.ID != "" {
				block.ID = event.ContentBlock.ID
			}
			if event.ContentBlock.Name != "" {
				block.Name = event.ContentBlock.Name
			}
			blocks[event.Index] = block
		case "content_block_delta":
			block := blocks[event.Index]
			if block == nil {
				block = &anthropicStreamBlock{}
				blocks[event.Index] = block
			}
			if event.Delta.Type == "text_delta" && event.Delta.Text != "" {
				content.WriteString(event.Delta.Text)
				if err := emitDelta(onEvent, event.Delta.Text); err != nil {
					return err
				}
			}
			if event.Delta.Type == "input_json_delta" {
				block.InputJSON += event.Delta.PartialJSON
			}
		}
		return nil
	})
	if err != nil {
		return Message{}, err
	}

	return Message{Role: RoleAssistant, Content: content.String(), ToolCalls: anthropicStreamCalls(blocks)}, nil
}

func (p *AnthropicProvider) validate(request ChatRequest) error {
	if p.config.APIKey == "" {
		return fmt.Errorf("anthropic api key is required")
	}
	if request.Model == "" && p.config.Model == "" {
		return fmt.Errorf("anthropic model is required")
	}
	return nil
}

func (p *AnthropicProvider) endpoint() string {
	return strings.TrimRight(p.config.BaseURL, "/") + "/messages"
}

func (p *AnthropicProvider) headers() map[string]string {
	return map[string]string{
		"x-api-key":         p.config.APIKey,
		"anthropic-version": "2023-06-01",
	}
}

func (p *AnthropicProvider) model(request ChatRequest) string {
	if request.Model != "" {
		return request.Model
	}
	return p.config.Model
}

func (p *AnthropicProvider) requestPayload(request ChatRequest, stream bool) anthropicRequest {
	system, messages := anthropicMessages(request.Messages)
	return anthropicRequest{
		Model:     p.model(request),
		MaxTokens: p.config.MaxTokens,
		System:    system,
		Messages:  messages,
		Tools:     anthropicTools(request.Tools),
		Stream:    stream,
	}
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
	Stream    bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	ToolUseID string         `json:"tool_use_id,omitempty"`
	Content   string         `json:"content,omitempty"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicResponse struct {
	Content []anthropicContent `json:"content"`
}

type anthropicStreamEvent struct {
	Type         string `json:"type"`
	Index        int    `json:"index"`
	ContentBlock struct {
		Type string `json:"type"`
		ID   string `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"content_block"`
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text,omitempty"`
		PartialJSON string `json:"partial_json,omitempty"`
	} `json:"delta"`
}

type anthropicStreamBlock struct {
	Kind      string
	ID        string
	Name      string
	InputJSON string
}

func anthropicMessages(messages []Message) (string, []anthropicMessage) {
	var system strings.Builder
	converted := make([]anthropicMessage, 0, len(messages))

	for _, message := range messages {
		if message.Role == RoleSystem {
			if text := strings.TrimSpace(message.Content); text != "" {
				if system.Len() > 0 {
					system.WriteString("\n\n")
				}
				system.WriteString(text)
			}
			continue
		}

		content := anthropicContentBlocks(message)
		if len(content) == 0 {
			continue
		}

		role := message.Role
		if role == RoleTool {
			role = RoleUser
		}
		converted = append(converted, anthropicMessage{Role: role, Content: content})
	}

	return system.String(), converted
}

func anthropicContentBlocks(message Message) []anthropicContent {
	blocks := make([]anthropicContent, 0, 1+len(message.ToolCalls))
	if message.Role == RoleTool {
		blocks = append(blocks, anthropicContent{
			Type:      "tool_result",
			ToolUseID: message.ToolCallID,
			Content:   message.Content,
		})
		return blocks
	}

	if message.Content != "" {
		blocks = append(blocks, anthropicContent{Type: "text", Text: message.Content})
	}
	for _, call := range message.ToolCalls {
		args := map[string]any{}
		if strings.TrimSpace(call.Function.Arguments) != "" {
			_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
		}
		blocks = append(blocks, anthropicContent{
			Type:  "tool_use",
			ID:    call.ID,
			Name:  call.Function.Name,
			Input: args,
		})
	}
	return blocks
}

func anthropicTools(tools []ToolDefinition) []anthropicTool {
	if len(tools) == 0 {
		return nil
	}
	converted := make([]anthropicTool, 0, len(tools))
	for _, tool := range tools {
		converted = append(converted, anthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.Parameters,
		})
	}
	return converted
}

func anthropicResponseMessage(response anthropicResponse) (Message, error) {
	var text strings.Builder
	var calls []ToolCall
	for _, block := range response.Content {
		switch block.Type {
		case "text":
			text.WriteString(block.Text)
		case "tool_use":
			args, err := json.Marshal(block.Input)
			if err != nil {
				return Message{}, err
			}
			calls = append(calls, ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: ToolCallFunction{
					Name:      block.Name,
					Arguments: string(args),
				},
			})
		}
	}
	if text.Len() == 0 && len(calls) == 0 {
		return Message{}, fmt.Errorf("anthropic response had no text or tool calls")
	}
	return Message{Role: RoleAssistant, Content: text.String(), ToolCalls: calls}, nil
}

func anthropicStreamCalls(blocks map[int]*anthropicStreamBlock) []ToolCall {
	if len(blocks) == 0 {
		return nil
	}
	calls := make([]ToolCall, 0, len(blocks))
	for i := 0; i < len(blocks); i++ {
		block := blocks[i]
		if block == nil || block.Kind != "tool_use" {
			continue
		}
		args := strings.TrimSpace(block.InputJSON)
		if args == "" {
			args = "{}"
		}
		calls = append(calls, ToolCall{
			ID:   block.ID,
			Type: "function",
			Function: ToolCallFunction{
				Name:      block.Name,
				Arguments: args,
			},
		})
	}
	return calls
}
