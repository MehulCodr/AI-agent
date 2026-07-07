package tools

import (
	"fmt"
	"sort"

	apperrors "github.com/MehulCodr/AI-agent/internal/errors"
)

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(tool Tool) error {
	if tool == nil {
		return fmt.Errorf("%w: register tool: tool is nil", apperrors.ErrInvalidInput)
	}

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("%w: register tool: name is required", apperrors.ErrInvalidInput)
	}
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("%w: register tool %q: already registered", apperrors.ErrInvalidInput, name)
	}

	r.tools[name] = tool
	return nil
}

func (r *Registry) Get(name string) (Tool, error) {
	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("%w: tool %q", apperrors.ErrToolNotFound, name)
	}

	return tool, nil
}

func (r *Registry) List() []Tool {
	names := r.Names()
	registered := make([]Tool, 0, len(names))
	for _, name := range names {
		registered = append(registered, r.tools[name])
	}

	return registered
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}
