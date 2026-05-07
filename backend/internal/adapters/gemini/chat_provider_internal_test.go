package gemini

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"godojo/internal/adapters/chatstore"
	"godojo/internal/core/domain"
)

func TestBuildChatRequestBody_OmitsThinkingConfig(t *testing.T) {
	history := []chatstore.ChatMessage{
		{Role: "user", Content: "Hola", Time: time.Now()},
		{Role: "sensei", Content: "Respuesta", Time: time.Now()},
	}

	body := buildChatRequestBody("Eres un sensei de Go.", history, nil)

	if _, ok := body["thinkingConfig"]; ok {
		t.Fatal("request body should not include top-level thinkingConfig")
	}

	generationConfig, ok := body["generationConfig"].(map[string]interface{})
	if !ok {
		t.Fatalf("generationConfig should be a map, got %T", body["generationConfig"])
	}

	// Now thinkingConfig SHOULD be present in generationConfig
	tc, ok := generationConfig["thinkingConfig"].(map[string]interface{})
	if !ok {
		t.Fatal("generationConfig.thinkingConfig should be present for gemma-4 function calling")
	}
	if level, ok := tc["thinkingLevel"]; !ok || level != "HIGH" {
		t.Fatalf("thinkingLevel should be HIGH, got %v", level)
	}

	contents, ok := body["contents"].([]map[string]interface{})
	if !ok {
		t.Fatalf("contents should be a slice of maps, got %T", body["contents"])
	}

	if got := len(contents); got != 2 {
		t.Fatalf("expected 2 contents, got %d", got)
	}

	if got := contents[0]["role"]; got != "user" {
		t.Fatalf("first message should map to user, got %v", got)
	}

	if got := contents[1]["role"]; got != "model" {
		t.Fatalf("sensei messages should map to model, got %v", got)
	}
}

func TestBuildChatRequestBody_CompactsLongHistory(t *testing.T) {
	longContent := strings.Repeat("Esto es un mensaje largo del sensei que no debería viajar completo. ", 12)
	history := []chatstore.ChatMessage{
		{Role: "user", Content: "Hola", Time: time.Now()},
		{Role: "sensei", Content: longContent, Time: time.Now()},
		{Role: "user", Content: "Quiero ver fundamentos", Time: time.Now()},
		{Role: "sensei", Content: "Vamos con fundamentos", Time: time.Now()},
		{Role: "user", Content: "Seguimos", Time: time.Now()},
		{Role: "sensei", Content: "Sí", Time: time.Now()},
		{Role: "user", Content: "Otra pregunta", Time: time.Now()},
	}

	body := buildChatRequestBody("Eres un sensei de Go.", history, nil)

	systemInstruction, ok := body["system_instruction"].(map[string]interface{})
	if !ok {
		t.Fatalf("system_instruction should be a map, got %T", body["system_instruction"])
	}
	parts, ok := systemInstruction["parts"].([]map[string]interface{})
	if !ok {
		t.Fatalf("system_instruction.parts should be a slice of maps, got %T", systemInstruction["parts"])
	}
	text, ok := parts[0]["text"].(string)
	if !ok {
		t.Fatalf("system instruction text should be a string, got %T", parts[0]["text"])
	}

	if !strings.Contains(text, "Resumen breve del contexto previo") {
		t.Fatal("long history should add a compact context summary to the system instruction")
	}

	contents, ok := body["contents"].([]map[string]interface{})
	if !ok {
		t.Fatalf("contents should be a slice of maps, got %T", body["contents"])
	}

	if got := len(contents); got != maxChatContextMessages {
		t.Fatalf("expected %d recent messages, got %d", maxChatContextMessages, got)
	}

	firstParts, ok := contents[0]["parts"].([]map[string]interface{})
	if !ok {
		t.Fatalf("contents[0].parts should be a slice of maps, got %T", contents[0]["parts"])
	}
	firstText, ok := firstParts[0]["text"].(string)
	if !ok {
		t.Fatalf("contents[0].parts[0].text should be a string, got %T", firstParts[0]["text"])
	}

	if len([]rune(firstText)) > maxChatMessagePromptRunes {
		t.Fatalf("prompt text should be truncated to %d runes, got %d", maxChatMessagePromptRunes, len([]rune(firstText)))
	}
}

// --- New tests for T09/T10: tools, thinkingConfig, functionCall parsing ---

func TestBuildToolsJSON_FormatsFunctionDeclarations(t *testing.T) {
	tools := []domain.ToolDeclaration{
		{
			Name:        "create_file",
			Description: "Crea un archivo Go",
			Parameters: domain.ToolParameters{
				Type: "OBJECT",
				Properties: map[string]domain.ToolProperty{
					"filename": {Type: "STRING", Description: "Nombre del archivo"},
					"content":  {Type: "STRING", Description: "Contenido"},
				},
				Required: []string{"filename", "content"},
			},
		},
	}

	result := buildToolsJSON(tools)

	if len(result) != 2 {
		t.Fatalf("expected 2 tool entries (functionDeclarations + googleSearch), got %d", len(result))
	}

	// First entry should have functionDeclarations
	fdRaw, ok := result[0]["functionDeclarations"]
	if !ok {
		t.Fatal("first tool entry should have functionDeclarations")
	}
	fds, ok := fdRaw.([]map[string]interface{})
	if !ok {
		t.Fatalf("functionDeclarations should be slice of maps, got %T", fdRaw)
	}
	if len(fds) != 1 {
		t.Fatalf("expected 1 function declaration, got %d", len(fds))
	}
	fd := fds[0]
	if fd["name"] != "create_file" {
		t.Errorf("function name = %v, want create_file", fd["name"])
	}
	if fd["description"] != "Crea un archivo Go" {
		t.Errorf("function description = %v", fd["description"])
	}

	// Second entry: googleSearch
	if _, ok := result[1]["googleSearch"]; !ok {
		t.Error("second tool entry should be googleSearch")
	}
}

func TestBuildToolsJSON_NilReturnsOnlyBuiltins(t *testing.T) {
	result := buildToolsJSON(nil)

	if len(result) != 0 {
		t.Fatalf("nil tools should return empty slice, got %d entries", len(result))
	}
}

func TestBuildToolsJSON_EmptyReturnsOnlyBuiltins(t *testing.T) {
	result := buildToolsJSON([]domain.ToolDeclaration{})

	if len(result) != 0 {
		t.Fatalf("empty tools should return empty slice, got %d entries", len(result))
	}
}

func TestBuildChatRequestBody_IncludesToolsWhenProvided(t *testing.T) {
	history := []chatstore.ChatMessage{
		{Role: "user", Content: "Crea un archivo", Time: time.Now()},
	}

	tools := []domain.ToolDeclaration{
		{
			Name:        "create_file",
			Description: "Crea un archivo",
			Parameters: domain.ToolParameters{
				Type: "OBJECT",
				Properties: map[string]domain.ToolProperty{
					"filename": {Type: "STRING", Description: "Nombre"},
				},
				Required: []string{"filename"},
			},
		},
	}

	body := buildChatRequestBody("Sos un sensei.", history, tools)

	toolsRaw, ok := body["tools"]
	if !ok {
		t.Fatal("request body should include tools key when tools provided")
	}
	toolsArr, ok := toolsRaw.([]map[string]interface{})
	if !ok {
		t.Fatalf("tools should be slice of maps, got %T", toolsRaw)
	}
	if len(toolsArr) == 0 {
		t.Fatal("tools array should not be empty")
	}
}

func TestBuildChatRequestBody_ExcludesToolsWhenNil(t *testing.T) {
	history := []chatstore.ChatMessage{
		{Role: "user", Content: "Hola", Time: time.Now()},
	}

	body := buildChatRequestBody("Sos un sensei.", history, nil)

	if _, ok := body["tools"]; ok {
		t.Fatal("request body should NOT include tools key when tools is nil")
	}
}

func TestBuildChatRequestBody_IncludesThinkingConfig(t *testing.T) {
	body := buildChatRequestBody("prompt", nil, nil)

	genConfig, ok := body["generationConfig"].(map[string]interface{})
	if !ok {
		t.Fatalf("generationConfig missing or wrong type: %T", body["generationConfig"])
	}

	tc, ok := genConfig["thinkingConfig"].(map[string]interface{})
	if !ok {
		t.Fatal("generationConfig.thinkingConfig should be present")
	}
	level, ok := tc["thinkingLevel"]
	if !ok || level != "HIGH" {
		t.Fatalf("thinkingLevel should be HIGH, got %v", level)
	}
}

func TestExtractContentParts_TextOnly(t *testing.T) {
	parts := []map[string]interface{}{
		{"text": "Hola, ¿cómo estás?"},
		{"text": "Vamos a aprender Go."},
	}

	result := extractContentParts(parts)

	if len(result) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(result))
	}
	if result[0].Text != "Hola, ¿cómo estás?" {
		t.Errorf("first part text = %q", result[0].Text)
	}
	if result[0].FunctionCall != nil {
		t.Error("first part should not have functionCall")
	}
	if result[1].Text != "Vamos a aprender Go." {
		t.Errorf("second part text = %q", result[1].Text)
	}
}

func TestExtractContentParts_SkipsThoughtParts(t *testing.T) {
	parts := []map[string]interface{}{
		{"text": "Voy a pensar paso a paso...", "thought": true},
		{"text": "Respuesta real"},
	}

	result := extractContentParts(parts)

	if len(result) != 1 {
		t.Fatalf("expected 1 content part (thought skipped), got %d: %+v", len(result), result)
	}
	if result[0].Text != "Respuesta real" {
		t.Errorf("text = %q, want 'Respuesta real'", result[0].Text)
	}
}

func TestExtractContentParts_ParsesFunctionCall(t *testing.T) {
	parts := []map[string]interface{}{
		{
			"functionCall": map[string]interface{}{
				"name": "create_file",
				"args": map[string]interface{}{
					"filename": "hola.go",
					"content":  "package main",
				},
			},
		},
	}

	result := extractContentParts(parts)

	if len(result) != 1 {
		t.Fatalf("expected 1 content part, got %d", len(result))
	}
	if result[0].FunctionCall == nil {
		t.Fatal("expected FunctionCall to be non-nil")
	}
	if result[0].FunctionCall.Name != "create_file" {
		t.Errorf("function name = %q, want create_file", result[0].FunctionCall.Name)
	}
	if result[0].FunctionCall.Args["filename"] != "hola.go" {
		t.Errorf("args filename = %v", result[0].FunctionCall.Args["filename"])
	}
	if result[0].FunctionCall.Args["content"] != "package main" {
		t.Errorf("args content = %v", result[0].FunctionCall.Args["content"])
	}
}

func TestExtractContentParts_MixedTextAndFunctionCall(t *testing.T) {
	parts := []map[string]interface{}{
		{"text": "Voy a crear el archivo", "thought": true},
		{"text": "Claro, aquí va:"},
		{
			"functionCall": map[string]interface{}{
				"name": "create_file",
				"args": map[string]interface{}{"filename": "ejemplo.go"},
			},
		},
		{"text": "El archivo está listo."},
	}

	result := extractContentParts(parts)

	if len(result) != 3 {
		t.Fatalf("expected 3 content parts (thought skipped), got %d", len(result))
	}
	// Part 0: text "Claro, aquí va:"
	if result[0].Text != "Claro, aquí va:" {
		t.Errorf("first text = %q", result[0].Text)
	}
	// Part 1: functionCall
	if result[1].FunctionCall == nil {
		t.Fatal("second part should be functionCall")
	}
	if result[1].FunctionCall.Name != "create_file" {
		t.Errorf("function name = %q", result[1].FunctionCall.Name)
	}
	// Part 2: text "El archivo está listo."
	if result[2].Text != "El archivo está listo." {
		t.Errorf("last text = %q", result[2].Text)
	}
}

func TestFormatGeminiAPIError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantParts  []string
		avoidParts []string
	}{
		{
			name:       "internal 500 from gemini is explained without raw json",
			statusCode: 500,
			body: `{
				"error": {
					"code": 500,
					"message": "Internal error encountered.",
					"status": "INTERNAL"
				}
			}`,
			wantParts: []string{
				"Gemini devolvió INTERNAL (500)",
				"Internal error encountered.",
				"fallo interno del proveedor",
				"Probá de nuevo en unos segundos.",
			},
			avoidParts: []string{"\"error\"", "{", "}"},
		},
		{
			name:       "fallback keeps useful preview when body is not json",
			statusCode: 502,
			body:       "upstream exploded badly",
			wantParts: []string{
				"Gemini devolvió HTTP_502 (502)",
				"upstream exploded badly",
				"fallo interno del proveedor",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatGeminiAPIError(tt.statusCode, []byte(tt.body))

			for _, want := range tt.wantParts {
				if !strings.Contains(got, want) {
					t.Errorf("formatted message = %q, want substring %q", got, want)
				}
			}

			for _, avoid := range tt.avoidParts {
				if strings.Contains(got, avoid) {
					t.Errorf("formatted message = %q, should not contain %q", got, avoid)
				}
			}
		})
	}
}

func TestChatProvider_DoGenerateContentRequest_RetriesRetryableStatuses(t *testing.T) {
	tests := []struct {
		name         string
		statuses     []int
		wantAttempts int32
		wantStatus   int
		wantBodyPart string
	}{
		{
			name:         "retries once on internal error",
			statuses:     []int{http.StatusInternalServerError, http.StatusOK},
			wantAttempts: 2,
			wantStatus:   http.StatusOK,
			wantBodyPart: "ok after retry",
		},
		{
			name:         "retries once on rate limit",
			statuses:     []int{http.StatusTooManyRequests, http.StatusOK},
			wantAttempts: 2,
			wantStatus:   http.StatusOK,
			wantBodyPart: "ok after retry",
		},
		{
			name:         "does not retry non retryable client error",
			statuses:     []int{http.StatusBadRequest},
			wantAttempts: 1,
			wantStatus:   http.StatusBadRequest,
			wantBodyPart: "bad request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attempts int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				current := int(atomic.AddInt32(&attempts, 1)) - 1
				status := tt.statuses[current]

				w.Header().Set("Content-Type", "application/json")
				if status == http.StatusTooManyRequests {
					w.Header().Set("Retry-After", "0")
				}
				w.WriteHeader(status)

				switch status {
				case http.StatusOK:
					_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok after retry"}]}}]}`))
				case http.StatusBadRequest:
					_, _ = w.Write([]byte(`bad request`))
				default:
					_, _ = w.Write([]byte(`{"error":{"code":500,"message":"retry me","status":"INTERNAL"}}`))
				}
			}))
			defer server.Close()

			provider := &ChatProvider{
				apiKey:     "test-key",
				timeout:    time.Second,
				httpClient: server.Client(),
			}

			respBody, statusCode, err := provider.doGenerateContentRequest(context.Background(), time.Second, server.URL, map[string]interface{}{
				"contents": []map[string]interface{}{{"role": "user"}},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if statusCode != tt.wantStatus {
				t.Fatalf("statusCode = %d, want %d", statusCode, tt.wantStatus)
			}
			if atomic.LoadInt32(&attempts) != tt.wantAttempts {
				t.Fatalf("attempts = %d, want %d", attempts, tt.wantAttempts)
			}
			if !strings.Contains(string(respBody), tt.wantBodyPart) {
				t.Fatalf("response body = %q, want substring %q", string(respBody), tt.wantBodyPart)
			}
		})
	}
}

func TestExtractContentParts_EmptyParts(t *testing.T) {
	result := extractContentParts(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 parts for nil input, got %d", len(result))
	}

	result = extractContentParts([]map[string]interface{}{})
	if len(result) != 0 {
		t.Errorf("expected 0 parts for empty input, got %d", len(result))
	}
}

// TestBuildFunctionResponseMessage verifies the function response message shape.
func TestBuildFunctionResponseMessage_Format(t *testing.T) {
	msg := buildFunctionResponseMessage("call-123", "create_file", map[string]interface{}{
		"filename": "hola.go",
		"success":  true,
	})

	if msg["role"] != "user" {
		t.Errorf("role = %v, want user", msg["role"])
	}

	parts, ok := msg["parts"].([]map[string]interface{})
	if !ok {
		t.Fatalf("parts should be slice of maps, got %T", msg["parts"])
	}
	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}

	fr, ok := parts[0]["functionResponse"].(map[string]interface{})
	if !ok {
		t.Fatalf("part should contain functionResponse, got %v", parts[0])
	}
	if fr["name"] != "create_file" {
		t.Errorf("functionResponse name = %v, want create_file", fr["name"])
	}
	response, ok := fr["response"].(map[string]interface{})
	if !ok {
		t.Fatalf("response should be a map, got %T", fr["response"])
	}
	if response["filename"] != "hola.go" {
		t.Errorf("response filename = %v", response["filename"])
	}
}
