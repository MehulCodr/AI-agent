package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/MehulCodr/AI-agent/internal/llm"
)

func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	return a.run(ctx, input, false, nil)
}

func (a *Agent) RunStream(ctx context.Context, input string, onEvent EventHandler) (string, error) {
	return a.run(ctx, input, true, onEvent)
}

func (a *Agent) run(ctx context.Context, input string, stream bool, onEvent EventHandler) (string, error) {
	if err := a.validateRun(ctx, input); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	a.messages = append(a.messages, llm.Message{
		Role:    llm.RoleUser,
		Content: strings.TrimSpace(input),
	})

	for step := 0; step < a.maxSteps; step++ {
		request := llm.ChatRequest{
			Messages: a.requestMessages(),
			Tools:    a.toolDefinitions(),
		}

		response, err := a.chat(ctx, request, stream, onEvent)
		if err != nil {
			return "", err
		}
		if response.Role == "" {
			response.Role = llm.RoleAssistant
		}

		a.messages = append(a.messages, response)
		if len(response.ToolCalls) == 0 {
			if err := a.saveMemory(ctx); err != nil {
				return "", err
			}
			return response.Content, nil
		}

		for i, call := range response.ToolCalls {
			if call.ID == "" {
				call.ID = fmt.Sprintf("call_%d_%d", step, i)
			}
			result := a.executeTool(ctx, call, onEvent)
			a.messages = append(a.messages, llm.Message{
				Role:       llm.RoleTool,
				Name:       call.Function.Name,
				ToolCallID: call.ID,
				Content:    result,
			})
		}
	}

	if err := a.saveMemory(ctx); err != nil {
		return "", err
	}
	return "", fmt.Errorf("agent stopped after %d steps without a final answer", a.maxSteps)
}

func (a *Agent) validateRun(ctx context.Context, input string) error {
	if a == nil {
		return errors.New("agent is required")
	}
	if ctx == nil {
		return errors.New("context is required")
	}
	if strings.TrimSpace(input) == "" {
		return errors.New("input cannot be empty")
	}
	if a.provider == nil {
		return errors.New("llm provider is required")
	}
	return nil
}

func (a *Agent) chat(ctx context.Context, request llm.ChatRequest, stream bool, onEvent EventHandler) (llm.Message, error) {
	if !stream {
		return a.provider.Chat(ctx, request)
	}
	return a.provider.Stream(ctx, request, func(event llm.StreamEvent) error {
		if onEvent == nil {
			return nil
		}
		return onEvent(Event{Type: "delta", Content: event.Delta})
	})
}

func (a *Agent) executeTool(ctx context.Context, call llm.ToolCall, onEvent EventHandler) string {
	name := strings.TrimSpace(call.Function.Name)
	if onEvent != nil {
		_ = onEvent(Event{Type: "tool_start", Tool: name})
	}

	result, err := a.callTool(ctx, name, call.Function.Arguments)
	if err != nil {
		result = "tool error: " + err.Error()
		if onEvent != nil {
			_ = onEvent(Event{Type: "tool_error", Tool: name, Content: err.Error()})
		}
		return result
	}

	if onEvent != nil {
		_ = onEvent(Event{Type: "tool_result", Tool: name, Content: result})
	}
	return result
}

func (a *Agent) callTool(ctx context.Context, name, rawArgs string) (string, error) {
	if a.tools == nil {
		return "", fmt.Errorf("tool registry is required")
	}
	tool, err := a.tools.Get(name)
	if err != nil {
		return "", err
	}

	args := map[string]any{}
	if strings.TrimSpace(rawArgs) != "" {
		if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
			return "", fmt.Errorf("parse %s arguments: %w", name, err)
		}
	}

	return tool.Execute(ctx, args)
}

func (a *Agent) requestMessages() []llm.Message {
	messages := []llm.Message{{
		Role:    llm.RoleSystem,
		Content: a.systemPrompt,
	}}
	messages = append(messages, a.messages...)
	return messages
}

func (a *Agent) toolDefinitions() []llm.ToolDefinition {
	if a == nil || a.tools == nil {
		return nil
	}
	registered := a.tools.List()
	definitions := make([]llm.ToolDefinition, 0, len(registered))
	for _, tool := range registered {
		definitions = append(definitions, llm.ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}
	return definitions
}

func (a *Agent) saveMemory(ctx context.Context) error {
	if a == nil || a.memory == nil {
		return nil
	}
	return a.memory.Save(ctx, a.messages)
}
