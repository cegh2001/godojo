package domain_test

import (
	"testing"
	"time"

	"godojo/internal/core/domain"
)

func TestNewTestResult(t *testing.T) {
	tests := []struct {
		name     string
		passed   bool
		output   string
		duration time.Duration
		wantErr  bool
	}{
		{
			name:     "passing test result",
			passed:   true,
			output:   "ok  \tejercicio\t0.123s",
			duration: 123 * time.Millisecond,
			wantErr:  false,
		},
		{
			name:     "failing test result",
			passed:   false,
			output:   "--- FAIL: TestSuma (0.00s)\n    ejercicio_test.go:10: got 5, want 6",
			duration: 50 * time.Millisecond,
			wantErr:  false,
		},
		{
			name:     "empty output is valid",
			passed:   true,
			output:   "",
			duration: 0,
			wantErr:  false,
		},
		{
			name:     "negative duration is invalid",
			passed:   false,
			output:   "fail",
			duration: -1 * time.Second,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, err := domain.NewTestResult(tt.passed, tt.output, tt.duration)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil, tr=%+v", tr)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tr.Passed != tt.passed {
				t.Errorf("passed = %v, want %v", tr.Passed, tt.passed)
			}
			if tr.Output != tt.output {
				t.Errorf("output = %q, want %q", tr.Output, tt.output)
			}
			if tr.Duration != tt.duration {
				t.Errorf("duration = %v, want %v", tr.Duration, tt.duration)
			}
		})
	}
}
