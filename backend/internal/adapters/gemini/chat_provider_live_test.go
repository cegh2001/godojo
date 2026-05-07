package gemini_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"godojo/internal/adapters/chatstore"
	"godojo/internal/adapters/gemini"
	"godojo/internal/core/domain"
)

func requireLiveGemini(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping live Gemini smoke tests in short mode")
	}

	if os.Getenv("GODOJO_GEMINI_LIVE_TEST") != "1" {
		t.Skip("set GODOJO_GEMINI_LIVE_TEST=1 to run live Gemini smoke tests")
	}

	if strings.TrimSpace(os.Getenv("GEMINI_API_KEY")) == "" {
		t.Skip("GEMINI_API_KEY not configured")
	}
}

func assertNoGeminiFallbackError(t *testing.T, text string) {
	t.Helper()

	lower := strings.ToLower(text)
	for _, fragment := range []string{
		"error de conexión con el sensei",
		"sensei no disponible",
		"gemini devolvió",
		"revisá la configuración de gemini_api_key",
		"probá de nuevo en unos segundos",
	} {
		if strings.Contains(lower, fragment) {
			t.Fatalf("live Gemini response looks like fallback error text: %q", text)
		}
	}
}

func firstFunctionCall(parts []domain.ContentPart) *domain.FunctionCall {
	for _, part := range parts {
		if part.FunctionCall != nil {
			return part.FunctionCall
		}
	}
	return nil
}

func TestChatProvider_LiveTextSmoke(t *testing.T) {
	requireLiveGemini(t)

	provider := gemini.NewChatProvider()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	parts, err := provider.SendMessage(ctx, "Sos un asistente de pruebas. Respondé breve y exacto.", []chatstore.ChatMessage{
		{Role: "user", Content: "Respondé exactamente con la frase smoke-ok.", Time: time.Now()},
	}, nil)
	if err != nil {
		t.Fatalf("SendMessage() error: %v", err)
	}

	text := strings.TrimSpace(partsToText(parts))
	if text == "" {
		t.Fatal("expected a non-empty live text response")
	}
	assertNoGeminiFallbackError(t, text)
	if !strings.Contains(strings.ToLower(text), "smoke-ok") {
		t.Fatalf("live text response = %q, want it to contain smoke-ok", text)
	}
}

func TestChatProvider_LiveInteractionsSmoke(t *testing.T) {
	requireLiveGemini(t)

	provider := gemini.NewChatProvider()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tools := []domain.ToolDeclaration{
		{
			Name:        "ping_tool",
			Description: "Devuelve el valor echo recibido para pruebas de integración.",
			Parameters: domain.ToolParameters{
				Type: "OBJECT",
				Properties: map[string]domain.ToolProperty{
					"echo": {Type: "STRING", Description: "Texto de prueba a devolver"},
				},
				Required: []string{"echo"},
			},
		},
	}

	systemPrompt := "Sos un agente de pruebas. Cuando el usuario pida usar ping_tool, llamá exactamente esa herramienta una vez y, después de recibir el resultado, respondé en texto plano incluyendo el valor echo."
	history := []chatstore.ChatMessage{
		{Role: "user", Content: "Usá ping_tool con echo=smoke-ok y después confirmá el resultado en una frase corta.", Time: time.Now()},
	}

	parts, err := provider.SendMessage(ctx, systemPrompt, history, tools)
	if err != nil {
		t.Fatalf("SendMessage() error: %v", err)
	}

	call := firstFunctionCall(parts)
	if call == nil {
		t.Fatalf("expected a function call from live interactions path, got parts: %+v", parts)
	}
	if call.ID == "" {
		t.Fatal("expected live function call to include a non-empty call ID")
	}
	if call.Name != "ping_tool" {
		t.Fatalf("function name = %q, want ping_tool", call.Name)
	}

	finalParts, err := provider.SendFunctionResponse(ctx, history, call.ID, call.Name, map[string]interface{}{
		"echo":    "smoke-ok",
		"success": true,
	})
	if err != nil {
		t.Fatalf("SendFunctionResponse() error: %v", err)
	}

	if nextCall := firstFunctionCall(finalParts); nextCall != nil {
		t.Fatalf("expected final text after function_result, got another function call: %+v", nextCall)
	}

	text := strings.TrimSpace(partsToText(finalParts))
	if text == "" {
		t.Fatal("expected a non-empty final text after live function_result")
	}
	assertNoGeminiFallbackError(t, text)
	if !strings.Contains(strings.ToLower(text), "smoke-ok") {
		t.Fatalf("live final response = %q, want it to contain smoke-ok", text)
	}
}
