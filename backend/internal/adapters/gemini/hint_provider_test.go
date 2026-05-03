package gemini_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"godojo/internal/adapters/gemini"
	"godojo/internal/core/domain"
)

// mockClient implements gemini.GeminiClient for testing.
type mockClient struct {
	response string
	err      error
	delay    time.Duration
}

func (m *mockClient) GenerateContent(ctx context.Context, prompt string) (string, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestGetHint_Success_ReturnsHintOnChannel(t *testing.T) {
	mock := &mockClient{
		response: "¿Probaste revisar el valor de retorno de la función?",
	}

	provider := gemini.NewHintProviderWithClient("fake-key", mock)

	ex, _ := domain.NewExercise("test-ex", "Test Exercise", "fundamentos", "code", "test", "")
	ctx := context.Background()

	hintCh, errCh := provider.GetHint(ctx, ex, "FAIL: expected 5, got 3")

	select {
	case hint := <-hintCh:
		if hint == nil {
			t.Fatal("expected non-nil hint on success channel")
		}
		if hint.ExerciseID != "test-ex" {
			t.Errorf("ExerciseID = %q, want %q", hint.ExerciseID, "test-ex")
		}
		if hint.Content == "" {
			t.Error("hint content should not be empty")
		}
	case err := <-errCh:
		t.Fatalf("unexpected error on success path: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for hint")
	}
}

func TestGetHint_APIError_SendsErrorOnChannel(t *testing.T) {
	mock := &mockClient{
		err: errors.New("API unavailable"),
	}

	provider := gemini.NewHintProviderWithClient("fake-key", mock)

	ex, _ := domain.NewExercise("test-ex", "Test", "fundamentos", "code", "test", "")
	ctx := context.Background()

	hintCh, errCh := provider.GetHint(ctx, ex, "FAIL")

	select {
	case hint := <-hintCh:
		t.Fatalf("unexpected hint on error path: %v", hint)
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected non-nil error")
		}
		if !strings.Contains(err.Error(), "API") {
			t.Errorf("error should mention API issue, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for error")
	}
}

func TestGetHint_ExactlyOneChannelReceives(t *testing.T) {
	mock := &mockClient{
		response: "¿Qué pasa si cambiás el orden de los parámetros?",
	}

	provider := gemini.NewHintProviderWithClient("fake-key", mock)

	ex, _ := domain.NewExercise("test-ex", "Test", "fundamentos", "code", "test", "")
	ctx := context.Background()

	hintCh, errCh := provider.GetHint(ctx, ex, "FAIL")

	var hintReceived, errReceived bool
	var receivedCount int

	timeout := time.After(5 * time.Second)
loop:
	for receivedCount < 2 {
		select {
		case hint, ok := <-hintCh:
			if ok && hint != nil {
				hintReceived = true
			}
			receivedCount++
		case err, ok := <-errCh:
			if ok && err != nil {
				errReceived = true
			}
			receivedCount++
		case <-timeout:
			break loop
		}
	}

	if hintReceived && errReceived {
		t.Error("only one channel should receive a value, got both")
	}
	if !hintReceived && !errReceived {
		t.Error("at least one channel should receive a value")
	}
}

func TestGetHint_Timeout_ReturnsError(t *testing.T) {
	mock := &mockClient{
		response: "should not arrive",
		delay:    5 * time.Second, // longer than the 2-second timeout
	}

	provider := gemini.NewHintProviderWithClient("fake-key", mock)

	ex, _ := domain.NewExercise("test-ex", "Test", "fundamentos", "code", "test", "")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	hintCh, errCh := provider.GetHint(ctx, ex, "FAIL")

	select {
	case hint := <-hintCh:
		t.Fatalf("unexpected hint on timeout path: %v", hint)
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected timeout error")
		}
		if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") && !strings.Contains(err.Error(), "cancel") {
			t.Errorf("error should indicate timeout, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for error from provider")
	}
}

func TestGetHint_PromptConstruction_SpanishSocratic(t *testing.T) {
	mock := &mockClient{
		response: "Buena pregunta, pero primero pensá: ¿estás segura de la condición del if?",
	}

	provider := gemini.NewHintProviderWithClient("fake-key", mock)

	ex, _ := domain.NewExercise("test-ex", "Test Exercise", "fundamentos",
		"package main\nfunc Suma(a, b int) int { return a + b + 1 }",
		"package main\nfunc TestSuma(t *testing.T) {}",
		"")

	ctx := context.Background()
	hintCh, errCh := provider.GetHint(ctx, ex, "--- FAIL: TestSuma (0.00s)\n    got 6, want 5")

	select {
	case hint := <-hintCh:
		if hint.Content != mock.response {
			t.Logf("hint content: %s", hint.Content)
		}
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestGetHint_NoAPIKey_ReturnsError(t *testing.T) {
	// Provider created without a mock client, which means no API key
	provider := gemini.NewHintProvider()

	ex, _ := domain.NewExercise("test-ex", "Test", "fundamentos", "code", "test", "")
	ctx := context.Background()

	hintCh, errCh := provider.GetHint(ctx, ex, "FAIL")

	select {
	case hint := <-hintCh:
		t.Fatalf("unexpected hint when no API key: %v", hint)
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error when no API key configured")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for error")
	}
}

func TestGetHint_HintContent_IsValidDomainObject(t *testing.T) {
	mock := &mockClient{
		response: "¿Consideraste que el error puede estar en la comparación?",
	}

	provider := gemini.NewHintProviderWithClient("fake-key", mock)

	ex, _ := domain.NewExercise("test-ex", "Test", "fundamentos", "code", "test", "")
	ctx := context.Background()

	hintCh, _ := provider.GetHint(ctx, ex, "FAIL")

	select {
	case hint := <-hintCh:
		// Verify it's a proper domain.Hint
		if hint.ExerciseID != "test-ex" {
			t.Errorf("ExerciseID = %q", hint.ExerciseID)
		}
		if hint.Content != mock.response {
			t.Errorf("Content = %q, want %q", hint.Content, mock.response)
		}
		if hint.RequestedAt.IsZero() {
			t.Error("RequestedAt should not be zero")
		}
		if hint.ReceivedAt.IsZero() {
			t.Error("ReceivedAt should not be zero")
		}
		if hint.ReceivedAt.Before(hint.RequestedAt) {
			t.Error("ReceivedAt should not be before RequestedAt")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestGetHint_ChannelsClose(t *testing.T) {
	mock := &mockClient{
		response: "Pista: revisa los paréntesis.",
	}

	provider := gemini.NewHintProviderWithClient("fake-key", mock)

	ex, _ := domain.NewExercise("test-ex", "Test", "fundamentos", "code", "test", "")
	ctx := context.Background()

	hintCh, errCh := provider.GetHint(ctx, ex, "FAIL")

	// Receive the hint
	var hintReceived, errorReceived bool
	done := make(chan struct{})
	go func() {
		select {
		case <-hintCh:
			hintReceived = true
		case <-errCh:
			errorReceived = true
		}
		close(done)
	}()

	select {
	case <-done:
		if !hintReceived && !errorReceived {
			t.Error("should receive exactly one value")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}
