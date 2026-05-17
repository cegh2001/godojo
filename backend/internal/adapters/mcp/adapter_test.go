package mcp

import (
	"context"
	"testing"

	"godojo/internal/core/domain"
)

// stubMCPAdapter is a compile-time check that a struct can implement MCPAdapter.
type stubMCPAdapter struct {
	tools []domain.ToolDeclaration
}

func (s *stubMCPAdapter) ListTools() []domain.ToolDeclaration {
	return s.tools
}

func (s *stubMCPAdapter) CallTool(_ context.Context, _ string, _ map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{"ok": true}, nil
}

// compile-time interface check
var _ MCPAdapter = (*stubMCPAdapter)(nil)

func TestMCPAdapter_InterfaceCompiles(t *testing.T) {
	var adapter MCPAdapter = &stubMCPAdapter{
		tools: []domain.ToolDeclaration{
			{
				Name:        "test_tool",
				Description: "A test tool declaration",
				Parameters: domain.ToolParameters{
					Type: "OBJECT",
					Properties: map[string]domain.ToolProperty{
						"arg": {Type: "STRING", Description: "A test argument"},
					},
					Required: []string{"arg"},
				},
			},
		},
	}

	// Verify ListTools returns the declaration
	tools := adapter.ListTools()
	if len(tools) != 1 {
		t.Fatalf("ListTools() = %d items, want 1", len(tools))
	}
	if tools[0].Name != "test_tool" {
		t.Errorf("tool name = %q, want %q", tools[0].Name, "test_tool")
	}

	// Verify CallTool works
	result, err := adapter.CallTool(context.Background(), "test_tool", nil)
	if err != nil {
		t.Fatalf("CallTool() error: %v", err)
	}
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("CallTool() result is not a map[string]interface{}")
	}
	if resultMap["ok"] != true {
		t.Errorf("CallTool() result = %v, want map[ok:true]", result)
	}
}

func TestMCPAdapter_NilInterface(t *testing.T) {
	// Verifies the interface type exists and can be used as a nil value
	var adapter MCPAdapter
	if adapter != nil {
		t.Error("nil MCPAdapter should be nil")
	}
}
