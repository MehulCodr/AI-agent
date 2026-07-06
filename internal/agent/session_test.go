package agent

import (
	"path/filepath"
	"testing"

	"github.com/MehulCodr/AI-agent/internal/llm"
)

func TestMemoryAddListMessages(t *testing.T) {
	memory := NewMemory()
	memory.Add(llm.Message{Role: "user", Content: "hello"})

	got := memory.List()
	if len(got) != 1 {
		t.Fatalf("len(List()) = %d, want 1", len(got))
	}
	if got[0].Role != "user" || got[0].Content != "hello" {
		t.Fatalf("message = %#v, want user hello", got[0])
	}

	got[0].Content = "changed"
	if memory.List()[0].Content != "hello" {
		t.Fatal("List returned internal message slice")
	}
}

func TestMemoryClearMessages(t *testing.T) {
	memory := NewMemory(llm.Message{Role: "user", Content: "hello"})
	memory.Clear()

	if len(memory.List()) != 0 {
		t.Fatalf("len(List()) = %d, want 0", len(memory.List()))
	}
}

func TestSaveSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".agent", "sessions", "latest.json")
	session := NewSession("test-session")
	session.Messages = []llm.Message{{Role: "user", Content: "hello"}}

	if err := SaveSession(path, session); err != nil {
		t.Fatalf("SaveSession returned error: %v", err)
	}

	loaded, err := LoadSession(path)
	if err != nil {
		t.Fatalf("LoadSession returned error: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded session is nil")
	}
	if loaded.ID != "test-session" {
		t.Fatalf("ID = %q, want test-session", loaded.ID)
	}
	if len(loaded.Messages) != 1 || loaded.Messages[0].Content != "hello" {
		t.Fatalf("Messages = %#v, want saved message", loaded.Messages)
	}
}

func TestLoadSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "latest.json")
	session := &Session{
		ID:       "loaded-session",
		Messages: []llm.Message{{Role: "assistant", Content: "hi"}},
	}
	if err := SaveSession(path, session); err != nil {
		t.Fatalf("SaveSession returned error: %v", err)
	}

	got, err := LoadSession(path)
	if err != nil {
		t.Fatalf("LoadSession returned error: %v", err)
	}
	if got.ID != "loaded-session" {
		t.Fatalf("ID = %q, want loaded-session", got.ID)
	}

	got.Messages[0].Content = "changed"
	again, err := LoadSession(path)
	if err != nil {
		t.Fatalf("LoadSession returned error: %v", err)
	}
	if again.Messages[0].Content != "hi" {
		t.Fatal("LoadSession returned mutable persisted data")
	}
}

func TestLoadMissingSessionDoesNotCrash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "latest.json")

	got, err := LoadSession(path)
	if err != nil {
		t.Fatalf("LoadSession returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("session = %#v, want nil", got)
	}
}

func TestAgentUsesLoadedSessionMemory(t *testing.T) {
	session := NewSession("loaded")
	session.Messages = []llm.Message{{Role: "user", Content: "previous"}}

	agent := NewWithSession(&fakeProvider{response: llm.Message{Role: "assistant", Content: "ok"}}, session)
	messages := agent.Messages()
	if len(messages) != 1 || messages[0].Content != "previous" {
		t.Fatalf("Messages() = %#v, want loaded history", messages)
	}
}
