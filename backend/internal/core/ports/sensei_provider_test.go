package ports

import (
	"errors"
	"testing"

	"godojo/internal/core/domain"
)

func TestStreamChunk_FieldsAccessible(t *testing.T) {
	// Test 1: Zero value compilation check
	var c StreamChunk
	if c.Text != "" {
		t.Errorf("zero Text = %q, want empty string", c.Text)
	}
	if c.FunctionCall != nil {
		t.Errorf("zero FunctionCall = %v, want nil", c.FunctionCall)
	}
	if c.Done {
		t.Errorf("zero Done = true, want false")
	}
	if c.Error != nil {
		t.Errorf("zero Error = %v, want nil", c.Error)
	}
}

func TestStreamChunk_WithAllFields(t *testing.T) {
	fc := &domain.FunctionCall{Name: "test_tool", Args: map[string]interface{}{"key": "value"}}
	err := errors.New("stream error")

	c := StreamChunk{
		Text:         "partial response",
		FunctionCall: fc,
		Done:         true,
		Error:        err,
	}

	if c.Text != "partial response" {
		t.Errorf("Text = %q, want %q", c.Text, "partial response")
	}
	if c.FunctionCall == nil {
		t.Fatal("FunctionCall is nil, want non-nil")
	}
	if c.FunctionCall.Name != "test_tool" {
		t.Errorf("FunctionCall.Name = %q, want %q", c.FunctionCall.Name, "test_tool")
	}
	if !c.Done {
		t.Errorf("Done = false, want true")
	}
	if c.Error == nil {
		t.Fatal("Error is nil, want non-nil")
	}
	if c.Error.Error() != "stream error" {
		t.Errorf("Error = %q, want %q", c.Error.Error(), "stream error")
	}
}

func TestStreamChunk_DoneWithText(t *testing.T) {
	// Edge case: both Done and Text populated (final chunk with text)
	c := StreamChunk{
		Text: "final answer",
		Done: true,
	}

	if !c.Done {
		t.Error("Done should be true for final chunk")
	}
	if c.Text != "final answer" {
		t.Errorf("Text = %q, want %q", c.Text, "final answer")
	}
}

func TestStreamChunk_ErrorOnly(t *testing.T) {
	// Edge case: error chunk without text
	c := StreamChunk{
		Error: errors.New("connection lost"),
	}

	if c.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if c.Done {
		t.Error("Done should be false for error-only chunk")
	}
	if c.Text != "" {
		t.Errorf("Text should be empty for error-only chunk, got %q", c.Text)
	}
}
