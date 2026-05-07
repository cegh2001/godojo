package gemini_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"godojo/internal/adapters/gemini"
	"godojo/internal/core/domain"
)

func TestHintProvider_LiveSmoke(t *testing.T) {
	requireLiveGemini(t)
	configureGemma4LiveEnv(t)

	provider := gemini.NewHintProvider()
	exercise, err := domain.NewExercise(
		"live-hint-smoke",
		"Live Hint Smoke",
		"fundamentos",
		"package main\n\nfunc Sumar(a, b int) int {\n\treturn a - b\n}\n",
		"package main\n\nimport \"testing\"\n\nfunc TestSumar(t *testing.T) {\n\tif got := Sumar(2, 3); got != 5 {\n\t\tt.Fatalf(\"got %d, want 5\", got)\n\t}\n}\n",
		"",
	)
	if err != nil {
		t.Fatalf("NewExercise() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	hintCh, errCh := provider.GetHint(ctx, exercise, "--- FAIL: TestSumar (0.00s)\n    got -1, want 5")

	select {
	case hint := <-hintCh:
		if hint == nil {
			t.Fatal("expected non-nil hint")
		}
		text := strings.TrimSpace(hint.Content)
		if text == "" {
			t.Fatal("expected non-empty hint text")
		}
		assertNoGeminiFallbackError(t, text)
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected either hint or error, got nil error")
		}
		t.Fatalf("GetHint() error: %v", err)
	case <-time.After(50 * time.Second):
		t.Fatal("timeout waiting for live hint")
	}
}
