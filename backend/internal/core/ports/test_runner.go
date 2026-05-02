package ports

import (
	"context"

	"godojo/internal/core/domain"
)

// TestRunner executes go test in an exercise workspace.
type TestRunner interface {
	// Run executes go test -json in the given directory and returns parsed results.
	Run(ctx context.Context, exerciseDir string) (*domain.TestResult, error)
}
