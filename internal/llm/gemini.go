package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GeminiConfig ProviderConfig

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
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

func (p *GeminiProvider) Name() string {
	return "gemini"
}

func (p *GeminiProvider) Chat(ctx context.Context, request ChatRequest) (Message, error) {
	if err := p.validate(request); err != nil {
		return Message{}, err
	}

	payload := geminiPayload(request)
	endpoint := p.endpoint(request.Model, "generateContent")

	var result geminiResponse
	if err := postJSON(ctx, p.client, endpoint, nil, payload, &result, "gemini"); err != nil {
		return Message{}, err
	}

	return geminiMessage(result)
}

func (p *GeminiProvider) Stream(ctx context.Context, request ChatRequest, onEvent StreamHandler) (Message, error) {
	if err := p.validate(request); err != nil {
		return Message{}, err
	}

	payload := geminiPayload(request)
	endpoint := p.endpoint(request.Model, "streamGenerateContent") + "&alt=sse"

	var content strings.Builder
	var calls []ToolCall
	err := postStream(ctx, p.client, endpoint, nil, payload, "gemini", func(data string) error {
		var chunk geminiResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return err
		}
		message, err := geminiMessage(chunk)
		if err != nil {
			return nil
		}
		if message.Content != "" {
			content.WriteString(message.Content)
			if err := emitDelta(onEvent, message.Content); err != nil {
				return err
			}
		}
		calls = append(calls, message.ToolCalls...)
		return nil
	})
	if err != nil {
		return Message{}, err
	}

	return Message{Role: RoleAssistant, Content: content.String(), ToolCalls: calls}, nil
}

func (p *GeminiProvider) validate(request ChatRequest) error {
	if p.config.APIKey == "" {
		return fmt.Errorf("gemini api key is required")
	}
	if request.Model == "" && p.config.Model == "" {
		return fmt.Errorf("gemini model is required")
	}
	return nil
}

func (p *GeminiProvider) endpoint(requestModel, method string) string {
	model := requestModel
	if model == "" {
		model = p.config.Model
	}
	model = strings.TrimPrefix(model, "models/")
	return strings.TrimRight(p.config.BaseURL, "/") + "/models/" + url.PathEscape(model) + ":" + method + "?key=" + url.QueryEscape(p.config.APIKey)
}

type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	Tools             []geminiTool    `json:"tools,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

type geminiFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations"`
}

type geminiFunctionDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func geminiPayload(request ChatRequest) geminiRequest {
	system, contents := geminiContents(request.Messages)
	return geminiRequest{
		SystemInstruction: system,
		Contents:          contents,
		Tools:             geminiTools(request.Tools),
	}
}

func geminiContents(messages []Message) (*geminiContent, []geminiContent) {
	var system strings.Builder
	contents := make([]geminiContent, 0, len(messages))

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

		parts := geminiParts(message)
		if len(parts) == 0 {
			continue
		}

		role := "user"
		switch message.Role {
		case RoleAssistant:
			role = "model"
		case RoleTool:
			role = "function"
		}

		contents = append(contents, geminiContent{Role: role, Parts: parts})
	}

	var systemContent *geminiContent
	if system.Len() > 0 {
		systemContent = &geminiContent{Parts: []geminiPart{{Text: system.String()}}}
	}
	return systemContent, contents
}

func geminiParts(message Message) []geminiPart {
	parts := make([]geminiPart, 0, 1+len(message.ToolCalls))
	if message.Content != "" {
		if message.Role == RoleTool {
			name := message.Name
			if name == "" {
				name = "tool"
			}
			parts = append(parts, geminiPart{FunctionResponse: &geminiFunctionResponse{
				Name:     name,
				Response: map[string]any{"result": message.Content},
			}})
		} else {
			parts = append(parts, geminiPart{Text: message.Content})
		}
	}
	for _, call := range message.ToolCalls {
		args := map[string]any{}
		if strings.TrimSpace(call.Function.Arguments) != "" {
			_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
		}
		parts = append(parts, geminiPart{FunctionCall: &geminiFunctionCall{
			Name: call.Function.Name,
			Args: args,
		}})
	}
	return parts
}

func geminiTools(tools []ToolDefinition) []geminiTool {
	if len(tools) == 0 {
		return nil
	}

	declarations := make([]geminiFunctionDeclaration, 0, len(tools))
	for _, tool := range tools {
		declarations = append(declarations, geminiFunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		})
	}
	return []geminiTool{{FunctionDeclarations: declarations}}
}

func geminiMessage(result geminiResponse) (Message, error) {
	if len(result.Candidates) == 0 {
		return Message{}, fmt.Errorf("gemini response had no candidates")
	}

	var text strings.Builder
	var calls []ToolCall
	for i, part := range result.Candidates[0].Content.Parts {
		text.WriteString(part.Text)
		if part.FunctionCall != nil {
			args, err := json.Marshal(part.FunctionCall.Args)
			if err != nil {
				return Message{}, err
			}
			calls = append(calls, ToolCall{
				ID:   fmt.Sprintf("gemini_call_%d", i),
				Type: "function",
				Function: ToolCallFunction{
					Name:      part.FunctionCall.Name,
					Arguments: string(args),
				},
			})
		}
	}

	if text.Len() == 0 && len(calls) == 0 {
		return Message{}, fmt.Errorf("gemini response had no text or tool calls")
	}

	return Message{Role: RoleAssistant, Content: text.String(), ToolCalls: calls}, nil
}
