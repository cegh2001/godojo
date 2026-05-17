package ports

import (
	"context"

	"godojo/internal/adapters/chatstore"
	"godojo/internal/core/domain"
)

// StreamChunk represents a single chunk in a streaming response.
// It may contain text, a function call, an error, or signal completion.
type StreamChunk struct {
	Text         string
	FunctionCall *domain.FunctionCall
	Done         bool
	Error        error
}

// SenseiProvider defines the contract for communicating with the AI sensei.
// It replaces the old chatProvider interface for the agentic TUI.
type SenseiProvider interface {
	// SendMessage sends the conversation history and tool declarations to the AI.
	// Returns content parts which may be text, function calls, or both.
	SendMessage(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) ([]domain.ContentPart, error)

	// SendMessageStream sends the conversation history and tool declarations to the AI
	// and returns a channel of streaming chunks. The channel is closed when the stream
	// ends or an error is sent. For errors that prevent the stream from starting, the
	// method returns (nil, error).
	SendMessageStream(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) (<-chan StreamChunk, error)

	// SendFunctionResponse sends a tool execution result back to the AI.
	// This continues the agent loop after a function call has been executed locally.
	SendFunctionResponse(ctx context.Context, history []chatstore.ChatMessage, callID string, name string, result interface{}) ([]domain.ContentPart, error)
}
