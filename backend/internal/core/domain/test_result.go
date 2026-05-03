// Deprecated: TestResult will be removed in sensei-first v2.
// Replaced by Gemma's CodeExecution tool output (Phase 2).
package domain

import (
	"fmt"
	"time"
)

// TestResult is a value object capturing test execution results.
type TestResult struct {
	Passed   bool
	Output   string
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
