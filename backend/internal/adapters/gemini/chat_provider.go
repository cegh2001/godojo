package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"godojo/internal/adapters/chatstore"
)

// ChatProvider handles chat conversations with Gemma via the Gemini API.
// Uses system_instruction for persona and contents for conversation history.
// See: https://ai.google.dev/gemini-api/docs/system-instructions
type ChatProvider struct {
	apiKey  string
	timeout time.Duration
}

// NewChatProvider creates a ChatProvider using the GEMINI_API_KEY environment variable.
func NewChatProvider() *ChatProvider {
	return &ChatProvider{
		apiKey:  os.Getenv("GEMINI_API_KEY"),
		timeout: 30 * time.Second, // longer timeout for chat
	}
}

// System prompt for the sensei — neutral Latin American Spanish.
const senseiSystemPrompt = `Eres un sensei experto en el lenguaje de programación Go (Golang), parte de GoDojo. 
Ayudas a estudiantes a aprender a programar en Go desde cero.
Conoces el roadmap de GoDojo (7 fases, 23 ejercicios): Fundamentos, Estructuras de Datos, Punteros, 
Métodos e Interfaces, Manejo de Errores, Concurrencia, y Standard Library.
Puedes explicar conceptos de programación, dar ejemplos de código, sugerir ejercicios, y responder preguntas. 
Usa español neutro latinoamericano. Sé didáctico pero conciso.
Si te preguntan algo que no sabes, dilo con honestidad.
NO eres un sensei del juego de mesa Go (weiqi/baduk). Eres un sensei de Golang.`

// SendMessage sends a message to the Gemini API and returns the response.
// Uses system_instruction for the sensei persona and contents for conversation.
// Based on Gemini REST API spec: https://ai.google.dev/gemini-api/docs/system-instructions
func (p *ChatProvider) SendMessage(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("Sensei no disponible — configura GEMINI_API_KEY en .env")
	}

	if systemPrompt == "" {
		systemPrompt = senseiSystemPrompt
	}

	// Build conversation contents (user/model alternating)
	// First message: the user's latest input is NOT here — it was already added to history
	// by the TUI before calling this. So we just send the full history.
	contents := make([]map[string]interface{}, 0, len(history))
	for _, msg := range history {
		role := "user"
		if msg.Role == "sensei" {
			role = "model"
		}
		contents = append(contents, map[string]interface{}{
			"role": role,
			"parts": []map[string]interface{}{
				{"text": msg.Content},
			},
		})
	}

	// If no history (first message), start fresh
	if len(contents) == 0 {
		contents = append(contents, map[string]interface{}{
			"role": "user",
			"parts": []map[string]interface{}{
				{"text": "Hola"},
			},
		})
	}

	// Apply context timeout
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Build the request with system_instruction as separate field
	model := "gemma-4-31b-it"
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, p.apiKey,
	)

	body := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": systemPrompt},
			},
		},
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"temperature":     0.7,
			"maxOutputTokens": 2000,
		},
		"thinkingConfig": map[string]interface{}{
			"thinkingBudget": 0, // disable chain-of-thought for faster chat responses
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("error al preparar la solicitud: %w", err)
	}

	httpClient := &http.Client{Timeout: p.timeout}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error al crear la solicitud HTTP: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Error de conexión con el sensei: %v. ¿Revisaste tu conexión a internet?", err), nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error al leer la respuesta: %w", err)
	}

	if resp.StatusCode != 200 {
		bodyPreview := string(respBody)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		return fmt.Sprintf("El sensei no está disponible ahora (error %d: %s). Intenta de nuevo en unos segundos.", resp.StatusCode, bodyPreview), nil
	}

	// Parse the response
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []map[string]interface{} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("error al interpretar la respuesta: %w", err)
	}

	if len(result.Candidates) == 0 {
		return "El sensei no tiene respuesta para eso. ¿Quieres reformular la pregunta?", nil
	}

	// Filter only text parts, ignore thought parts (chain-of-thought reasoning)
	text := extractTextOnly(result.Candidates[0].Content.Parts)
	if text == "" {
		return "El sensei no tiene respuesta para eso. ¿Quieres reformular la pregunta?", nil
	}

	return text, nil
}

// extractTextOnly extracts only the actual response text from Gemini API response parts,
// skipping "thought" parts (chain-of-thought reasoning that gemma-4 models include).
// Thought parts have both "text" and "thought":true keys — we skip those.
func extractTextOnly(parts []map[string]interface{}) string {
	var texts []string
	for _, part := range parts {
		// Skip chain-of-thought reasoning parts
		if isThought, ok := part["thought"].(bool); ok && isThought {
			continue
		}
		if text, ok := part["text"].(string); ok && text != "" {
			texts = append(texts, text)
		}
	}
	return strings.Join(texts, "\n")
}
