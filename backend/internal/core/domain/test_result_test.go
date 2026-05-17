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
			// New fields default to empty via constructor
			if tr.Stdout != "" {
				t.Errorf("stdout = %q, want empty", tr.Stdout)
			}
			if tr.Stderr != "" {
				t.Errorf("stderr = %q, want empty", tr.Stderr)
			}
		})
	}
}

func TestTestResult_StdoutStderr_StructLiteral(t *testing.T) {
	tests := []struct {
		name       string
		passed     bool
		output     string
		stdout     string
		stderr     string
		duration   time.Duration
		wantStdout string
		wantStderr string
	}{
		{
			name:       "both stdout and stderr populated",
			passed:     true,
			output:     "combined output",
			stdout:     `{"Action":"pass"}`,
			stderr:     "warning: deprecated flag",
			duration:   100 * time.Millisecond,
			wantStdout: `{"Action":"pass"}`,
			wantStderr: "warning: deprecated flag",
		},
		{
			name:       "stdout only, stderr empty",
			passed:     true,
			output:     "ok",
			stdout:     "test output",
			stderr:     "",
			duration:   50 * time.Millisecond,
			wantStdout: "test output",
			wantStderr: "",
		},
		{
			name:       "stderr only, stdout empty (compile error)",
			passed:     false,
			output:     "compilation failed",
			stdout:     "",
			stderr:     "syntax error at line 10",
			duration:   10 * time.Millisecond,
			wantStdout: "",
			wantStderr: "syntax error at line 10",
		},
		{
			name:       "both empty (no output)",
			passed:     false,
			output:     "",
			stdout:     "",
			stderr:     "",
			duration:   0,
			wantStdout: "",
			wantStderr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := domain.TestResult{
				Passed:   tt.passed,
				Output:   tt.output,
				Stdout:   tt.stdout,
				Stderr:   tt.stderr,
				Duration: tt.duration,
			}
			if tr.Stdout != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", tr.Stdout, tt.wantStdout)
			}
			if tr.Stderr != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", tr.Stderr, tt.wantStderr)
			}
		})
	}
}
