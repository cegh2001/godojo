package ports

import (
	"context"

	"godojo/internal/core/domain"
)

// Deprecated: TestRunner will be removed in sensei-first v2.
// Replaced by Gemma's built-in CodeExecution tool (Phase 2).
type TestRunner interface {
	// Run executes go test -json in the given directory and returns parsed results.
	Run(ctx context.Context, exerciseDir string) (*domain.TestResult, error)
}
