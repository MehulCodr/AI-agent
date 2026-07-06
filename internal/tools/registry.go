package tools

import (
	"fmt"
	"sort"
)

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(tool Tool) error {
	if tool == nil {
		return fmt.Errorf("register tool: tool is nil")
	}

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("register tool: name is required")
	}
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("register tool %q: already registered", name)
	}

	r.tools[name] = tool
	return nil
}

func (r *Registry) Get(name string) (Tool, error) {
	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %q not found", name)
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
