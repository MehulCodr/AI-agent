package agent

import (
	"github.com/MehulCodr/AI-agent/internal/llm"
)

func (a *Agent) toolDefinitions() []llm.ToolDefinition {
	if a == nil || a.tools == nil {
		return nil
	}

	tools := a.tools.List()
	definitions := make([]llm.ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		definitions = append(definitions, llm.ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  toolParameters(tool.Name()),
		})
	}
	return definitions
}

func toolParameters(name string) map[string]any {
	switch name {
	case "current_directory":
		return objectSchema(nil, nil)
	case "echo":
		return objectSchema(map[string]any{
			"text": stringSchema("Text to echo."),
		}, []string{"text"})
	case "edit_file":
		return objectSchema(map[string]any{
			"path":  stringSchema("Project-relative path to edit."),
			"old":   stringSchema("Exact text to replace."),
			"new":   stringSchema("Replacement text."),
			"apply": boolSchema("Whether to apply the edit. Use false to preview."),
		}, []string{"path", "old", "new"})
	case "list_files":
		return objectSchema(map[string]any{
			"path": stringSchema("Directory path to list. Defaults to current directory."),
		}, nil)
	case "read_file":
		return objectSchema(map[string]any{
			"path": stringSchema("Project-relative path to read."),
		}, []string{"path"})
	case "run_shell":
		return objectSchema(map[string]any{
			"command": stringSchema("Command executable to run."),
			"args": map[string]any{
				"type":        "array",
				"description": "Command arguments.",
				"items":       map[string]any{"type": "string"},
			},
		}, []string{"command"})
	case "write_file":
		return objectSchema(map[string]any{
			"path":    stringSchema("Project-relative path to write."),
			"content": stringSchema("Complete file content."),
		}, []string{"path", "content"})
	default:
		return objectSchema(nil, nil)
	}
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	if len(properties) > 0 {
		schema["properties"] = properties
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringSchema(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
	}
}

func boolSchema(description string) map[string]any {
	return map[string]any{
		"type":        "boolean",
		"description": description,
	}
}
