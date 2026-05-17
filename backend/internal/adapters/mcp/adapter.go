// Package mcp defines the adapter interface for exposing GoDojo tools
// via the Model Context Protocol (MCP) stdio transport.
//
// This is an interface-only package — no implementation exists yet.
// Future implementation will wrap the ToolRegistry, WorkspaceManager,
// and TestRunner to expose them as MCP-compatible tools.
package mcp

import (
	"context"

	"godojo/internal/core/domain"
)

// MCPAdapter defines the contract for exposing GoDojo tools via the MCP stdio protocol.
// Implementations handle JSON-RPC message serialization over stdio and
// delegate tool execution to the existing domain services.
type MCPAdapter interface {
	// ListTools returns all available tool declarations exposed via MCP.
	// Each declaration describes a tool's name, description, and parameter schema.
	ListTools() []domain.ToolDeclaration

	// CallTool executes a tool by name with the given arguments.
	// Returns the result as an opaque value (maps to JSON in the MCP response).
	// Returns an error if the tool is not found or execution fails.
	CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)
}
