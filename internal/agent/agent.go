package agent

import (
	"context"

	"github.com/MehulCodr/AI-agent/internal/llm"
	"github.com/MehulCodr/AI-agent/internal/tools"
)

const defaultMaxSteps = 8

type Memory interface {
	Load(ctx context.Context) ([]llm.Message, error)
	Save(ctx context.Context, messages []llm.Message) error
	Clear(ctx context.Context) error
}

type Event struct {
	Type    string
	Content string
	Tool    string
}

type EventHandler func(Event) error

type Option func(*Agent)

type Agent struct {
	provider     llm.Provider
	tools        *tools.Registry
	maxSteps     int
	messages     []llm.Message
	memory       Memory
	systemPrompt string
	approval     tools.ApprovalFunc
	repoSearcher tools.RepoSearcher
}

func New(provider llm.Provider, options ...Option) *Agent {
	agent := &Agent{
		provider:     provider,
		maxSteps:     defaultMaxSteps,
		systemPrompt: defaultSystemPrompt,
	}
	for _, option := range options {
		if option != nil {
			option(agent)
		}
	}
	if agent.tools == nil {
		agent.tools = defaultToolRegistry(agent.approval, agent.repoSearcher)
	}
	return agent
}

func WithMaxSteps(n int) Option {
	return func(a *Agent) {
		if n > 0 {
			a.maxSteps = n
		}
	}
}

func WithMemory(memory Memory) Option {
	return func(a *Agent) {
		a.memory = memory
	}
}

func WithSystemPrompt(prompt string) Option {
	return func(a *Agent) {
		if prompt != "" {
			a.systemPrompt = prompt
		}
	}
}

func WithApproval(approval tools.ApprovalFunc) Option {
	return func(a *Agent) {
		a.approval = approval
	}
}

func WithRepoSearcher(searcher tools.RepoSearcher) Option {
	return func(a *Agent) {
		a.repoSearcher = searcher
	}
}

func WithTools(registry *tools.Registry) Option {
	return func(a *Agent) {
		a.tools = registry
	}
}

func (a *Agent) LoadMemory(ctx context.Context) error {
	if a == nil || a.memory == nil {
		return nil
	}
	messages, err := a.memory.Load(ctx)
	if err != nil {
		return err
	}
	a.messages = cloneMessages(messages)
	return nil
}

func (a *Agent) Messages() []llm.Message {
	if a == nil {
		return nil
	}
	return cloneMessages(a.messages)
}

func (a *Agent) Clear() {
	if a == nil {
		return
	}
	a.messages = nil
	if a.memory != nil {
		_ = a.memory.Clear(context.Background())
	}
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

func defaultToolRegistry(approval tools.ApprovalFunc, searcher tools.RepoSearcher) *tools.Registry {
	registry := tools.NewRegistry()
	defaults := []tools.Tool{
		tools.EchoTool{},
		tools.CurrentDirectoryTool{},
		tools.ListFilesTool{},
		tools.ReadFileTool{},
		tools.WriteFileTool{},
		tools.EditFileTool{},
	}
	if approval != nil {
		defaults = append(defaults, tools.ShellTool{Approve: approval})
	}
	if searcher != nil {
		defaults = append(defaults, tools.SearchRepoTool{Searcher: searcher})
	}
	for _, tool := range defaults {
		_ = registry.Register(tool)
	}

	return registry
}

func cloneMessages(messages []llm.Message) []llm.Message {
	cloned := make([]llm.Message, len(messages))
	for i, message := range messages {
		cloned[i] = message
		if len(message.ToolCalls) > 0 {
			cloned[i].ToolCalls = append([]llm.ToolCall(nil), message.ToolCalls...)
		}
	}
	return cloned
}
