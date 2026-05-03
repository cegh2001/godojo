package gemini

import (
	"strings"
	"testing"
	"time"

	"godojo/internal/adapters/chatstore"
)

func TestBuildChatRequestBody_OmitsThinkingConfig(t *testing.T) {
	history := []chatstore.ChatMessage{
		{Role: "user", Content: "Hola", Time: time.Now()},
		{Role: "sensei", Content: "Respuesta", Time: time.Now()},
	}

	body := buildChatRequestBody("Eres un sensei de Go.", history)

	if _, ok := body["thinkingConfig"]; ok {
		t.Fatal("request body should not include top-level thinkingConfig")
	}

	generationConfig, ok := body["generationConfig"].(map[string]interface{})
	if !ok {
		t.Fatalf("generationConfig should be a map, got %T", body["generationConfig"])
	}

	if _, ok := generationConfig["thinkingConfig"]; ok {
		t.Fatal("request body should not include generationConfig.thinkingConfig for gemma-4")
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

	body := buildChatRequestBody("Eres un sensei de Go.", history)

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
