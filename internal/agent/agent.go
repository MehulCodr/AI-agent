package agent

import "github.com/MehulCodr/AI-agent/internal/llm"

const defaultMaxSteps = 5

type Agent struct {
	provider llm.Provider
	maxSteps int
	messages []llm.Message
}

func New(provider llm.Provider) *Agent {
	return &Agent{
		provider: provider,
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
