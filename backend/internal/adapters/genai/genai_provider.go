package genai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"godojo/internal/adapters/chatstore"
	"godojo/internal/core/domain"

	g "google.golang.org/genai"
)

const (
	maxChatContextMessages     = 6
	maxChatMessagePromptRunes  = 360
	maxChatMessageSummaryRunes = 140
	defaultChatModel           = "gemma-4-31b-it"
	defaultSenseiTimeout       = 30 * time.Second
	defaultSenseiToolTimeout   = 60 * time.Second
	senseiModelEnv             = "GODOJO_SENSEI_MODEL"
	senseiTimeoutEnv           = "GODOJO_SENSEI_TIMEOUT_SECONDS"
	senseiToolTimeoutEnv       = "GODOJO_SENSEI_TOOL_TIMEOUT_SECONDS"
)

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

// GenaiProvider implements ports.SenseiProvider using the google.golang.org/genai SDK.
type GenaiProvider struct {
	client      *g.Client
	apiKey      string
	model       string
	timeout     time.Duration
	toolTimeout time.Duration
}

// NewGenaiProvider creates a GenaiProvider using environment variables for configuration.
// The genai Client is created lazily on first use.
func NewGenaiProvider() *GenaiProvider {
	timeout, toolTimeout := loadSenseiTimeoutsFromEnv()
	model := resolveModelFromEnv()
	return &GenaiProvider{
		apiKey:      os.Getenv("GEMINI_API_KEY"),
		model:       model,
		timeout:     timeout,
		toolTimeout: toolTimeout,
	}
}

// SendMessage sends a message to the Gemini API and returns the response as content parts.
func (p *GenaiProvider) SendMessage(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) ([]domain.ContentPart, error) {
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

	// Build contents and config
	memory, contents := buildGenaiContents(history)
	config := buildGenaiConfig(systemPrompt, memory, tools)

	// Ensure client is initialized
	if err := p.ensureClient(); err != nil {
		return nil, err
	}

	resp, err := p.client.Models.GenerateContent(ctx, p.model, contents, config)
	if err != nil {
		return nil, fmt.Errorf("error del sensei: %w", err)
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return []domain.ContentPart{{Text: "El sensei no tiene respuesta para eso. ¿Querés reformular la pregunta?"}}, nil
	}

	// Filter out thought parts
	var filtered []*g.Part
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Thought {
			continue
		}
		if part.Text == "" && part.FunctionCall == nil {
			continue
		}
		filtered = append(filtered, part)
	}

	if len(filtered) == 0 {
		return []domain.ContentPart{{Text: "El sensei no tiene respuesta para eso. ¿Querés reformular la pregunta?"}}, nil
	}

	return PartsToDomain(filtered), nil
}

// SendFunctionResponse sends a function execution result back to the AI and returns the next response.
func (p *GenaiProvider) SendFunctionResponse(ctx context.Context, history []chatstore.ChatMessage, callID string, name string, result interface{}) ([]domain.ContentPart, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Sensei no disponible — configura GEMINI_API_KEY en .env")
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Build contents: history + function response
	contents := buildFunctionResponseContents(history, callID, name, result)
	config := buildGenaiConfig(senseiSystemPrompt, "", nil) // no tools, no memory for function response

	// Ensure client is initialized
	if err := p.ensureClient(); err != nil {
		return nil, err
	}

	resp, err := p.client.Models.GenerateContent(ctx, p.model, contents, config)
	if err != nil {
		return nil, fmt.Errorf("error del sensei: %w", err)
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return []domain.ContentPart{{Text: "El sensei no tiene respuesta para eso. ¿Querés reformular la pregunta?"}}, nil
	}

	var filtered []*g.Part
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Thought {
			continue
		}
		if part.Text == "" && part.FunctionCall == nil {
			continue
		}
		filtered = append(filtered, part)
	}

	if len(filtered) == 0 {
		return []domain.ContentPart{{Text: "El sensei no tiene respuesta para eso. ¿Querés reformular la pregunta?"}}, nil
	}

	return PartsToDomain(filtered), nil
}

// ensureClient creates the genai client lazily if not already initialized.
func (p *GenaiProvider) ensureClient() error {
	if p.client != nil {
		return nil
	}
	if p.apiKey == "" {
		return fmt.Errorf("Sensei no disponible — configura GEMINI_API_KEY en .env")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := g.NewClient(ctx, &g.ClientConfig{
		APIKey:  p.apiKey,
		Backend: g.BackendGeminiAPI,
	})
	if err != nil {
		return fmt.Errorf("no se pudo conectar con Gemini: %w", err)
	}
	p.client = client
	return nil
}

// buildGenaiContents converts chat history to genai Content slices.
// Preserves context window truncation (max 6 messages, rune limits).
func buildGenaiContents(history []chatstore.ChatMessage) (string, []*g.Content) {
	if len(history) == 0 {
		return "", []*g.Content{
			{
				Role:  "user",
				Parts: []*g.Part{{Text: "Hola"}},
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

	contents := make([]*g.Content, 0, len(history)-start)
	for _, msg := range history[start:] {
		content, note := trimChatMessageForPrompt(msg)
		if note != "" {
			memoryLines = append(memoryLines, "- "+note)
		}

		role := "user"
		if msg.Role == "sensei" {
			role = "model"
		}
		contents = append(contents, &g.Content{
			Role:  role,
			Parts: []*g.Part{{Text: content}},
		})
	}

	if len(contents) == 0 {
		contents = append(contents, &g.Content{
			Role:  "user",
			Parts: []*g.Part{{Text: "Hola"}},
		})
	}

	return strings.Join(memoryLines, "\n"), contents
}

// buildGenaiConfig creates a GenerateContentConfig with system instruction and tools.
func buildGenaiConfig(systemPrompt string, memory string, tools []domain.ToolDeclaration) *g.GenerateContentConfig {
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

	config := &g.GenerateContentConfig{
		SystemInstruction: &g.Content{
			Parts: []*g.Part{{Text: systemText}},
		},
	}

	if len(tools) > 0 {
		funcDecls := ToolDeclarationsToGenai(tools)
		config.Tools = []*g.Tool{
			{FunctionDeclarations: funcDecls},
		}
	}

	return config
}

// buildFunctionResponseContents builds contents with the function response appended at the end.
func buildFunctionResponseContents(history []chatstore.ChatMessage, callID string, name string, result interface{}) []*g.Content {
	_, contents := buildGenaiContents(history)

	// Build the function response
	responseMap := make(map[string]any)
	switch v := result.(type) {
	case map[string]interface{}:
		responseMap = v
	case string:
		responseMap["result"] = v
	default:
		responseMap["result"] = fmt.Sprint(v)
	}

	contents = append(contents, &g.Content{
		Role: "user",
		Parts: []*g.Part{
			{
				FunctionResponse: &g.FunctionResponse{
					ID:       callID,
					Name:     name,
					Response: responseMap,
				},
			},
		},
	})

	return contents
}

// --- Context window helpers (preserved from chat_provider.go) ---

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

// --- Error handling ---

// mapGenaiError maps common errors to Spanish messages.
func mapGenaiError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "El sensei tardó demasiado en responder. Probá de nuevo en unos segundos."
	}
	return fmt.Sprintf("Error de conexión con el sensei: %v. ¿Revisaste tu conexión a internet?", err)
}

// --- Env helpers ---

func resolveModelFromEnv() string {
	if model := strings.TrimSpace(os.Getenv(senseiModelEnv)); model != "" {
		return model
	}
	return defaultChatModel
}

func loadSenseiTimeoutsFromEnv() (time.Duration, time.Duration) {
	timeout := envDurationSecondsOrDefault(senseiTimeoutEnv, defaultSenseiTimeout)
	toolTimeout := envDurationSecondsOrDefault(senseiToolTimeoutEnv, defaultSenseiToolTimeout)
	if toolTimeout < timeout {
		toolTimeout = timeout
	}
	return timeout, toolTimeout
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
