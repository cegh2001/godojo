package core

import (
	"fmt"
	"sync"

	"godojo/internal/core/domain"
)

// ToolHandler is a function that executes a tool with the given arguments.
type ToolHandler func(args map[string]interface{}) (interface{}, error)

// toolEntry holds the declaration and handler for a registered tool.
type toolEntry struct {
	declaration domain.ToolDeclaration
	handler     ToolHandler
}

// ToolRegistry maps tool names to their declarations and handlers.
// It is safe for concurrent use.
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]toolEntry
}

// NewToolRegistry creates an empty ToolRegistry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]toolEntry),
	}
}

// Register adds a tool to the registry. If a tool with the same name
// already exists, it is silently overwritten.
func (r *ToolRegistry) Register(name string, declaration domain.ToolDeclaration, handler ToolHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[name] = toolEntry{
		declaration: declaration,
		handler:     handler,
	}
}

// Execute runs the named tool with the given arguments.
// It validates that all required parameters are present.
func (r *ToolRegistry) Execute(name string, args map[string]interface{}) (interface{}, error) {
	r.mu.RLock()
	entry, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("herramienta desconocida: %q", name)
	}

	// Validate required args
	for _, required := range entry.declaration.Parameters.Required {
		if _, exists := args[required]; !exists {
			return nil, fmt.Errorf("falta el parámetro requerido %q para la herramienta %q", required, name)
		}
	}

	return entry.handler(args)
}

// GetDeclarations returns the declarations of all registered tools.
func (r *ToolRegistry) GetDeclarations() []domain.ToolDeclaration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.ToolDeclaration, 0, len(r.tools))
	for _, entry := range r.tools {
		result = append(result, entry.declaration)
	}
	return result
}
