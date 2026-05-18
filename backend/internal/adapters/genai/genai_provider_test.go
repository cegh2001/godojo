package genai

import (
	"context"
	"strings"
	"testing"
	"time"

	"godojo/internal/adapters/chatstore"
	"godojo/internal/core/domain"
)

// --- History conversion tests ---

func TestBuildGenaiContents_EmptyHistory(t *testing.T) {
	memory, contents := buildGenaiContents([]chatstore.ChatMessage{})

	if memory != "" {
		t.Errorf("expected empty memory for empty history, got %q", memory)
	}
	if len(contents) != 1 {
		t.Fatalf("expected 1 content (default Hola), got %d", len(contents))
	}
	if contents[0].Role != "user" {
		t.Errorf("expected role 'user', got %q", contents[0].Role)
	}
	if len(contents[0].Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(contents[0].Parts))
	}
	if contents[0].Parts[0].Text != "Hola" {
		t.Errorf("expected text 'Hola', got %q", contents[0].Parts[0].Text)
	}
}

func TestBuildGenaiContents_SingleUserMessage(t *testing.T) {
	history := []chatstore.ChatMessage{
		{Role: "user", Content: "¿Qué es una variable?", Time: time.Now()},
	}

	memory, contents := buildGenaiContents(history)

	if memory != "" {
		t.Errorf("expected empty memory, got %q", memory)
	}
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}
	if contents[0].Role != "user" {
		t.Errorf("expected role 'user', got %q", contents[0].Role)
	}
	if contents[0].Parts[0].Text != "¿Qué es una variable?" {
		t.Errorf("expected text %q, got %q", "¿Qué es una variable?", contents[0].Parts[0].Text)
	}
}

func TestBuildGenaiContents_UserAndSenseiMessages(t *testing.T) {
	history := []chatstore.ChatMessage{
		{Role: "user", Content: "Hola", Time: time.Now()},
		{Role: "sensei", Content: "¡Buenas! ¿En qué te ayudo?", Time: time.Now()},
	}

	memory, contents := buildGenaiContents(history)

	if memory != "" {
		t.Errorf("expected empty memory, got %q", memory)
	}
	if len(contents) != 2 {
		t.Fatalf("expected 2 contents, got %d", len(contents))
	}
	if contents[0].Role != "user" {
		t.Errorf("expected role 'user' for first content, got %q", contents[0].Role)
	}
	if contents[1].Role != "model" {
		t.Errorf("expected role 'model' for sensei message, got %q", contents[1].Role)
	}
	if contents[1].Parts[0].Text != "¡Buenas! ¿En qué te ayudo?" {
		t.Errorf("expected sensei text, got %q", contents[1].Parts[0].Text)
	}
}

func TestBuildGenaiContents_HistoryExceedsMaxMessages(t *testing.T) {
	// Create 8 messages (exceeds maxChatContextMessages = 6)
	history := make([]chatstore.ChatMessage, 8)
	for i := 0; i < 8; i++ {
		role := "user"
		if i%2 == 1 {
			role = "sensei"
		}
		history[i] = chatstore.ChatMessage{
			Role:    role,
			Content: strings.Repeat("a", 50), // long enough to trigger summaries
			Time:    time.Now(),
		}
	}

	memory, contents := buildGenaiContents(history)

	if !strings.Contains(memory, "Resumen") && !strings.Contains(memory, "contexto") {
		t.Errorf("expected memory summary for truncated history, got %q", memory)
	}
	// Should contain at most maxChatContextMessages entries
	if len(contents) > maxChatContextMessages {
		t.Errorf("expected at most %d contents, got %d", maxChatContextMessages, len(contents))
	}
}

func TestBuildGenaiContents_SenseiRoleMapping(t *testing.T) {
	history := []chatstore.ChatMessage{
		{Role: "sensei", Content: "Claro, una variable es...", Time: time.Now()},
	}

	_, contents := buildGenaiContents(history)

	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}
	// Sensei role should map to "model" for Gemini API
	if contents[0].Role != "model" {
		t.Errorf("expected role 'model' for sensei, got %q", contents[0].Role)
	}
}

func TestBuildGenaiContents_NilHistory(t *testing.T) {
	memory, contents := buildGenaiContents(nil)

	if memory != "" {
		t.Errorf("expected empty memory for nil history, got %q", memory)
	}
	if len(contents) != 1 {
		t.Fatalf("expected 1 content (default Hola), got %d", len(contents))
	}
}

// --- Config building tests ---

func TestBuildGenaiConfig_WithSystemPromptAndTools(t *testing.T) {
	tools := []domain.ToolDeclaration{
		{
			Name:        "test_tool",
			Description: "A test tool",
			Parameters: domain.ToolParameters{
				Type:       "object",
				Properties: map[string]domain.ToolProperty{},
			},
		},
	}

	config := buildGenaiConfig("Sos un sensei.", "Resumen breve", tools)

	if config.SystemInstruction == nil {
		t.Fatal("SystemInstruction should not be nil")
	}
	if len(config.SystemInstruction.Parts) == 0 {
		t.Fatal("SystemInstruction should have parts")
	}
	sysText := config.SystemInstruction.Parts[0].Text
	if !strings.Contains(sysText, "Sos un sensei.") {
		t.Errorf("system instruction missing prompt, got %q", sysText)
	}
	if !strings.Contains(sysText, "Resumen breve") {
		t.Errorf("system instruction missing memory, got %q", sysText)
	}
	if len(config.Tools) != 1 {
		t.Fatalf("expected 1 tool in config, got %d", len(config.Tools))
	}
}

func TestBuildGenaiConfig_NoTools(t *testing.T) {
	config := buildGenaiConfig("Sos un sensei.", "", nil)

	if config.SystemInstruction == nil {
		t.Fatal("SystemInstruction should not be nil")
	}
	if len(config.Tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(config.Tools))
	}
}

func TestBuildGenaiConfig_EmptyPromptUsesDefault(t *testing.T) {
	config := buildGenaiConfig("", "", nil)

	if config.SystemInstruction == nil {
		t.Fatal("SystemInstruction should not be nil")
	}
	sysText := config.SystemInstruction.Parts[0].Text
	if !strings.Contains(sysText, "sensei") {
		t.Errorf("expected default prompt when empty, got %q", sysText)
	}
}

// --- Function response building tests ---

func TestBuildFunctionResponseContents_AppendsResponse(t *testing.T) {
	history := []chatstore.ChatMessage{
		{Role: "user", Content: "Creá un archivo", Time: time.Now()},
		{Role: "sensei", Content: "[functionCall: create_exercise_file]", Time: time.Now()},
	}

	contents := buildFunctionResponseContents(history, "call-1", "create_exercise_file", map[string]interface{}{
		"success": true,
	})

	// Should have 2 history contents + 1 function response content = 3
	if len(contents) != 3 {
		t.Fatalf("expected 3 contents (2 history + 1 response), got %d", len(contents))
	}

	// Last content should be the function response
	last := contents[len(contents)-1]
	if last.Role != "user" {
		t.Errorf("expected role 'user' for function response, got %q", last.Role)
	}
	if len(last.Parts) != 1 {
		t.Fatalf("expected 1 part in function response, got %d", len(last.Parts))
	}
	fr := last.Parts[0].FunctionResponse
	if fr == nil {
		t.Fatal("expected FunctionResponse part")
	}
	if fr.Name != "create_exercise_file" {
		t.Errorf("expected name 'create_exercise_file', got %q", fr.Name)
	}
	if fr.ID != "call-1" {
		t.Errorf("expected ID 'call-1', got %q", fr.ID)
	}
}

func TestBuildFunctionResponseContents_EmptyHistory(t *testing.T) {
	contents := buildFunctionResponseContents(nil, "call-2", "read_codebase", map[string]interface{}{
		"files": map[string]string{"main.go": "package main"},
	})

	// buildGenaiContents(nil) returns 1 default "Hola" content + 1 function response = 2
	if len(contents) != 2 {
		t.Fatalf("expected 2 contents (1 default + 1 response), got %d", len(contents))
	}
	// The function response should be the last content
	last := contents[len(contents)-1]
	if last.Role != "user" {
		t.Errorf("expected role 'user' for function response, got %q", last.Role)
	}
	if last.Parts[0].FunctionResponse == nil {
		t.Fatal("expected FunctionResponse part in last content")
	}
}

// --- Provider error tests ---

func TestGenaiProvider_SendMessage_MissingAPIKey(t *testing.T) {
	p := &GenaiProvider{apiKey: ""}

	_, err := p.SendMessage(context.Background(), "Sos un sensei.", nil, nil)
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
	if !strings.Contains(err.Error(), "GEMINI_API_KEY") {
		t.Errorf("expected error mentioning GEMINI_API_KEY, got %q", err.Error())
	}
}

func TestGenaiProvider_SendFunctionResponse_MissingAPIKey(t *testing.T) {
	p := &GenaiProvider{apiKey: ""}

	_, err := p.SendFunctionResponse(context.Background(), nil, "call-1", "test", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
	if !strings.Contains(err.Error(), "GEMINI_API_KEY") {
		t.Errorf("expected error mentioning GEMINI_API_KEY, got %q", err.Error())
	}
}

// --- NewGenaiProvider default values test ---

func TestNewGenaiProvider_Defaults(t *testing.T) {
	// Without API key set, should still create provider
	p := NewGenaiProvider()

	if p == nil {
		t.Fatal("NewGenaiProvider should not return nil")
	}
	if p.model == "" {
		t.Error("expected model to have a default, got empty")
	}
	if p.timeout <= 0 {
		t.Errorf("expected positive timeout, got %v", p.timeout)
	}
	if p.toolTimeout <= 0 {
		t.Errorf("expected positive tool timeout, got %v", p.toolTimeout)
	}
	if p.toolTimeout < p.timeout {
		t.Errorf("tool timeout (%v) should be >= timeout (%v)", p.toolTimeout, p.timeout)
	}
}

// --- Context timeout mapping test ---

func TestGenaiProvider_mapGenaiError_DeadlineExceeded(t *testing.T) {
	err := mapGenaiError(context.DeadlineExceeded)
	if !strings.Contains(err, "tardó demasiado") {
		t.Errorf("expected timeout message, got %q", err)
	}
}

// --- Model selection tests ---

func TestNewGenaiProvider_WithModelSelection(t *testing.T) {
	tests := []struct {
		name           string
		fastEnv        string
		flashEnv       string
		wantFastModel  string
		wantFlashModel string
	}{
		{
			name:           "both env vars set",
			fastEnv:        "gemma-4-26b-a4b-it",
			flashEnv:       "gemini-2.5-flash",
			wantFastModel:  "gemma-4-26b-a4b-it",
			wantFlashModel: "gemini-2.5-flash",
		},
		{
			name:           "only fast model set",
			fastEnv:        "custom-fast-model",
			flashEnv:       "",
			wantFastModel:  "custom-fast-model",
			wantFlashModel: "",
		},
		{
			name:           "neither set, use defaults",
			fastEnv:        "",
			flashEnv:       "",
			wantFastModel:  defaultFastModel,
			wantFlashModel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars for this test case
			if tt.fastEnv != "" {
				t.Setenv(senseiFastModelEnv, tt.fastEnv)
			}
			if tt.flashEnv != "" {
				t.Setenv(senseiFlashModelEnv, tt.flashEnv)
			}

			p := NewGenaiProvider()

			if p.fastModel != tt.wantFastModel {
				t.Errorf("fastModel = %q, want %q", p.fastModel, tt.wantFastModel)
			}
			if p.flashModel != tt.wantFlashModel {
				t.Errorf("flashModel = %q, want %q", p.flashModel, tt.wantFlashModel)
			}
		})
	}
}

func TestSelectEffectiveModel_ToolsPresent(t *testing.T) {
	// When tools are present, always use heavy model
	result := selectEffectiveModel(
		[]domain.ToolDeclaration{{Name: "test_tool"}},
		nil,
		"fast-model",
		"flash-model",
		"heavy-model",
	)
	if result != "heavy-model" {
		t.Errorf("with tools present, got %q, want heavy-model", result)
	}
}

func TestSelectEffectiveModel_NoTools_FastModel(t *testing.T) {
	// No tools, no flash configured → fast model
	result := selectEffectiveModel(
		nil,
		[]chatstore.ChatMessage{{Role: "user", Content: "¿Qué es un puntero?"}},
		"fast-model",
		"", // flash disabled
		"heavy-model",
	)
	if result != "fast-model" {
		t.Errorf("with no tools and no flash, got %q, want fast-model", result)
	}
}

func TestSelectEffectiveModel_NoTools_FlashTriggered(t *testing.T) {
	// Simple greeting + flash configured → flash model
	result := selectEffectiveModel(
		nil,
		[]chatstore.ChatMessage{{Role: "user", Content: "hola"}},
		"fast-model",
		"flash-model",
		"heavy-model",
	)
	if result != "flash-model" {
		t.Errorf("with greeting 'hola' and flash enabled, got %q, want flash-model", result)
	}
}

func TestSelectEffectiveModel_NoTools_SimpleIntentNoFlash(t *testing.T) {
	// Simple greeting but flash NOT configured → fast model
	result := selectEffectiveModel(
		nil,
		[]chatstore.ChatMessage{{Role: "user", Content: "holi"}},
		"fast-model",
		"", // no flash
		"heavy-model",
	)
	if result != "fast-model" {
		t.Errorf("with greeting and no flash, got %q, want fast-model", result)
	}
}

func TestSelectEffectiveModel_FlashIntentHints(t *testing.T) {
	tests := []struct {
		name          string
		message       string
		flashDisabled bool
		wantFast      string
		wantFlash     string
	}{
		{name: "hola triggers flash", message: "hola", flashDisabled: false, wantFlash: "flash-model"},
		{name: "holi triggers flash", message: "holi", flashDisabled: false, wantFlash: "flash-model"},
		{name: "chau triggers flash", message: "chau", flashDisabled: false, wantFlash: "flash-model"},
		{name: "adiós triggers flash", message: "adiós", flashDisabled: false, wantFlash: "flash-model"},
		{name: "qué tal triggers flash", message: "qué tal", flashDisabled: false, wantFlash: "flash-model"},
		{name: "qué podés hacer triggers flash", message: "qué podés hacer", flashDisabled: false, wantFlash: "flash-model"},
		{name: "gracias triggers flash", message: "gracias", flashDisabled: false, wantFlash: "flash-model"},
		{name: "buenas triggers flash", message: "buenas", flashDisabled: false, wantFlash: "flash-model"},
		{name: "ayuda triggers flash", message: "ayuda", flashDisabled: false, wantFlash: "flash-model"},
		{name: "complex question uses fast", message: "¿Cómo implemento un binary search tree en Go?", flashDisabled: false, wantFast: "fast-model"},
		{name: "flash disabled falls back to fast", message: "hola", flashDisabled: true, wantFast: "fast-model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flashModel := "flash-model"
			if tt.flashDisabled {
				flashModel = ""
			}

			result := selectEffectiveModel(
				nil,
				[]chatstore.ChatMessage{{Role: "user", Content: tt.message}},
				"fast-model",
				flashModel,
				"heavy-model",
			)

			if tt.wantFlash != "" {
				if result != tt.wantFlash {
					t.Errorf("message %q got %q, want flash: %q", tt.message, result, tt.wantFlash)
				}
			} else if tt.wantFast != "" {
				if result != tt.wantFast {
					t.Errorf("message %q got %q, want fast: %q", tt.message, result, tt.wantFast)
				}
			}
		})
	}
}

func TestSelectEffectiveModel_LastUserMessageOnly(t *testing.T) {
	// Only the LAST user message is checked for simple intent
	history := []chatstore.ChatMessage{
		{Role: "user", Content: "hola"},
		{Role: "sensei", Content: "¡Buenas! ¿En qué te ayudo?"},
		{Role: "user", Content: "¿Cómo implemento concurrencia con goroutines?"},
	}
	result := selectEffectiveModel(
		nil,
		history,
		"fast-model",
		"flash-model",
		"heavy-model",
	)
	if result != "fast-model" {
		t.Errorf("last message is complex, should use fast model, got %q", result)
	}
}

func TestSelectEffectiveModel_EmptyHistory_UsesFast(t *testing.T) {
	// No messages → fast model
	result := selectEffectiveModel(
		nil,
		nil,
		"fast-model",
		"flash-model",
		"heavy-model",
	)
	if result != "fast-model" {
		t.Errorf("empty history with no tools, got %q, want fast-model", result)
	}
}

// --- SendMessageStream tests ---

func TestGenaiProvider_SendMessageStream_RejectsTools(t *testing.T) {
	p := &GenaiProvider{apiKey: "test-key"}

	tools := []domain.ToolDeclaration{{Name: "test_tool"}}
	_, err := p.SendMessageStream(context.Background(), "Sos un sensei.", nil, tools)
	if err == nil {
		t.Fatal("expected error when tools provided to streaming")
	}
	if !strings.Contains(err.Error(), "streaming") || !strings.Contains(err.Error(), "tools") {
		t.Errorf("expected error about streaming with tools, got %q", err.Error())
	}
}

func TestGenaiProvider_SendMessageStream_MissingAPIKey(t *testing.T) {
	p := &GenaiProvider{apiKey: ""}

	_, err := p.SendMessageStream(context.Background(), "Sos un sensei.", nil, nil)
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
	if !strings.Contains(err.Error(), "GEMINI_API_KEY") {
		t.Errorf("expected error mentioning GEMINI_API_KEY, got %q", err.Error())
	}
}

func TestGenaiProvider_SendMessageStream_ContextTimeout(t *testing.T) {
	p := &GenaiProvider{apiKey: "test-key"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	_, err := p.SendMessageStream(ctx, "Sos un sensei.", nil, nil)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	// The error should NOT be the stub "not yet implemented"
	if strings.Contains(err.Error(), "not yet implemented") {
		t.Fatalf("expected context cancellation error, got stub error: %q", err.Error())
	}
}

func TestGenaiProvider_SendMessageStream_ModelSelection_PassesToolsCheck(t *testing.T) {
	// When tools == nil, the method should NOT return "streaming not supported with tools".
	// It should proceed past the tools guard — the error should be about the API key,
	// NOT about tools or "not yet implemented".
	p := &GenaiProvider{apiKey: ""}

	_, err := p.SendMessageStream(context.Background(), "Sos un sensei.", nil, nil)
	if err == nil {
		t.Fatal("expected an error (no API key)")
	}
	if strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("expected real implementation, got stub error: %q", err.Error())
	}
	if strings.Contains(strings.ToLower(err.Error()), "streaming not supported") {
		t.Errorf("expected no tools rejection, got: %q", err.Error())
	}
}
