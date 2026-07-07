package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"

	projectcontext "github.com/MehulCodr/AI-agent/internal/context"
	"github.com/MehulCodr/AI-agent/internal/llm"
)

func (a *Agent) providerMessages(ctx context.Context) ([]llm.Message, error) {
	prompt, err := a.systemPrompt(ctx)
	if err != nil {
		return nil, err
	}

	messages := []llm.Message{{Role: "system", Content: prompt}}
	messages = append(messages, a.Messages()...)
	return messages, nil
}

func (a *Agent) systemPrompt(ctx context.Context) (string, error) {
	summary, err := projectcontext.NewScanner("").Scan(ctx)
	if err != nil {
		return "", err
	}

	var prompt strings.Builder
	prompt.WriteString("You are AI-agent, a CLI-first coding assistant.\n")
	prompt.WriteString("Use the project context and available tools before making claims about files.\n")
	prompt.WriteString("When the user asks about current files, directories, or code contents, inspect them with tools instead of guessing.\n")
	prompt.WriteString("Before changing a file, read it first. Use edit_file for targeted replacements and write_file only for new files or explicit full rewrites.\n")
	prompt.WriteString("Use edit_file with apply=false to preview an edit unless the user clearly asks you to modify files.\n")
	prompt.WriteString("For shell commands, use run_shell only for safe commands.\n")
	prompt.WriteString("If native tool calling is unavailable, respond only with JSON in this shape: {\"tool_calls\":[{\"function\":{\"name\":\"list_files\",\"arguments\":{\"path\":\".\"}}}]}.\n")

	prompt.WriteString("\nAvailable tools:\n")
	for _, tool := range a.toolDefinitions() {
		prompt.WriteString("- ")
		prompt.WriteString(tool.Name)
		if tool.Description != "" {
			prompt.WriteString(": ")
			prompt.WriteString(tool.Description)
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("\nProject context:\n")
	prompt.WriteString(fmt.Sprintf("Root: %s\n", summary.Root))
	prompt.WriteString(fmt.Sprintf("Total files: %d\n", summary.TotalFiles))
	prompt.WriteString(fmt.Sprintf("Go files: %d\n", summary.GoFiles))
	prompt.WriteString("Languages: ")
	prompt.WriteString(formatLanguages(summary.Languages))
	prompt.WriteString("\n")
	prompt.WriteString("Important directories: ")
	prompt.WriteString(strings.Join(summary.ImportantDirs, ", "))
	prompt.WriteString("\nTree:\n")
	prompt.WriteString(summary.Tree)

	return prompt.String(), nil
}

func formatLanguages(languages map[string]int) string {
	if len(languages) == 0 {
		return "none"
	}

	names := make([]string, 0, len(languages))
	for name := range languages {
		names = append(names, name)
	}
	sort.Strings(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("%s=%d", name, languages[name]))
	}
	return strings.Join(parts, ", ")
}
