package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	apperrors "github.com/MehulCodr/AI-agent/internal/errors"
	"github.com/MehulCodr/AI-agent/internal/llm"
)

const (
	maxToolResultChars    = 12000
	toolResultInstruction = "Use these tool results to continue answering the original user request. If more information is needed, call another tool."
)

func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	if a == nil {
		return "", fmt.Errorf("%w: agent is required", apperrors.ErrInvalidInput)
	}
	if ctx == nil {
		return "", fmt.Errorf("%w: context is required", apperrors.ErrInvalidInput)
	}
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("%w: input cannot be empty", apperrors.ErrInvalidInput)
	}
	if a.provider == nil {
		return "", fmt.Errorf("%w: llm provider is required", apperrors.ErrInvalidInput)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	a.memory.Add(llm.Message{
		Role:    "user",
		Content: input,
	})

	seenToolCalls := make(map[string]bool)
	var lastToolResult string

	for step := 0; step < a.maxSteps; step++ {
		messages, err := a.providerMessages(ctx)
		if err != nil {
			return "", err
		}

		response, err := a.chat(ctx, messages)
		if err != nil {
			return "", err
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}

		response.ToolCalls = mergeToolCalls(response.ToolCalls, parseToolCalls(response.Content))
		if len(response.ToolCalls) == 0 {
			a.memory.Add(response)
			a.syncSession()
			return response.Content, nil
		}

		if repeatedToolCalls(seenToolCalls, response.ToolCalls) && lastToolResult != "" {
			content := fallbackToolResponse(lastToolResult)
			a.memory.Add(llm.Message{Role: "assistant", Content: content})
			a.syncSession()
			return content, nil
		}
		rememberToolCalls(seenToolCalls, response.ToolCalls)

		toolResult, err := a.executeToolCalls(ctx, response.ToolCalls)
		if err != nil {
			return "", err
		}
		lastToolResult = toolResult
		a.memory.Add(llm.Message{
			Role:    "user",
			Content: toolResult,
		})
	}

	if lastToolResult != "" {
		content := fallbackToolResponse(lastToolResult)
		a.memory.Add(llm.Message{Role: "assistant", Content: content})
		a.syncSession()
		return content, nil
	}

	return "", fmt.Errorf("%w: reached max agent steps", apperrors.ErrInvalidInput)
}

func (a *Agent) chat(ctx context.Context, messages []llm.Message) (llm.Message, error) {
	if provider, ok := a.provider.(llm.ToolAwareProvider); ok {
		return provider.ChatWithTools(ctx, messages, a.toolDefinitions())
	}
	return a.provider.Chat(ctx, messages)
}

func (a *Agent) executeToolCalls(ctx context.Context, calls []llm.ToolCall) (string, error) {
	if a.tools == nil {
		return "", fmt.Errorf("%w: tools registry is required", apperrors.ErrInvalidInput)
	}

	var result strings.Builder
	result.WriteString("Tool results:\n")

	for _, call := range calls {
		name := strings.TrimSpace(call.Function.Name)
		if name == "" {
			return "", fmt.Errorf("%w: tool call name is required", apperrors.ErrInvalidInput)
		}

		tool, err := a.tools.Get(name)
		if err != nil {
			return "", err
		}

		input, err := decodeToolArguments(call.Function.Arguments)
		if err != nil {
			return "", fmt.Errorf("parse %s arguments: %w", name, err)
		}

		output, err := tool.Execute(ctx, input)
		if err != nil {
			return "", fmt.Errorf("execute %s: %w", name, err)
		}

		result.WriteString("\n")
		result.WriteString(name)
		result.WriteString(":\n")
		result.WriteString(truncateToolResult(output))
		result.WriteString("\n")
	}

	result.WriteString("\n")
	result.WriteString(toolResultInstruction)
	return result.String(), nil
}

func repeatedToolCalls(seen map[string]bool, calls []llm.ToolCall) bool {
	if len(calls) == 0 {
		return false
	}
	for _, call := range calls {
		if seen[toolCallKey(call)] {
			return true
		}
	}
	return false
}

func rememberToolCalls(seen map[string]bool, calls []llm.ToolCall) {
	for _, call := range calls {
		seen[toolCallKey(call)] = true
	}
}

func toolCallKey(call llm.ToolCall) string {
	return strings.TrimSpace(call.Function.Name) + ":" + strings.TrimSpace(call.Function.Arguments)
}

func fallbackToolResponse(toolResult string) string {
	toolResult = strings.TrimSpace(toolResult)
	toolResult = strings.TrimSuffix(toolResult, toolResultInstruction)
	toolResult = strings.TrimSpace(toolResult)
	if toolResult == "" {
		return "The model kept requesting tools and did not produce a final response."
	}
	return "The model kept requesting tools, so here are the latest tool results:\n\n" + toolResult
}

func decodeToolArguments(arguments string) (map[string]any, error) {
	if strings.TrimSpace(arguments) == "" {
		return map[string]any{}, nil
	}

	var input map[string]any
	if err := json.Unmarshal([]byte(arguments), &input); err != nil {
		return nil, err
	}
	if input == nil {
		input = map[string]any{}
	}
	return input, nil
}

func truncateToolResult(output string) string {
	if len(output) <= maxToolResultChars {
		return output
	}
	return output[:maxToolResultChars] + "\n[tool result truncated]"
}

func mergeToolCalls(primary, fallback []llm.ToolCall) []llm.ToolCall {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func parseToolCalls(content string) []llm.ToolCall {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	content = trimJSONFence(content)

	var payload struct {
		ToolCalls []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil
	}
	if len(payload.ToolCalls) == 0 {
		return nil
	}

	calls := make([]llm.ToolCall, 0, len(payload.ToolCalls))
	for _, call := range payload.ToolCalls {
		arguments := strings.TrimSpace(string(call.Function.Arguments))
		if arguments == "" || arguments == "null" {
			arguments = "{}"
		}
		if strings.HasPrefix(arguments, `"`) {
			var decoded string
			if err := json.Unmarshal(call.Function.Arguments, &decoded); err == nil {
				arguments = decoded
			}
		}

		calls = append(calls, llm.ToolCall{
			ID:   call.ID,
			Type: call.Type,
			Function: llm.ToolCallFunction{
				Name:      call.Function.Name,
				Arguments: arguments,
			},
		})
	}
	return calls
}

func trimJSONFence(content string) string {
	if !strings.HasPrefix(content, "```") {
		return content
	}

	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "json")
	content = strings.TrimSpace(content)
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content)
}
