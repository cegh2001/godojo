package gemini

import (
	"context"
	"fmt"
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
// Returns a provider that will return an error if the API key is not configured.
func NewHintProvider() *HintProvider {
	apiKey := os.Getenv("GEMINI_API_KEY")
	return &HintProvider{
		apiKey:  apiKey,
		timeout: 15 * time.Second,
	}
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
		defer close(hintCh)
		defer close(errCh)

		// Apply our internal timeout
		ctx, cancel := context.WithTimeout(ctx, p.timeout)
		defer cancel()

		requestedAt := time.Now()

		// Check if we have a client configured
		if p.client == nil {
			if p.apiKey == "" {
				errCh <- fmt.Errorf("API key de Gemini no configurada. Configurá la variable de entorno GEMINI_API_KEY.")
			} else {
				errCh <- fmt.Errorf("cliente de Gemini no inicializado.")
			}
			return
		}

		// Build the Socratic prompt in Spanish
		prompt := buildSocraticPrompt(exercise, testOutput)

		// Call the Gemini API
		response, err := p.client.GenerateContent(ctx, prompt)
		if err != nil {
			if ctx.Err() != nil {
				errCh <- fmt.Errorf("timeout: el sensei no respondió a tiempo. Probá de nuevo: %w", ctx.Err())
			} else {
				errCh <- fmt.Errorf("error al consultar al sensei (API): %w", err)
			}
			return
		}

		receivedAt := time.Now()

		hint, err := domain.NewHint(exercise.Slug, response, requestedAt, receivedAt)
		if err != nil {
			errCh <- fmt.Errorf("error al crear la pista: %w", err)
			return
		}

		hintCh <- hint
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
	sb.WriteString("Sos un sensei de Go. No des la respuesta. Guiá con preguntas socráticas.\n")
	sb.WriteString("El estudiante está trabado en este ejercicio.\n\n")
	sb.WriteString(fmt.Sprintf("📝 Ejercicio: %s\n", exercise.Title))
	sb.WriteString(fmt.Sprintf("📂 Tema: %s\n\n", exercise.TopicSlug))

	sb.WriteString("--- Test output ---\n")
	sb.WriteString(output)
	sb.WriteString("\n--- Fin test output ---\n\n")

	sb.WriteString("--- Código actual ---\n")
	sb.WriteString(code)
	sb.WriteString("\n--- Fin código ---\n\n")

	sb.WriteString("Dame UNA sola pregunta o pista que lo haga pensar. No le des la solución.\n")
	sb.WriteString("Usá voseo (español rioplatense).")

	return sb.String()
}
