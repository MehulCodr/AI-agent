package agent

import (
	"github.com/MehulCodr/AI-agent/internal/llm"
	"github.com/MehulCodr/AI-agent/internal/tools"
)

const defaultMaxSteps = 5

type Agent struct {
	provider llm.Provider
	tools    *tools.Registry
	maxSteps int
	messages []llm.Message
}

func New(provider llm.Provider) *Agent {
	return &Agent{
		provider: provider,
		tools:    defaultToolRegistry(),
		maxSteps: defaultMaxSteps,
	}
}

func (a *Agent) Messages() []llm.Message {
	if a == nil {
		return nil
	}
	messages := make([]llm.Message, len(a.messages))
	copy(messages, a.messages)
	return messages
}

func (a *Agent) Clear() {
	if a == nil {
		return
	}
	a.messages = nil
}

func (a *Agent) SetMaxSteps(n int) {
	if a == nil {
		return
	}
	if n <= 0 {
		return
	}
	a.maxSteps = n
}

func (a *Agent) Tools() []string {
	if a == nil || a.tools == nil {
		return nil
	}

	return a.tools.Names()
}

func (a *Agent) SetTools(registry *tools.Registry) {
	if a == nil {
		return
	}

	a.tools = registry
}

func defaultToolRegistry() *tools.Registry {
	registry := tools.NewRegistry()
	for _, tool := range []tools.Tool{
		tools.EchoTool{},
		tools.CurrentDirectoryTool{},
		tools.ListFilesTool{},
		tools.ReadFileTool{},
		tools.WriteFileTool{},
		tools.EditFileTool{},
		tools.ShellTool{},
	} {
		_ = registry.Register(tool)
	}

	return registry
}
