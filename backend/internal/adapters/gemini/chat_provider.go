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

const (
	maxChatContextMessages     = 6
	maxChatMessagePromptRunes  = 360
	maxChatMessageSummaryRunes = 140
	chatMaxOutputTokens        = 1200
)

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
Responde breve por defecto. No repitas el roadmap completo salvo que te lo pidan.
Si el usuario pide una sección concreta, enfócate solo en esa sección.
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

	// Apply context timeout
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	model := "gemma-4-31b-it"
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		model,
	)

	body := buildChatRequestBody(systemPrompt, history)

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
	req.Header.Set("x-goog-api-key", p.apiKey)

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

func buildChatContents(history []chatstore.ChatMessage) []map[string]interface{} {
	_, contents := buildChatPromptContext(history)
	return contents
}

func buildChatRequestBody(systemPrompt string, history []chatstore.ChatMessage) map[string]interface{} {
	memory, contents := buildChatPromptContext(history)
	systemText := strings.TrimSpace(systemPrompt)
	if memory != "" {
		if systemText != "" {
			systemText += "\n\n"
		}
		systemText += memory
	}

	if systemText == "" {
		systemText = senseiSystemPrompt
	}

	return map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": systemText},
			},
		},
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"temperature":     0.7,
			"maxOutputTokens": chatMaxOutputTokens,
		},
	}
}

func buildChatPromptContext(history []chatstore.ChatMessage) (string, []map[string]interface{}) {
	if len(history) == 0 {
		return "", []map[string]interface{}{
			{
				"role":  "user",
				"parts": []map[string]interface{}{{"text": "Hola"}},
			},
		}
	}

	start := 0
	memoryLines := make([]string, 0, 4)
	if len(history) > maxChatContextMessages {
		dropped := history[:len(history)-maxChatContextMessages]
		memoryLines = append(memoryLines, "Resumen breve del contexto previo:")
		for _, msg := range dropped {
			memoryLines = append(memoryLines, "- "+summarizeChatMessage(msg, maxChatMessageSummaryRunes))
		}
		start = len(history) - maxChatContextMessages
	}

	contents := make([]map[string]interface{}, 0, len(history)-start)
	for _, msg := range history[start:] {
		content, note := trimChatMessageForPrompt(msg)
		if note != "" {
			memoryLines = append(memoryLines, "- "+note)
		}

		role := "user"
		if msg.Role == "sensei" {
			role = "model"
		}
		contents = append(contents, map[string]interface{}{
			"role": role,
			"parts": []map[string]interface{}{
				{"text": content},
			},
		})
	}

	if len(contents) == 0 {
		contents = append(contents, map[string]interface{}{
			"role":  "user",
			"parts": []map[string]interface{}{{"text": "Hola"}},
		})
	}

	return strings.Join(memoryLines, "\n"), contents
}

func trimChatMessageForPrompt(msg chatstore.ChatMessage) (string, string) {
	normalized := normalizeChatText(msg.Content)
	if normalized == "" {
		return "", ""
	}

	trimmed := abbreviateChatText(normalized, maxChatMessagePromptRunes)
	if trimmed == normalized {
		return normalized, ""
	}

	return trimmed, summarizeChatMessage(msg, maxChatMessageSummaryRunes)
}

func summarizeChatMessage(msg chatstore.ChatMessage, maxRunes int) string {
	label := "Usuario"
	if msg.Role == "sensei" {
		label = "Sensei"
	}
	return label + ": " + abbreviateChatText(normalizeChatText(msg.Content), maxRunes)
}

func normalizeChatText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func abbreviateChatText(text string, maxRunes int) string {
	if maxRunes < 1 {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}

	return strings.TrimSpace(string(runes[:maxRunes])) + "..."
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
