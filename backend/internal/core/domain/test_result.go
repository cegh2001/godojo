package domain

import (
	"fmt"
	"time"
)

// TestResult is a value object capturing test execution results.
// Migration note: the long-term direction is Gemma's CodeExecution tool output.
type TestResult struct {
	Passed   bool
	Output   string          // combined summary (backward-compatible)
	Stdout   string          // raw stdout captured separately
	Stderr   string          // raw stderr captured separately
	Duration time.Duration
}

// NewTestResult creates a validated TestResult.
func NewTestResult(passed bool, output string, duration time.Duration) (*TestResult, error) {
	if duration < 0 {
		return nil, fmt.Errorf("duration cannot be negative")
	}
	return &TestResult{
		Passed:   passed,
		Output:   output,
		Duration: duration,
	}, nil
}
