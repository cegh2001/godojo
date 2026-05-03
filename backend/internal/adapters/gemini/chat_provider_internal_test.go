package gemini

import (
	"testing"
	"time"

	"godojo/internal/adapters/chatstore"
)

func TestBuildChatRequestBody_OmitsThinkingConfig(t *testing.T) {
	history := []chatstore.ChatMessage{
		{Role: "user", Content: "Hola", Time: time.Now()},
		{Role: "sensei", Content: "Respuesta", Time: time.Now()},
	}

	contents := buildChatContents(history)
	body := buildChatRequestBody("Eres un sensei de Go.", contents)

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
