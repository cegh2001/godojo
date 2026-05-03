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
	"godojo/internal/core/domain"
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

// SendMessage sends a message to the Gemini API and returns the response as content parts.
// When tools is nil/empty, behaves as before (text-only responses).
// When tools is non-empty, includes functionDeclarations + codeExecution + googleSearch.
// The response may contain text parts, functionCall parts, or both.
func (p *ChatProvider) SendMessage(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) ([]domain.ContentPart, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Sensei no disponible — configura GEMINI_API_KEY en .env")
	}

	if systemPrompt == "" {
		systemPrompt = senseiSystemPrompt
	}

	// Apply context timeout — longer for tool-enabled requests
	effectiveTimeout := p.timeout
	if len(tools) > 0 {
		effectiveTimeout = 90 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, effectiveTimeout)
	defer cancel()

	model := "gemma-4-31b-it"
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		model,
	)

	body := buildChatRequestBody(systemPrompt, history, tools)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error al preparar la solicitud: %w", err)
	}

	httpClient := &http.Client{Timeout: effectiveTimeout}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error al crear la solicitud HTTP: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return []domain.ContentPart{{Text: fmt.Sprintf("Error de conexión con el sensei: %v. ¿Revisaste tu conexión a internet?", err)}}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error al leer la respuesta: %w", err)
	}

	if resp.StatusCode != 200 {
		bodyPreview := string(respBody)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		return []domain.ContentPart{{Text: fmt.Sprintf("El sensei no está disponible ahora (error %d: %s). Intenta de nuevo en unos segundos.", resp.StatusCode, bodyPreview)}}, nil
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
		return nil, fmt.Errorf("error al interpretar la respuesta: %w", err)
	}

	if len(result.Candidates) == 0 {
		return []domain.ContentPart{{Text: "El sensei no tiene respuesta para eso. ¿Quieres reformular la pregunta?"}}, nil
	}

	// Extract content parts (text + functionCall), skip thoughts
	parts := extractContentParts(result.Candidates[0].Content.Parts)
	if len(parts) == 0 {
		return []domain.ContentPart{{Text: "El sensei no tiene respuesta para eso. ¿Quieres reformular la pregunta?"}}, nil
	}

	return parts, nil
}

func buildChatContents(history []chatstore.ChatMessage) []map[string]interface{} {
	_, contents := buildChatPromptContext(history)
	return contents
}

func buildChatRequestBody(systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) map[string]interface{} {
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

	body := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": systemText},
			},
		},
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"temperature":     0.7,
			"maxOutputTokens": chatMaxOutputTokens,
			"thinkingConfig": map[string]interface{}{
				"thinkingLevel": "HIGH",
			},
		},
	}

	// Include tools only when provided (backward compatible: nil/empty means no tools)
	toolEntries := buildToolsJSON(tools)
	if len(toolEntries) > 0 {
		body["tools"] = toolEntries
		// Required when combining built-in tools (codeExecution, googleSearch)
		// with custom functionDeclarations.
		body["toolConfig"] = map[string]interface{}{
			"includeServerSideToolInvocations": true,
		}
	}

	return body
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

	ellipsis := "..."
	ellipsisRunes := []rune(ellipsis)
	if maxRunes <= len(ellipsisRunes) {
		return string(ellipsisRunes[:maxRunes])
	}

	cutoff := maxRunes - len(ellipsisRunes)
	return strings.TrimSpace(string(runes[:cutoff])) + ellipsis
}

// extractContentParts extracts content parts from the Gemini API response.
// Parses both text and functionCall keys, skipping thought parts.
func extractContentParts(parts []map[string]interface{}) []domain.ContentPart {
	var result []domain.ContentPart
	for _, part := range parts {
		// Skip chain-of-thought reasoning parts
		if isThought, ok := part["thought"].(bool); ok && isThought {
			continue
		}

		var cp domain.ContentPart

		// Check for functionCall
		if fcRaw, ok := part["functionCall"]; ok {
			fcMap, ok := fcRaw.(map[string]interface{})
			if ok {
				name, _ := fcMap["name"].(string)
				args, _ := fcMap["args"].(map[string]interface{})
				cp.FunctionCall = &domain.FunctionCall{
					Name: name,
					Args: args,
				}
			}
		}

		// Check for text
		if text, ok := part["text"].(string); ok && text != "" {
			cp.Text = text
		}

		// Skip empty parts (e.g., thought-only with no other content)
		if cp.Text == "" && cp.FunctionCall == nil {
			continue
		}

		result = append(result, cp)
	}
	return result
}

// buildToolsJSON converts ToolDeclarations into the Gemini API tools format.
// Includes googleSearch for live grounding. codeExecution is reserved for Phase 2.
// Returns nil when no custom tools are provided (caller should exclude the key entirely).
func buildToolsJSON(tools []domain.ToolDeclaration) []map[string]interface{} {
	if len(tools) == 0 {
		return nil
	}

	var functionDeclarations []map[string]interface{}
	for _, tool := range tools {
		params := map[string]interface{}{
			"type": tool.Parameters.Type,
		}
		if len(tool.Parameters.Properties) > 0 {
			props := make(map[string]interface{}, len(tool.Parameters.Properties))
			for key, prop := range tool.Parameters.Properties {
				props[key] = map[string]interface{}{
					"type":        prop.Type,
					"description": prop.Description,
				}
			}
			params["properties"] = props
		}
		if len(tool.Parameters.Required) > 0 {
			params["required"] = tool.Parameters.Required
		}

		functionDeclarations = append(functionDeclarations, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"parameters":  params,
		})
	}

	return []map[string]interface{}{
		{"functionDeclarations": functionDeclarations},
		{"googleSearch": map[string]interface{}{}},
	}
}

// buildFunctionResponseMessage creates a user role message containing a functionResponse part.
// This is appended to the conversation history before sending back to the model.
func buildFunctionResponseMessage(callID string, name string, result interface{}) map[string]interface{} {
	return map[string]interface{}{
		"role": "user",
		"parts": []map[string]interface{}{
			{
				"functionResponse": map[string]interface{}{
					"name":     name,
					"response": result,
				},
			},
		},
	}
}

// SendFunctionResponse sends a function execution result back to the AI and returns the next response.
// The caller is responsible for having already appended the model's functionCall message to the history.
func (p *ChatProvider) SendFunctionResponse(ctx context.Context, history []chatstore.ChatMessage, callID string, name string, result interface{}) ([]domain.ContentPart, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Sensei no disponible — configura GEMINI_API_KEY en .env")
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Build full contents from history + append the functionResponse message
	memory, existingContents := buildChatPromptContext(history)
	_ = memory // memory is handled in system_instruction

	// Append functionResponse as a user message
	functionResponseMsg := buildFunctionResponseMessage(callID, name, result)
	contents := append(existingContents, functionResponseMsg)

	model := "gemma-4-31b-it"
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		model,
	)

	body := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": senseiSystemPrompt},
			},
		},
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"temperature":     0.7,
			"maxOutputTokens": chatMaxOutputTokens,
			"thinkingConfig": map[string]interface{}{
				"thinkingLevel": "HIGH",
			},
		},
		// Include tools so model can call more tools if needed
		"tools": buildToolsJSON([]domain.ToolDeclaration{}), // built-ins only
		"toolConfig": map[string]interface{}{
			"includeServerSideToolInvocations": true,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error al preparar la solicitud: %w", err)
	}

	httpClient := &http.Client{Timeout: p.timeout}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error al crear la solicitud HTTP: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return []domain.ContentPart{{Text: fmt.Sprintf("Error de conexión con el sensei: %v. ¿Revisaste tu conexión a internet?", err)}}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error al leer la respuesta: %w", err)
	}

	if resp.StatusCode != 200 {
		bodyPreview := string(respBody)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		return []domain.ContentPart{{Text: fmt.Sprintf("El sensei no está disponible ahora (error %d: %s). Intenta de nuevo en unos segundos.", resp.StatusCode, bodyPreview)}}, nil
	}

	var result2 struct {
		Candidates []struct {
			Content struct {
				Parts []map[string]interface{} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &result2); err != nil {
		return nil, fmt.Errorf("error al interpretar la respuesta: %w", err)
	}

	if len(result2.Candidates) == 0 {
		return []domain.ContentPart{{Text: "El sensei no tiene respuesta para eso. ¿Quieres reformular la pregunta?"}}, nil
	}

	parts := extractContentParts(result2.Candidates[0].Content.Parts)
	if len(parts) == 0 {
		return []domain.ContentPart{{Text: "El sensei no tiene respuesta para eso. ¿Quieres reformular la pregunta?"}}, nil
	}

	return parts, nil
}
