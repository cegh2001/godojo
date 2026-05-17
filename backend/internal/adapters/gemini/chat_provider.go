// Deprecated: Replaced by adapters/genai.GenaiProvider. Will be removed in a future version.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"godojo/internal/adapters/chatstore"
	"godojo/internal/core/domain"
)

// ChatProvider handles chat conversations with Gemma via the Gemini API.
// Uses system_instruction for persona and contents for conversation history.
// See: https://ai.google.dev/gemini-api/docs/system-instructions
type ChatProvider struct {
	apiKey        string
	timeout       time.Duration
	toolTimeout   time.Duration
	httpClient    *http.Client
	requestConfig chatRequestConfig
	mu            sync.Mutex
	callContexts  map[string]interactionContext
}

type chatRequestConfig struct {
	model              string
	textModel          string
	maxOutputTokens    int
	thinkingBudget     int
	enableGoogleSearch bool
}

type interactionContext struct {
	interactionID string
	systemPrompt  string
	tools         []domain.ToolDeclaration
}

type geminiAPIErrorEnvelope struct {
	Error struct {
		Code    interface{} `json:"code"`
		Message string      `json:"message"`
		Status  string      `json:"status"`
	} `json:"error"`
}

const (
	maxChatContextMessages     = 6
	maxChatMessagePromptRunes  = 360
	maxChatMessageSummaryRunes = 140
	defaultChatMaxOutputTokens = 768
	defaultChatThinkingBudget  = 1024
	defaultChatThinkingLevel   = "low"
	defaultFastModel           = "gemma-4-26b-a4b-it"
	defaultHeavyModel          = "gemma-4-31b-it"
	defaultChatModel           = defaultHeavyModel
	defaultSenseiTimeout       = 30 * time.Second
	defaultSenseiToolTimeout   = 60 * time.Second
	senseiModelEnv             = "GODOJO_SENSEI_MODEL"
	senseiFastModelEnv         = "GODOJO_SENSEI_FAST_MODEL"
	senseiHeavyModelEnv        = "GODOJO_SENSEI_HEAVY_MODEL"
	senseiTimeoutEnv           = "GODOJO_SENSEI_TIMEOUT_SECONDS"
	senseiToolTimeoutEnv       = "GODOJO_SENSEI_TOOL_TIMEOUT_SECONDS"
	geminiMaxAttempts          = 3
	geminiBaseRetryDelay       = 250 * time.Millisecond
	geminiMaxRetryDelay        = 2 * time.Second
)

// NewChatProvider creates a ChatProvider using the GEMINI_API_KEY environment variable.
func NewChatProvider() *ChatProvider {
	timeout, toolTimeout := loadSenseiTimeoutsFromEnv()
	return &ChatProvider{
		apiKey:        os.Getenv("GEMINI_API_KEY"),
		timeout:       timeout,
		toolTimeout:   toolTimeout,
		requestConfig: loadChatRequestConfigFromEnv(),
		callContexts:  make(map[string]interactionContext),
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
Si generas ejercicios o archivos, preferí scaffolds con pistas dentro del mismo archivo, comentarios TODO y ayudas graduales; no des soluciones completas salvo que te lo pidan.
Tienes acceso indirecto al workspace del estudiante mediante herramientas para listar y leer archivos Go.
Si el usuario pregunta qué archivos, ejercicios o lecciones tiene en el workspace, usa esas herramientas antes de responder.
No digas que no puedes revisar el workspace cuando la consulta pueda resolverse listando o leyendo archivos.
Si el usuario dice que ya terminó un ejercicio, te pide revisarlo, o hace referencia al último archivo sin repetir el nombre, usa la herramienta para leer el archivo reciente antes de pedirle que pegue contenido.
Cuando crees ejercicios sobre un tema del roadmap, primero identifica el tema y guarda el archivo dentro de su carpeta temática; evitá dejar archivos sueltos en la raíz salvo que el usuario lo pida explícitamente.
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
		effectiveTimeout = p.toolTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, effectiveTimeout)
	defer cancel()

	if len(tools) > 0 {
		return p.sendInteractionMessage(ctx, effectiveTimeout, systemPrompt, history, tools)
	}

	requestConfig := p.requestConfig.forTextRequest()
	model := requestConfig.model
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		model,
	)

	body := buildChatRequestBodyWithConfig(systemPrompt, history, tools, requestConfig)

	respBody, statusCode, err := p.doGenerateContentRequest(ctx, effectiveTimeout, url, body)
	if err != nil {
		return []domain.ContentPart{{Text: formatGeminiTransportError(err)}}, nil
	}

	if statusCode != http.StatusOK {
		return []domain.ContentPart{{Text: formatGeminiAPIError(statusCode, respBody)}}, nil
	}

	parts, err := parseGenerateContentParts(respBody)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return []domain.ContentPart{{Text: "El sensei no tiene respuesta para eso. ¿Quieres reformular la pregunta?"}}, nil
	}

	return parts, nil
}

func buildChatContents(history []chatstore.ChatMessage) []map[string]interface{} {
	_, contents := buildChatPromptContext(history)
	return contents
}

func buildInteractionRequestBody(systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration, cfg chatRequestConfig) map[string]interface{} {
	input := buildInteractionInput(systemPrompt, history)
	body := map[string]interface{}{
		"model": pOrDefaultModel(cfg.model),
		"input": input,
		"tools": buildInteractionToolsJSON(tools, cfg.enableGoogleSearch),
	}
	return body
}

func buildInteractionInput(systemPrompt string, history []chatstore.ChatMessage) string {
	memory, contents := buildChatPromptContext(history)
	var inputParts []string
	if prompt := strings.TrimSpace(systemPrompt); prompt != "" {
		inputParts = append(inputParts, prompt)
	}
	if memory != "" {
		inputParts = append(inputParts, memory)
	}
	for _, content := range contents {
		role, _ := content["role"].(string)
		parts, _ := content["parts"].([]map[string]interface{})
		for _, part := range parts {
			text, _ := part["text"].(string)
			if text == "" {
				continue
			}
			if role == "model" {
				inputParts = append(inputParts, "Sensei: "+text)
			} else {
				inputParts = append(inputParts, "Usuario: "+text)
			}
		}
	}
	if len(inputParts) == 0 {
		return "Hola"
	}
	return strings.Join(inputParts, "\n\n")
}

func pOrDefaultModel(model string) string {
	if strings.TrimSpace(model) == "" {
		return defaultChatModel
	}
	return model
}

func buildChatRequestBody(systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) map[string]interface{} {
	return buildChatRequestBodyWithConfig(systemPrompt, history, tools, defaultChatRequestConfig())
}

func buildChatRequestBodyWithConfig(systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration, cfg chatRequestConfig) map[string]interface{} {
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

	generationConfig := map[string]interface{}{
		"temperature":     0.7,
		"maxOutputTokens": cfg.maxOutputTokens,
	}
	if thinkingConfig := buildThinkingConfig(cfg.model, cfg.thinkingBudget); len(thinkingConfig) > 0 {
		generationConfig["thinkingConfig"] = thinkingConfig
	}

	body := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": systemText},
			},
		},
		"contents":         contents,
		"generationConfig": generationConfig,
	}

	// Include tools only when provided (backward compatible: nil/empty means no tools)
	toolEntries := buildToolsJSONWithGoogleSearch(tools, cfg.enableGoogleSearch)
	if len(toolEntries) > 0 {
		body["tools"] = toolEntries
		if cfg.enableGoogleSearch {
			// Required when combining built-in tools (like googleSearch)
			// with custom functionDeclarations.
			body["toolConfig"] = map[string]interface{}{
				"includeServerSideToolInvocations": true,
			}
		}
	}

	return body
}

func defaultChatRequestConfig() chatRequestConfig {
	return chatRequestConfig{
		model:              defaultChatModel,
		textModel:          defaultFastModel,
		maxOutputTokens:    defaultChatMaxOutputTokens,
		thinkingBudget:     defaultChatThinkingBudget,
		enableGoogleSearch: false,
	}
}

func loadChatRequestConfigFromEnv() chatRequestConfig {
	cfg := defaultChatRequestConfig()
	cfg.model = resolveChatModelFromEnv()
	cfg.textModel = resolveFastModelFromEnv()
	cfg.maxOutputTokens = envIntOrDefault("GODOJO_SENSEI_MAX_OUTPUT_TOKENS", cfg.maxOutputTokens)
	cfg.thinkingBudget = envIntOrDefault("GODOJO_SENSEI_THINKING_BUDGET", cfg.thinkingBudget)
	cfg.enableGoogleSearch = envBoolOrDefault("GODOJO_SENSEI_ENABLE_GOOGLE_SEARCH", cfg.enableGoogleSearch)
	return cfg
}

func loadSenseiTimeoutsFromEnv() (time.Duration, time.Duration) {
	timeout := envDurationSecondsOrDefault(senseiTimeoutEnv, defaultSenseiTimeout)
	toolTimeout := envDurationSecondsOrDefault(senseiToolTimeoutEnv, defaultSenseiToolTimeout)
	if toolTimeout < timeout {
		toolTimeout = timeout
	}
	return timeout, toolTimeout
}

func (cfg chatRequestConfig) forTextRequest() chatRequestConfig {
	requestCfg := cfg
	if strings.TrimSpace(requestCfg.textModel) != "" {
		requestCfg.model = requestCfg.textModel
	}
	return requestCfg
}

func resolveFastModelFromEnv() string {
	return envOrDefault(senseiFastModelEnv, defaultFastModel)
}

func resolveChatModelFromEnv() string {
	return envOrDefault(senseiModelEnv, resolveHeavyModelFromEnv())
}

func resolveHeavyModelFromEnv() string {
	return envOrDefault(senseiHeavyModelEnv, defaultHeavyModel)
}

func buildThinkingConfig(model string, thinkingBudget int) map[string]interface{} {
	normalizedModel := strings.ToLower(strings.TrimSpace(model))
	if normalizedModel == "" {
		normalizedModel = defaultChatModel
	}

	if strings.HasPrefix(normalizedModel, "gemma-4") {
		return nil
	}

	if strings.HasPrefix(normalizedModel, "gemini-3") {
		thinkingLevel := defaultChatThinkingLevel
		if thinkingBudget == 0 {
			thinkingLevel = "minimal"
		}
		return map[string]interface{}{"thinkingLevel": thinkingLevel}
	}

	return map[string]interface{}{"thinkingBudget": thinkingBudget}
}

func envOrDefault(name string, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return defaultValue
}

func envIntOrDefault(name string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func envDurationSecondsOrDefault(name string, defaultValue time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return defaultValue
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return defaultValue
	}
	return time.Duration(seconds) * time.Second
}

func envBoolOrDefault(name string, defaultValue bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if value == "" {
		return defaultValue
	}
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
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

	ellipsis := "..."
	ellipsisRunes := []rune(ellipsis)
	if maxRunes <= len(ellipsisRunes) {
		return string(ellipsisRunes[:maxRunes])
	}

	cutoff := maxRunes - len(ellipsisRunes)
	return strings.TrimSpace(string(runes[:cutoff])) + ellipsis
}

func formatGeminiAPIError(statusCode int, respBody []byte) string {
	statusLabel := fmt.Sprintf("HTTP_%d", statusCode)
	detail := abbreviateChatText(normalizeChatText(string(respBody)), 180)

	var envelope geminiAPIErrorEnvelope
	if err := json.Unmarshal(respBody, &envelope); err == nil {
		if codeLabel := geminiErrorCodeLabel(envelope.Error.Code); codeLabel != "" && envelope.Error.Status == "" {
			statusLabel = codeLabel
		}
		if envelope.Error.Status != "" {
			statusLabel = envelope.Error.Status
		}
		if envelope.Error.Message != "" {
			detail = abbreviateChatText(normalizeChatText(envelope.Error.Message), 180)
		}
		if numericCode := geminiErrorNumericCode(envelope.Error.Code); numericCode != 0 {
			statusCode = numericCode
		}
	}

	message := fmt.Sprintf("El sensei no está disponible ahora: Gemini devolvió %s (%d).", statusLabel, statusCode)
	if detail != "" {
		message += " " + detail
	}

	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		message += " Revisá la configuración de GEMINI_API_KEY."
	case http.StatusTooManyRequests:
		message += " Parece un límite de cuota o rate limit de tu cuenta/proyecto en Gemini."
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		message += " Parece un fallo interno del proveedor, no de tu ejercicio."
	}

	message += " Probá de nuevo en unos segundos."
	return message
}

func geminiErrorNumericCode(code interface{}) int {
	switch value := code.(type) {
	case float64:
		return int(value)
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	default:
		return 0
	}
}

func geminiErrorCodeLabel(code interface{}) string {
	switch value := code.(type) {
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return ""
		}
		return strings.ToUpper(trimmed)
	default:
		return ""
	}
}

func (p *ChatProvider) doGenerateContentRequest(ctx context.Context, timeout time.Duration, url string, body map[string]interface{}) ([]byte, int, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("error al preparar la solicitud: %w", err)
	}

	httpClient := p.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}

	for attempt := 0; attempt < geminiMaxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
		if err != nil {
			return nil, 0, fmt.Errorf("error al crear la solicitud HTTP: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-goog-api-key", p.apiKey)

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, 0, err
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, 0, fmt.Errorf("error al leer la respuesta: %w", readErr)
		}

		if resp.StatusCode == http.StatusOK {
			return respBody, resp.StatusCode, nil
		}

		if !shouldRetryGeminiStatus(resp.StatusCode) || attempt == geminiMaxAttempts-1 {
			return respBody, resp.StatusCode, nil
		}

		if err := waitForGeminiRetry(ctx, attempt, resp.Header.Get("Retry-After")); err != nil {
			return nil, 0, err
		}
	}

	return nil, 0, fmt.Errorf("Gemini no devolvió respuesta")
}

func shouldRetryGeminiStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError
}

func waitForGeminiRetry(ctx context.Context, attempt int, retryAfter string) error {
	delay := geminiRetryDelay(attempt, retryAfter)
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func geminiRetryDelay(attempt int, retryAfter string) time.Duration {
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && seconds > 0 {
			return minDuration(time.Duration(seconds)*time.Second, geminiMaxRetryDelay)
		}
		if retryAt, err := http.ParseTime(retryAfter); err == nil {
			delay := time.Until(retryAt)
			if delay > 0 {
				return minDuration(delay, geminiMaxRetryDelay)
			}
		}
	}

	delay := time.Duration(attempt+1) * geminiBaseRetryDelay
	return minDuration(delay, geminiMaxRetryDelay)
}

func minDuration(a time.Duration, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func formatGeminiTransportError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "El sensei tardó demasiado en responder. Probá de nuevo en unos segundos."
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "El sensei tardó demasiado en responder. Probá de nuevo en unos segundos."
	}

	return fmt.Sprintf("Error de conexión con el sensei: %v. ¿Revisaste tu conexión a internet?", err)
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
				id, _ := fcMap["id"].(string)
				name, _ := fcMap["name"].(string)
				args, _ := fcMap["args"].(map[string]interface{})
				cp.FunctionCall = &domain.FunctionCall{
					ID:   id,
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
	return buildToolsJSONWithGoogleSearch(tools, false)
}

func buildInteractionToolsJSON(tools []domain.ToolDeclaration, enableGoogleSearch bool) []map[string]interface{} {
	if len(tools) == 0 && !enableGoogleSearch {
		return nil
	}

	result := make([]map[string]interface{}, 0, len(tools)+1)
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

		result = append(result, map[string]interface{}{
			"type":        "function",
			"name":        tool.Name,
			"description": tool.Description,
			"parameters":  params,
		})
	}
	if enableGoogleSearch {
		result = append(result, map[string]interface{}{"type": "google_search"})
	}
	return result
}

func buildToolsJSONWithGoogleSearch(tools []domain.ToolDeclaration, enableGoogleSearch bool) []map[string]interface{} {
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

	result := []map[string]interface{}{
		{"functionDeclarations": functionDeclarations},
	}
	if enableGoogleSearch {
		result = append(result, map[string]interface{}{"googleSearch": map[string]interface{}{}})
	}
	return result
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

func buildInteractionFunctionResultInput(callID string, name string, result interface{}) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"type":    "function_result",
			"name":    name,
			"call_id": callID,
			"result":  normalizeInteractionFunctionResult(result),
		},
	}
}

func normalizeInteractionFunctionResult(result interface{}) interface{} {
	switch value := result.(type) {
	case nil:
		return "null"
	case string:
		return value
	case []interface{}:
		return value
	case []map[string]interface{}:
		return value
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprint(value)
		}
		return string(encoded)
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

	if ctxData, ok := p.takeInteractionContext(callID); ok {
		return p.sendInteractionFunctionResult(ctx, callID, name, result, ctxData)
	}

	// Build full contents from history + append the functionResponse message
	memory, existingContents := buildChatPromptContext(history)
	_ = memory // memory is handled in system_instruction

	// Append functionResponse as a user message
	functionResponseMsg := buildFunctionResponseMessage(callID, name, result)
	contents := append(existingContents, functionResponseMsg)

	model := p.requestConfig.model
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		model,
	)

	body := buildChatRequestBodyWithConfig(senseiSystemPrompt, history, nil, p.requestConfig)
	body["contents"] = contents

	respBody, statusCode, err := p.doGenerateContentRequest(ctx, p.timeout, url, body)
	if err != nil {
		return []domain.ContentPart{{Text: formatGeminiTransportError(err)}}, nil
	}

	if statusCode != http.StatusOK {
		return []domain.ContentPart{{Text: formatGeminiAPIError(statusCode, respBody)}}, nil
	}

	parts, err := parseGenerateContentParts(respBody)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return []domain.ContentPart{{Text: "El sensei no tiene respuesta para eso. ¿Quieres reformular la pregunta?"}}, nil
	}

	return parts, nil
}

func (p *ChatProvider) sendInteractionMessage(ctx context.Context, timeout time.Duration, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) ([]domain.ContentPart, error) {
	body := buildInteractionRequestBody(systemPrompt, history, tools, p.requestConfig)
	url := "https://generativelanguage.googleapis.com/v1beta/interactions"
	respBody, statusCode, err := p.doInteractionsRequest(ctx, timeout, url, body)
	if err != nil {
		return []domain.ContentPart{{Text: formatGeminiTransportError(err)}}, nil
	}
	if statusCode != http.StatusOK {
		return []domain.ContentPart{{Text: formatGeminiAPIError(statusCode, respBody)}}, nil
	}
	parts, interactionID, err := parseInteractionResponse(respBody)
	if err != nil {
		return nil, err
	}
	for _, part := range parts {
		if part.FunctionCall != nil && part.FunctionCall.ID != "" {
			p.storeInteractionContext(part.FunctionCall.ID, interactionContext{
				interactionID: interactionID,
				systemPrompt:  systemPrompt,
				tools:         cloneToolDeclarations(tools),
			})
		}
	}
	return parts, nil
}

func (p *ChatProvider) sendInteractionFunctionResult(ctx context.Context, callID string, name string, result interface{}, ctxData interactionContext) ([]domain.ContentPart, error) {
	body := map[string]interface{}{
		"model":                   pOrDefaultModel(p.requestConfig.model),
		"previous_interaction_id": ctxData.interactionID,
		"tools":                   buildInteractionToolsJSON(ctxData.tools, p.requestConfig.enableGoogleSearch),
		"input":                   buildInteractionFunctionResultInput(callID, name, result),
	}
	url := "https://generativelanguage.googleapis.com/v1beta/interactions"
	respBody, statusCode, err := p.doInteractionsRequest(ctx, p.toolTimeout, url, body)
	if err != nil {
		return []domain.ContentPart{{Text: formatGeminiTransportError(err)}}, nil
	}
	if statusCode != http.StatusOK {
		return []domain.ContentPart{{Text: formatGeminiAPIError(statusCode, respBody)}}, nil
	}
	parts, interactionID, err := parseInteractionResponse(respBody)
	if err != nil {
		return nil, err
	}
	for _, part := range parts {
		if part.FunctionCall != nil && part.FunctionCall.ID != "" {
			p.storeInteractionContext(part.FunctionCall.ID, interactionContext{
				interactionID: interactionID,
				systemPrompt:  ctxData.systemPrompt,
				tools:         cloneToolDeclarations(ctxData.tools),
			})
		}
	}
	return parts, nil
}

func (p *ChatProvider) doInteractionsRequest(ctx context.Context, timeout time.Duration, url string, body map[string]interface{}) ([]byte, int, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("error al preparar la solicitud: %w", err)
	}

	httpClient := p.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, 0, fmt.Errorf("error al crear la solicitud HTTP: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("error al leer la respuesta: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

func parseGenerateContentParts(respBody []byte) ([]domain.ContentPart, error) {
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
		return nil, nil
	}
	return extractContentParts(result.Candidates[0].Content.Parts), nil
}

func parseInteractionResponse(respBody []byte) ([]domain.ContentPart, string, error) {
	var result struct {
		ID      string `json:"id"`
		Outputs []struct {
			Type      string                 `json:"type"`
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
			ID        string                 `json:"id"`
			Text      string                 `json:"text"`
		} `json:"outputs"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, "", fmt.Errorf("error al interpretar la respuesta: %w", err)
	}

	parts := make([]domain.ContentPart, 0, len(result.Outputs))
	for _, output := range result.Outputs {
		switch output.Type {
		case "function_call":
			parts = append(parts, domain.ContentPart{FunctionCall: &domain.FunctionCall{ID: output.ID, Name: output.Name, Args: output.Arguments}})
		case "text":
			if output.Text != "" {
				parts = append(parts, domain.ContentPart{Text: output.Text})
			}
		default:
			if output.Text != "" {
				parts = append(parts, domain.ContentPart{Text: output.Text})
			}
		}
	}

	return parts, result.ID, nil
}

func (p *ChatProvider) storeInteractionContext(callID string, ctx interactionContext) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.callContexts[callID] = ctx
}

func (p *ChatProvider) takeInteractionContext(callID string) (interactionContext, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	ctx, ok := p.callContexts[callID]
	if ok {
		delete(p.callContexts, callID)
	}
	return ctx, ok
}

func cloneToolDeclarations(tools []domain.ToolDeclaration) []domain.ToolDeclaration {
	if len(tools) == 0 {
		return nil
	}
	cloned := make([]domain.ToolDeclaration, len(tools))
	for i, tool := range tools {
		properties := make(map[string]domain.ToolProperty, len(tool.Parameters.Properties))
		for key, value := range tool.Parameters.Properties {
			properties[key] = value
		}
		required := append([]string(nil), tool.Parameters.Required...)
		cloned[i] = domain.ToolDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters: domain.ToolParameters{
				Type:       tool.Parameters.Type,
				Properties: properties,
				Required:   required,
			},
		}
	}
	return cloned
}
