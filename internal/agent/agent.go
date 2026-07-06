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
	memory   *Memory
	session  *Session
}

func New(provider llm.Provider) *Agent {
	return NewWithSession(provider, nil)
}

func NewWithSession(provider llm.Provider, session *Session) *Agent {
	if session == nil {
		session = NewSession("latest")
	}
	return &Agent{
		provider: provider,
		tools:    defaultToolRegistry(),
		maxSteps: defaultMaxSteps,
		memory:   NewMemory(session.Messages...),
		session:  session,
	}
}

func (a *Agent) Messages() []llm.Message {
	if a == nil || a.memory == nil {
		return nil
	}
	return a.memory.List()
}

func (a *Agent) Clear() {
	if a == nil {
		return
	}
	if a.memory != nil {
		a.memory.Clear()
	}
	a.syncSession()
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

func (a *Agent) SaveSession() error {
	if a == nil {
		return nil
	}
	a.syncSession()
	return SaveLatestSession(a.session)
}

func (a *Agent) syncSession() {
	if a == nil || a.session == nil {
		return
	}
	a.session.Messages = a.Messages()
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
