package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"godojo/internal/adapters/chatstore"
	"godojo/internal/core/domain"
)

func TestBuildChatRequestBody_UsesFastDefaults(t *testing.T) {
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

	if _, ok := generationConfig["thinkingConfig"]; ok {
		t.Fatal("Gemma 4 generateContent requests should omit thinkingConfig by default")
	}
	if tokens, ok := generationConfig["maxOutputTokens"]; !ok || tokens != defaultChatMaxOutputTokens {
		t.Fatalf("maxOutputTokens should be %d, got %v", defaultChatMaxOutputTokens, tokens)
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

	if len(result) != 1 {
		t.Fatalf("expected 1 tool entry (functionDeclarations), got %d", len(result))
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

	if _, ok := result[0]["googleSearch"]; ok {
		t.Error("googleSearch should be opt-in for local tool usage")
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

func TestBuildChatRequestBody_OmitsThinkingConfigForGemma4(t *testing.T) {
	body := buildChatRequestBody("prompt", nil, nil)

	genConfig, ok := body["generationConfig"].(map[string]interface{})
	if !ok {
		t.Fatalf("generationConfig missing or wrong type: %T", body["generationConfig"])
	}

	if _, ok := genConfig["thinkingConfig"]; ok {
		t.Fatal("Gemma 4 generateContent requests should omit thinkingConfig")
	}
}

func TestBuildThinkingConfig_OmitsConfigForGemma4(t *testing.T) {
	config := buildThinkingConfig(defaultHeavyModel, 0)

	if len(config) != 0 {
		t.Fatalf("config = %#v, want empty map for Gemma 4", config)
	}
}

func TestBuildInteractionRequestBody_ExcludesGenerationConfig(t *testing.T) {
	body := buildInteractionRequestBody("Sos un sensei.", []chatstore.ChatMessage{{Role: "user", Content: "Hola", Time: time.Now()}}, []domain.ToolDeclaration{{
		Name:        "create_file",
		Description: "Crea un archivo",
		Parameters: domain.ToolParameters{
			Type:       "OBJECT",
			Properties: map[string]domain.ToolProperty{},
		},
	}}, defaultChatRequestConfig())

	if _, ok := body["generationConfig"]; ok {
		t.Fatal("interactions request body should not include generationConfig")
	}
	if _, ok := body["model"]; !ok {
		t.Fatal("interactions request body should include model")
	}
	if _, ok := body["input"]; !ok {
		t.Fatal("interactions request body should include input")
	}
}

func TestBuildInteractionFunctionResultInput_NormalizesStructuredResult(t *testing.T) {
	input := buildInteractionFunctionResultInput("call-1", "create_file", map[string]interface{}{
		"success": true,
		"path":    "hola.go",
	})

	if len(input) != 1 {
		t.Fatalf("expected 1 input entry, got %d", len(input))
	}
	result, ok := input[0]["result"].(string)
	if !ok {
		t.Fatalf("result should be normalized to string, got %T", input[0]["result"])
	}
	if !strings.Contains(result, `"success":true`) {
		t.Fatalf("result = %q, want serialized JSON", result)
	}
}

func TestBuildToolsJSONWithGoogleSearch_OptIn(t *testing.T) {
	tools := []domain.ToolDeclaration{
		{
			Name:        "read_roadmap_section",
			Description: "Lee una sección del roadmap",
			Parameters: domain.ToolParameters{
				Type:       "OBJECT",
				Properties: map[string]domain.ToolProperty{},
			},
		},
	}

	result := buildToolsJSONWithGoogleSearch(tools, true)
	if len(result) != 2 {
		t.Fatalf("expected functionDeclarations + googleSearch, got %d entries", len(result))
	}
	if _, ok := result[1]["googleSearch"]; !ok {
		t.Fatal("googleSearch entry should be present when explicitly enabled")
	}
}

func TestLoadChatRequestConfigFromEnv(t *testing.T) {
	t.Setenv("GODOJO_SENSEI_MODEL", defaultHeavyModel)
	t.Setenv(senseiFastModelEnv, defaultFastModel)
	t.Setenv("GODOJO_SENSEI_MAX_OUTPUT_TOKENS", "512")
	t.Setenv("GODOJO_SENSEI_THINKING_BUDGET", "0")
	t.Setenv("GODOJO_SENSEI_ENABLE_GOOGLE_SEARCH", "true")

	cfg := loadChatRequestConfigFromEnv()

	if cfg.model != defaultHeavyModel {
		t.Fatalf("model = %q", cfg.model)
	}
	if cfg.textModel != defaultFastModel {
		t.Fatalf("textModel = %q", cfg.textModel)
	}
	if cfg.maxOutputTokens != 512 {
		t.Fatalf("maxOutputTokens = %d", cfg.maxOutputTokens)
	}
	if cfg.thinkingBudget != 0 {
		t.Fatalf("thinkingBudget = %d", cfg.thinkingBudget)
	}
	if !cfg.enableGoogleSearch {
		t.Fatal("googleSearch should be enabled from env")
	}
}

func TestLoadChatRequestConfigFromEnv_UsesHeavyModelEnvFallback(t *testing.T) {
	t.Setenv(senseiModelEnv, "")
	t.Setenv(senseiHeavyModelEnv, defaultHeavyModel)

	cfg := loadChatRequestConfigFromEnv()

	if cfg.model != defaultHeavyModel {
		t.Fatalf("model = %q, want %q", cfg.model, defaultHeavyModel)
	}
}

func TestLoadChatRequestConfigFromEnv_UsesFastModelEnvFallback(t *testing.T) {
	t.Setenv(senseiFastModelEnv, "custom-fast-model")

	cfg := loadChatRequestConfigFromEnv()

	if cfg.textModel != "custom-fast-model" {
		t.Fatalf("textModel = %q, want %q", cfg.textModel, "custom-fast-model")
	}
}

func TestResolveChatModelFromEnv_PrefersExplicitChatOverride(t *testing.T) {
	t.Setenv(senseiHeavyModelEnv, defaultHeavyModel)
	t.Setenv(senseiModelEnv, "custom-chat-model")

	if model := resolveChatModelFromEnv(); model != "custom-chat-model" {
		t.Fatalf("model = %q, want %q", model, "custom-chat-model")
	}
}

func TestResolveFastModelFromEnv_PrefersExplicitFastOverride(t *testing.T) {
	t.Setenv(senseiFastModelEnv, "custom-fast-model")

	if model := resolveFastModelFromEnv(); model != "custom-fast-model" {
		t.Fatalf("model = %q, want %q", model, "custom-fast-model")
	}
}

func TestLoadSenseiTimeoutsFromEnv_UsesDefaults(t *testing.T) {
	t.Setenv(senseiTimeoutEnv, "")
	t.Setenv(senseiToolTimeoutEnv, "")

	timeout, toolTimeout := loadSenseiTimeoutsFromEnv()

	if timeout != defaultSenseiTimeout {
		t.Fatalf("timeout = %s, want %s", timeout, defaultSenseiTimeout)
	}
	if toolTimeout != defaultSenseiToolTimeout {
		t.Fatalf("toolTimeout = %s, want %s", toolTimeout, defaultSenseiToolTimeout)
	}
}

func TestLoadSenseiTimeoutsFromEnv_UsesOverrides(t *testing.T) {
	t.Setenv(senseiTimeoutEnv, "35")
	t.Setenv(senseiToolTimeoutEnv, "75")

	timeout, toolTimeout := loadSenseiTimeoutsFromEnv()

	if timeout != 35*time.Second {
		t.Fatalf("timeout = %s, want %s", timeout, 35*time.Second)
	}
	if toolTimeout != 75*time.Second {
		t.Fatalf("toolTimeout = %s, want %s", toolTimeout, 75*time.Second)
	}
}

func TestLoadSenseiTimeoutsFromEnv_KeepsToolTimeoutAtLeastTextTimeout(t *testing.T) {
	t.Setenv(senseiTimeoutEnv, "40")
	t.Setenv(senseiToolTimeoutEnv, "10")

	timeout, toolTimeout := loadSenseiTimeoutsFromEnv()

	if timeout != 40*time.Second {
		t.Fatalf("timeout = %s, want %s", timeout, 40*time.Second)
	}
	if toolTimeout != 40*time.Second {
		t.Fatalf("toolTimeout = %s, want %s", toolTimeout, 40*time.Second)
	}
}

func TestFormatGeminiTransportError_DeadlineExceeded(t *testing.T) {
	msg := formatGeminiTransportError(context.DeadlineExceeded)

	if !strings.Contains(msg, "tardó demasiado") {
		t.Fatalf("message = %q", msg)
	}
	if strings.Contains(strings.ToLower(msg), "revisaste tu conexión") {
		t.Fatalf("message should not blame the connection: %q", msg)
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
				"id":   "call-123",
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
	if result[0].FunctionCall.ID != "call-123" {
		t.Errorf("function ID = %q, want call-123", result[0].FunctionCall.ID)
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
		{
			name:       "rate limit with string code is parsed and explained",
			statusCode: 429,
			body: `{
				"error": {
					"message": "You do not have enough quota to make this request.",
					"code": "too_many_requests"
				}
			}`,
			wantParts: []string{
				"Gemini devolvió TOO_MANY_REQUESTS (429)",
				"You do not have enough quota to make this request.",
				"límite de cuota o rate limit",
			},
			avoidParts: []string{"\"error\"", "{"},
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

func TestParseInteractionResponse_FunctionCallAndText(t *testing.T) {
	respBody := []byte(`{
		"id": "interaction-123",
		"outputs": [
			{"type": "function_call", "name": "create_file", "arguments": {"filename": "hola.go"}, "id": "call-123"},
			{"type": "text", "text": "Listo"}
		]
	}`)

	parts, interactionID, err := parseInteractionResponse(respBody)
	if err != nil {
		t.Fatalf("parseInteractionResponse error: %v", err)
	}
	if interactionID != "interaction-123" {
		t.Fatalf("interactionID = %q", interactionID)
	}
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if parts[0].FunctionCall == nil {
		t.Fatal("first part should be a function call")
	}
	if parts[0].FunctionCall.ID != "call-123" {
		t.Fatalf("function call ID = %q", parts[0].FunctionCall.ID)
	}
	if parts[1].Text != "Listo" {
		t.Fatalf("text part = %q", parts[1].Text)
	}
}

func TestChatProvider_SendMessage_UsesInteractionsForTools(t *testing.T) {
	type capturedRequest struct {
		Path string
		Body map[string]interface{}
	}

	var captured capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		captured.Path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&captured.Body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"interaction-1","outputs":[{"type":"function_call","name":"create_file","arguments":{"filename":"hola.go"},"id":"call-1"}]}`))
	}))
	defer server.Close()

	provider := &ChatProvider{
		apiKey:        "test-key",
		timeout:       time.Second,
		toolTimeout:   time.Second,
		httpClient:    redirectedHTTPClient(t, server.URL),
		requestConfig: defaultChatRequestConfig(),
		callContexts:  make(map[string]interactionContext),
	}

	parts, err := provider.SendMessage(context.Background(), "Sos un sensei.", []chatstore.ChatMessage{{Role: "user", Content: "Creá un archivo", Time: time.Now()}}, []domain.ToolDeclaration{{
		Name:        "create_file",
		Description: "Crea un archivo",
		Parameters: domain.ToolParameters{
			Type: "OBJECT",
			Properties: map[string]domain.ToolProperty{
				"filename": {Type: "STRING", Description: "Nombre"},
			},
			Required: []string{"filename"},
		},
	}})
	if err != nil {
		t.Fatalf("SendMessage error: %v", err)
	}
	if captured.Path != "/v1beta/interactions" {
		t.Fatalf("request path = %q", captured.Path)
	}
	if _, ok := captured.Body["generationConfig"]; ok {
		t.Fatal("interactions request should not send generationConfig")
	}
	if len(parts) != 1 || parts[0].FunctionCall == nil {
		t.Fatalf("expected 1 function call part, got %+v", parts)
	}
	if parts[0].FunctionCall.ID != "call-1" {
		t.Fatalf("function call ID = %q", parts[0].FunctionCall.ID)
	}
	ctxData, ok := provider.takeInteractionContext("call-1")
	if !ok {
		t.Fatal("interaction context should be stored for call-1")
	}
	if ctxData.interactionID != "interaction-1" {
		t.Fatalf("interactionID = %q", ctxData.interactionID)
	}
	toolsRaw, ok := captured.Body["tools"].([]interface{})
	if !ok || len(toolsRaw) != 1 {
		t.Fatalf("tools = %#v", captured.Body["tools"])
	}
}

func TestChatProvider_SendMessage_UsesFastModelWithoutTools(t *testing.T) {
	type capturedRequest struct {
		Path string
		Body map[string]interface{}
	}

	var captured capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		captured.Path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&captured.Body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"respuesta rápida"}]}}]}`))
	}))
	defer server.Close()

	provider := &ChatProvider{
		apiKey:     "test-key",
		timeout:    time.Second,
		httpClient: redirectedHTTPClient(t, server.URL),
		requestConfig: chatRequestConfig{
			model:              defaultHeavyModel,
			textModel:          defaultFastModel,
			maxOutputTokens:    defaultChatMaxOutputTokens,
			thinkingBudget:     defaultChatThinkingBudget,
			enableGoogleSearch: false,
		},
		callContexts: make(map[string]interactionContext),
	}

	parts, err := provider.SendMessage(context.Background(), "Sos un sensei.", []chatstore.ChatMessage{{Role: "user", Content: "Hola", Time: time.Now()}}, nil)
	if err != nil {
		t.Fatalf("SendMessage error: %v", err)
	}
	if captured.Path != "/v1beta/models/"+defaultFastModel+":generateContent" {
		t.Fatalf("request path = %q", captured.Path)
	}
	if len(parts) != 1 || parts[0].Text != "respuesta rápida" {
		t.Fatalf("parts = %+v", parts)
	}
	if _, ok := captured.Body["tools"]; ok {
		t.Fatal("text-only request should not include tools")
	}
}

func TestChatProvider_SendFunctionResponse_UsesInteractionsContinuation(t *testing.T) {
	type capturedRequest struct {
		Path string
		Body map[string]interface{}
	}

	var captured capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		captured.Path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&captured.Body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"interaction-2","outputs":[{"type":"text","text":"Archivo creado"}]}`))
	}))
	defer server.Close()

	provider := &ChatProvider{
		apiKey:        "test-key",
		timeout:       time.Second,
		toolTimeout:   time.Second,
		httpClient:    redirectedHTTPClient(t, server.URL),
		requestConfig: defaultChatRequestConfig(),
		callContexts: map[string]interactionContext{
			"call-1": {
				interactionID: "interaction-1",
				systemPrompt:  "Sos un sensei.",
				tools: []domain.ToolDeclaration{{
					Name:        "create_file",
					Description: "Crea un archivo",
					Parameters:  domain.ToolParameters{Type: "OBJECT"},
				}},
			},
		},
	}

	parts, err := provider.SendFunctionResponse(context.Background(), nil, "call-1", "create_file", map[string]interface{}{"success": true})
	if err != nil {
		t.Fatalf("SendFunctionResponse error: %v", err)
	}
	if captured.Path != "/v1beta/interactions" {
		t.Fatalf("request path = %q", captured.Path)
	}
	if _, ok := captured.Body["generationConfig"]; ok {
		t.Fatal("interaction continuation should not send generationConfig")
	}
	if captured.Body["previous_interaction_id"] != "interaction-1" {
		t.Fatalf("previous_interaction_id = %#v", captured.Body["previous_interaction_id"])
	}
	inputRaw, ok := captured.Body["input"].([]interface{})
	if !ok || len(inputRaw) != 1 {
		t.Fatalf("input = %#v", captured.Body["input"])
	}
	inputEntry, ok := inputRaw[0].(map[string]interface{})
	if !ok {
		t.Fatalf("input[0] = %#v", inputRaw[0])
	}
	if inputEntry["call_id"] != "call-1" {
		t.Fatalf("call_id = %#v", inputEntry["call_id"])
	}
	result, ok := inputEntry["result"].(string)
	if !ok {
		t.Fatalf("result = %#v", inputEntry["result"])
	}
	if !strings.Contains(result, `"success":true`) {
		t.Fatalf("result = %q", result)
	}
	if len(parts) != 1 || parts[0].Text != "Archivo creado" {
		t.Fatalf("parts = %+v", parts)
	}
	if _, ok := provider.takeInteractionContext("call-1"); ok {
		t.Fatal("call-1 context should be consumed after continuation")
	}
}

func redirectedHTTPClient(t *testing.T, serverURL string) *http.Client {
	t.Helper()
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	original := http.DefaultTransport
	return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		clone := req.Clone(req.Context())
		clone.URL.Scheme = parsedURL.Scheme
		clone.URL.Host = parsedURL.Host
		return original.RoundTrip(clone)
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
