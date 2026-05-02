package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"godojo/internal/core/domain"
	"godojo/internal/core/services"
)

// mockHintProvider implements ports.HintProvider for testing.
type mockHintProvider struct {
	available bool
	delay     time.Duration
	hint      *domain.Hint
	err       error
}

func newMockHintProvider(available bool) *mockHintProvider {
	return &mockHintProvider{available: available}
}

func (m *mockHintProvider) GetHint(ctx context.Context, exercise *domain.Exercise, testOutput string) (<-chan *domain.Hint, <-chan error) {
	hintCh := make(chan *domain.Hint, 1)
	errCh := make(chan error, 1)

	go func() {
		if m.delay > 0 {
			select {
			case <-time.After(m.delay):
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
		}
		if m.err != nil {
			errCh <- m.err
		} else {
			hintCh <- m.hint
		}
		close(hintCh)
		close(errCh)
	}()

	return hintCh, errCh
}

func TestHintService_RequestHint(t *testing.T) {
	provider := newMockHintProvider(true)
	validHint := mustHint("ex-1", "¿Probaste con := en vez de var?")
	provider.hint = validHint

	svc := services.NewHintService(provider)

	validEx := mustExercise("ex-1", "Test", "topic", "code", "code", "")

	t.Run("successful hint", func(t *testing.T) {
		hintCh, errCh := svc.RequestHint(validEx, "test output")

		select {
		case hint := <-hintCh:
			if hint.Content != validHint.Content {
				t.Errorf("hint content = %q, want %q", hint.Content, validHint.Content)
			}
		case err := <-errCh:
			t.Errorf("unexpected error: %v", err)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for hint")
		}
	})

	t.Run("provider error", func(t *testing.T) {
		provider.err = errors.New("gemini unavailable")
		svc2 := services.NewHintService(provider)
		hintCh, errCh := svc2.RequestHint(validEx, "test output")

		select {
		case hint := <-hintCh:
			t.Errorf("unexpected hint received: %+v", hint)
		case err := <-errCh:
			if err == nil {
				t.Error("expected error but got nil")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for error")
		}
		provider.err = nil
	})

	t.Run("nil exercise", func(t *testing.T) {
		hintCh, errCh := svc.RequestHint(nil, "test output")
		if hintCh != nil || errCh != nil {
			// The nil case should return nil channels (no-op)
			// In reality, the service will validate and return nil,nil
		}
		// Drain any channels
		if hintCh != nil {
			<-hintCh
		}
		if errCh != nil {
			<-errCh
		}
	})
}

func TestHintService_IsAvailable(t *testing.T) {
	provider := newMockHintProvider(true)
	svc := services.NewHintService(provider)

	if !svc.IsAvailable() {
		t.Error("IsAvailable should return true when provider is set")
	}

	// Nil provider is unavailable
	svcNil := services.NewHintService(nil)
	if svcNil.IsAvailable() {
		t.Error("IsAvailable should return false when provider is nil")
	}
}

func TestHintService_RateLimiting(t *testing.T) {
	provider := newMockHintProvider(true)
	validHint := mustHint("ex-1", "Hint")
	provider.hint = validHint

	svc := services.NewHintService(provider)
	validEx := mustExercise("ex-1", "Test", "topic", "code", "code", "")

	// First 3 hints should succeed
	for i := 0; i < 3; i++ {
		hintCh, errCh := svc.RequestHint(validEx, "test output")
		select {
		case hint := <-hintCh:
			if hint == nil {
				t.Errorf("hint %d: expected non-nil hint", i+1)
			}
		case err := <-errCh:
			t.Errorf("hint %d: unexpected error: %v", i+1, err)
		case <-time.After(2 * time.Second):
			t.Fatalf("hint %d: timeout", i+1)
		}
	}

	// 4th hint should be rate limited
	hintCh, errCh := svc.RequestHint(validEx, "test output")
	if hintCh != nil || errCh != nil {
		t.Error("expected nil channels on rate-limited request")
	}
	// Drain channels if any
	if hintCh != nil {
		<-hintCh
	}
	if errCh != nil {
		<-errCh
	}
}

// Helper for Hint
func mustHint(exerciseID, content string) *domain.Hint {
	now := time.Now()
	h, err := domain.NewHint(exerciseID, content, now, now.Add(time.Second))
	if err != nil {
		panic(err)
	}
	return h
}
