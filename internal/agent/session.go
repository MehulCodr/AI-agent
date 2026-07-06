package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MehulCodr/AI-agent/internal/llm"
)

const LatestSessionPath = ".agent/sessions/latest.json"

type Session struct {
	ID        string        `json:"id"`
	CreatedAt time.Time     `json:"created_at"`
	Messages  []llm.Message `json:"messages"`
}

func NewSession(id string) *Session {
	if id == "" {
		id = "latest"
	}
	return &Session{
		ID:        id,
		CreatedAt: time.Now().UTC(),
	}
}

func SaveLatestSession(session *Session) error {
	return SaveSession(LatestSessionPath, session)
}

func LoadLatestSession() (*Session, error) {
	return LoadSession(LatestSessionPath)
}

func SaveSession(path string, session *Session) error {
	if session == nil {
		return fmt.Errorf("session is required")
	}
	if path == "" {
		return fmt.Errorf("session path is required")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write session: %w", err)
	}
	return nil
}

func LoadSession(path string) (*Session, error) {
	if path == "" {
		return nil, fmt.Errorf("session path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read session: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}
	session.Messages = copyMessages(session.Messages)
	return &session, nil
}
