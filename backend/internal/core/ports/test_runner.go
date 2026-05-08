package ports

import (
	"context"

	"godojo/internal/core/domain"
)

// TestRunner defines the current test execution contract.
// Migration note: the long-term direction is Gemma's built-in CodeExecution tool.
type TestRunner interface {
	// Run executes go test -json in the given directory and returns parsed results.
	Run(ctx context.Context, exerciseDir string) (*domain.TestResult, error)
}
