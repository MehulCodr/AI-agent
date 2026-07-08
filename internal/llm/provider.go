package llm

import "context"

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function ToolCallFunction `json:"function,omitempty"`
}

type ToolCallFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ChatRequest struct {
	Messages []Message
	Tools    []ToolDefinition
	Model    string
}

type StreamEvent struct {
	Delta string
}

type StreamHandler func(StreamEvent) error

type Provider interface {
	Name() string
	Chat(ctx context.Context, request ChatRequest) (Message, error)
	Stream(ctx context.Context, request ChatRequest, onEvent StreamHandler) (Message, error)
}

func StreamViaChat(ctx context.Context, provider Provider, request ChatRequest, onEvent StreamHandler) (Message, error) {
	message, err := provider.Chat(ctx, request)
	if err != nil {
		return Message{}, err
	}
	if onEvent != nil && message.Content != "" {
		if err := onEvent(StreamEvent{Delta: message.Content}); err != nil {
			return Message{}, err
		}
	}
	return message, nil
}
