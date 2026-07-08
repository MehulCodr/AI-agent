package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MehulCodr/AI-agent/internal/llm"
)

type FileStore struct {
	Path string
}

func NewFileStore(path string) FileStore {
	return FileStore{Path: path}
}

func (s FileStore) Load(ctx context.Context) ([]llm.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read session %s: %w", s.Path, err)
	}
	if len(data) == 0 {
		return nil, nil
	}

	var messages []llm.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("parse session %s: %w", s.Path, err)
	}
	return messages, nil
}

func (s FileStore) Save(ctx context.Context, messages []llm.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0755); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(s.Path, data, 0600); err != nil {
		return fmt.Errorf("write session %s: %w", s.Path, err)
	}
	return nil
}

func (s FileStore) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Remove(s.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clear session %s: %w", s.Path, err)
	}
	return nil
}
