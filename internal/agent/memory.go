package agent

import "github.com/MehulCodr/AI-agent/internal/llm"

type Memory struct {
	messages []llm.Message
}

func NewMemory(messages ...llm.Message) *Memory {
	memory := &Memory{}
	memory.messages = copyMessages(messages)
	return memory
}

func (m *Memory) Add(message llm.Message) {
	if m == nil {
		return
	}
	m.messages = append(m.messages, message)
}

func (m *Memory) List() []llm.Message {
	if m == nil {
		return nil
	}
	return copyMessages(m.messages)
}

func (m *Memory) Clear() {
	if m == nil {
		return
	}
	m.messages = nil
}

func copyMessages(messages []llm.Message) []llm.Message {
	copied := make([]llm.Message, len(messages))
	copy(copied, messages)
	return copied
}
