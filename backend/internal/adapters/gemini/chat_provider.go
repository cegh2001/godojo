package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"godojo/internal/adapters/chatstore"
)

// ChatProvider handles chat conversations with a Gemma-4-31b-it model via the Gemini API.
// It follows the same HTTP pattern used by realGeminiClient in hint_provider.go.
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

// System prompt for the sensei — Spanish Rioplatense.
const senseiSystemPrompt = `Sos un sensei experto en Go, parte de GoDojo. Ayudás a estudiantes a aprender Go desde cero.
Conocés el roadmap de GoDojo (7 fases, 23 ejercicios). Podés explicar conceptos, dar ejemplos,
sugerir ejercicios, y responder preguntas. Usá voseo rioplatense. Sé didáctico pero conciso.
Si te preguntan algo que no sabés, decilo con honestidad.`

// SendMessage sends a message to the Gemini API and returns the response.
// It includes the full conversation history + system prompt.
func (p *ChatProvider) SendMessage(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("Sensei no disponible — configurá GEMINI_API_KEY en .env")
	}

	if systemPrompt == "" {
		systemPrompt = senseiSystemPrompt
	}

	// Build the contents array for the Gemini API
	// First entry: system prompt as a "user" role message (Gemma doesn't support system role)
	contents := []map[string]interface{}{
		{
			"role": "user",
			"parts": []map[string]interface{}{
				{"text": systemPrompt},
			},
		},
		{
			"role": "model",
			"parts": []map[string]interface{}{
				{"text": "Entendido. Soy el sensei de GoDojo. ¿En qué te puedo ayudar hoy?"},
			},
		},
	}

	// Add conversation history
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

	// Apply context timeout
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Build the request
	model := "gemma-4-31b-it"
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, p.apiKey,
	)

	body := map[string]interface{}{
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"temperature":     0.7,
			"maxOutputTokens": 2000,
			"topP":            0.95,
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
		return fmt.Sprintf("El sensei no está disponible ahora (error %d). Intentá de nuevo en unos segundos.", resp.StatusCode), nil
	}

	// Parse the response
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("error al interpretar la respuesta: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "El sensei no tiene respuesta para eso. ¿Querés reformular la pregunta?", nil
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}
