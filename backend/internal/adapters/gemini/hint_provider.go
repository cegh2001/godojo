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

	"godojo/internal/core/domain"
)

// GeminiClient abstracts the Gemini API call for testing and real use.
type GeminiClient interface {
	GenerateContent(ctx context.Context, prompt string) (string, error)
}

// HintProvider implements ports.HintProvider using the Gemini API.
type HintProvider struct {
	apiKey    string
	client    GeminiClient
	timeout   time.Duration
}

// NewHintProvider creates a HintProvider using the GEMINI_API_KEY environment variable.
// Uses the Gemini API via direct HTTP (no genai dependency).
func NewHintProvider() *HintProvider {
	apiKey := os.Getenv("GEMINI_API_KEY")
	p := &HintProvider{
		apiKey:  apiKey,
		timeout: 15 * time.Second,
	}
	if apiKey != "" {
		p.client = &realGeminiClient{apiKey: apiKey}
	}
	return p
}

// NewHintProviderWithClient creates a HintProvider with a pre-configured client (for testing).
func NewHintProviderWithClient(apiKey string, client GeminiClient) *HintProvider {
	return &HintProvider{
		apiKey:  apiKey,
		client:  client,
		timeout: 15 * time.Second,
	}
}

// GetHint requests a Socratic hint from the AI sensei.
// Returns two channels: one for the hint, one for errors.
// Exactly one of them will receive a value. The goroutine runs asynchronously.
func (p *HintProvider) GetHint(ctx context.Context, exercise *domain.Exercise, testOutput string) (<-chan *domain.Hint, <-chan error) {
	hintCh := make(chan *domain.Hint, 1)
	errCh := make(chan error, 1)

	go func() {
		// Apply our internal timeout
		ctx, cancel := context.WithTimeout(ctx, p.timeout)
		defer cancel()

		requestedAt := time.Now()

		// Check if we have a client configured
		if p.client == nil {
			if p.apiKey == "" {
				errCh <- fmt.Errorf("API key de Gemini no configurada. Configura la variable de entorno GEMINI_API_KEY.")
			} else {
				errCh <- fmt.Errorf("cliente de Gemini no inicializado.")
			}
			close(errCh)
			return
		}

		// Build the Socratic prompt in Spanish
		prompt := buildSocraticPrompt(exercise, testOutput)

		// Call the Gemini API
		response, err := p.client.GenerateContent(ctx, prompt)
		if err != nil {
			if ctx.Err() != nil {
				errCh <- fmt.Errorf("timeout: el sensei no respondió a tiempo. Prueba de nuevo: %w", ctx.Err())
			} else {
				errCh <- fmt.Errorf("error al consultar al sensei (API): %w", err)
			}
			close(errCh)
			return
		}

		receivedAt := time.Now()

		hint, err := domain.NewHint(exercise.Slug, response, requestedAt, receivedAt)
		if err != nil {
			errCh <- fmt.Errorf("error al crear la pista: %w", err)
			close(errCh)
			return
		}

		hintCh <- hint
		close(hintCh)
	}()

	return hintCh, errCh
}

// buildSocraticPrompt constructs a Socratic prompt in Spanish Rioplatense.
func buildSocraticPrompt(exercise *domain.Exercise, testOutput string) string {
	// Truncate very long code to avoid token limits
	code := exercise.TemplateCode
	if len(code) > 4000 {
		code = code[:4000] + "\n// ... (código truncado)\n"
	}

	// Truncate test output if too long
	output := testOutput
	if len(output) > 2000 {
		output = output[:2000] + "\n... (salida truncada)"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📝 Ejercicio: %s\n", exercise.Title))
	sb.WriteString(fmt.Sprintf("📂 Tema: %s\n\n", exercise.TopicSlug))

	sb.WriteString("--- Test output ---\n")
	sb.WriteString(output)
	sb.WriteString("\n--- Fin test output ---\n\n")

	sb.WriteString("--- Código actual ---\n")
	sb.WriteString(code)
	sb.WriteString("\n--- Fin código ---\n\n")

	sb.WriteString("Dame UNA sola pregunta o pista que lo haga pensar. No le des la solución.\n")
	sb.WriteString("Usa español neutro latinoamericano.")

	return sb.String()
}

// realGeminiClient implements GeminiClient using direct HTTP calls to the Gemini API.
type realGeminiClient struct {
	apiKey     string
	httpClient *http.Client
}

func (c *realGeminiClient) GenerateContent(ctx context.Context, prompt string) (string, error) {
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	// Gemini API endpoint — uses gemini-2.5-flash for fast, reliable hints
	model := "gemini-2.5-flash"
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, c.apiKey,
	)

	body := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": "Eres un sensei de Go. Das pistas socráticas. No das la solución completa. Usas español neutro latinoamericano."},
			},
		},
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.7,
			"maxOutputTokens": 800,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("error al preparar la solicitud: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error al crear la solicitud HTTP: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error al consultar Gemini: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error al leer la respuesta: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Gemini API devolvió error %d: %s", resp.StatusCode, string(respBody))
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
		return "", fmt.Errorf("error al interpretar la respuesta de Gemini: %w", err)
	}

	if len(result.Candidates) == 0 {
		return "", fmt.Errorf("Gemini no devolvió contenido")
	}

	// Filter only text parts, ignore thought parts (chain-of-thought reasoning) and functionCalls
	parts := extractContentParts(result.Candidates[0].Content.Parts)
	var texts []string
	for _, part := range parts {
		if part.Text != "" {
			texts = append(texts, part.Text)
		}
	}
	text := strings.Join(texts, "\n")
	if text == "" {
		return "", fmt.Errorf("Gemini no devolvió contenido")
	}

	return text, nil
}
