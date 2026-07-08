package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type OpenAIConfig ProviderConfig

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
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) Chat(ctx context.Context, request ChatRequest) (Message, error) {
	if err := p.validate(request); err != nil {
		return Message{}, err
	}

	payload := openAIRequest{
		Model:    p.model(request),
		Messages: openAIMessages(request.Messages),
		Tools:    openAITools(request.Tools),
	}

	var result openAIResponse
	if err := postJSON(ctx, p.client, p.endpoint(), p.headers(), payload, &result, "openai"); err != nil {
		return Message{}, err
	}
	if len(result.Choices) == 0 {
		return Message{}, fmt.Errorf("openai response had no choices")
	}
	return result.Choices[0].Message.toMessage(), nil
}

func (p *OpenAIProvider) Stream(ctx context.Context, request ChatRequest, onEvent StreamHandler) (Message, error) {
	if err := p.validate(request); err != nil {
		return Message{}, err
	}

	payload := openAIRequest{
		Model:    p.model(request),
		Messages: openAIMessages(request.Messages),
		Tools:    openAITools(request.Tools),
		Stream:   true,
	}

	var content strings.Builder
	callParts := map[int]*openAIStreamToolCall{}
	err := postStream(ctx, p.client, p.endpoint(), p.headers(), payload, "openai", func(data string) error {
		if data == "[DONE]" {
			return nil
		}
		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return err
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				content.WriteString(choice.Delta.Content)
				if err := emitDelta(onEvent, choice.Delta.Content); err != nil {
					return err
				}
			}
			for _, toolCall := range choice.Delta.ToolCalls {
				part := callParts[toolCall.Index]
				if part == nil {
					part = &openAIStreamToolCall{Index: toolCall.Index}
					callParts[toolCall.Index] = part
				}
				if toolCall.ID != "" {
					part.ID = toolCall.ID
				}
				if toolCall.Type != "" {
					part.Type = toolCall.Type
				}
				if toolCall.Function.Name != "" {
					part.Function.Name += toolCall.Function.Name
				}
				if toolCall.Function.Arguments != "" {
					part.Function.Arguments += toolCall.Function.Arguments
				}
			}
		}
		return nil
	})
	if err != nil {
		return Message{}, err
	}

	return Message{Role: RoleAssistant, Content: content.String(), ToolCalls: openAIStreamCalls(callParts)}, nil
}

func (p *OpenAIProvider) validate(request ChatRequest) error {
	if p.config.APIKey == "" {
		return fmt.Errorf("openai api key is required")
	}
	if request.Model == "" && p.config.Model == "" {
		return fmt.Errorf("openai model is required")
	}
	return nil
}

func (p *OpenAIProvider) model(request ChatRequest) string {
	if request.Model != "" {
		return request.Model
	}
	return p.config.Model
}

func (p *OpenAIProvider) endpoint() string {
	return strings.TrimRight(p.config.BaseURL, "/") + "/chat/completions"
}

func (p *OpenAIProvider) headers() map[string]string {
	return map[string]string{"Authorization": "Bearer " + p.config.APIKey}
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Tools    []openAITool    `json:"tools,omitempty"`
	Stream   bool            `json:"stream,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIToolCall struct {
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta openAIStreamDelta `json:"delta"`
	} `json:"choices"`
}

type openAIStreamDelta struct {
	Content   string                 `json:"content,omitempty"`
	ToolCalls []openAIStreamToolCall `json:"tool_calls,omitempty"`
}

type openAIStreamToolCall struct {
	Index    int                `json:"index"`
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function openAIFunctionCall `json:"function,omitempty"`
}

func openAIMessages(messages []Message) []openAIMessage {
	converted := make([]openAIMessage, 0, len(messages))
	for _, message := range messages {
		converted = append(converted, openAIMessage{
			Role:       message.Role,
			Content:    message.Content,
			Name:       message.Name,
			ToolCallID: message.ToolCallID,
			ToolCalls:  openAIToolCalls(message.ToolCalls),
		})
	}
	return converted
}

func openAIToolCalls(calls []ToolCall) []openAIToolCall {
	if len(calls) == 0 {
		return nil
	}
	converted := make([]openAIToolCall, 0, len(calls))
	for _, call := range calls {
		callType := call.Type
		if callType == "" {
			callType = "function"
		}
		converted = append(converted, openAIToolCall{
			ID:   call.ID,
			Type: callType,
			Function: openAIFunctionCall{
				Name:      call.Function.Name,
				Arguments: call.Function.Arguments,
			},
		})
	}
	return converted
}

func openAITools(tools []ToolDefinition) []openAITool {
	if len(tools) == 0 {
		return nil
	}
	converted := make([]openAITool, 0, len(tools))
	for _, tool := range tools {
		converted = append(converted, openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}
	return converted
}

func (m openAIMessage) toMessage() Message {
	return Message{
		Role:      RoleAssistant,
		Content:   m.Content,
		ToolCalls: openAIMessageToolCalls(m.ToolCalls),
	}
}

func openAIMessageToolCalls(calls []openAIToolCall) []ToolCall {
	if len(calls) == 0 {
		return nil
	}
	converted := make([]ToolCall, 0, len(calls))
	for _, call := range calls {
		callType := call.Type
		if callType == "" {
			callType = "function"
		}
		converted = append(converted, ToolCall{
			ID:   call.ID,
			Type: callType,
			Function: ToolCallFunction{
				Name:      call.Function.Name,
				Arguments: call.Function.Arguments,
			},
		})
	}
	return converted
}

func openAIStreamCalls(parts map[int]*openAIStreamToolCall) []ToolCall {
	if len(parts) == 0 {
		return nil
	}
	calls := make([]ToolCall, 0, len(parts))
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if part == nil {
			continue
		}
		callType := part.Type
		if callType == "" {
			callType = "function"
		}
		calls = append(calls, ToolCall{
			ID:   part.ID,
			Type: callType,
			Function: ToolCallFunction{
				Name:      part.Function.Name,
				Arguments: part.Function.Arguments,
			},
		})
	}
	return calls
}
