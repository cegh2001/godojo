package ports

import (
	"context"

	"godojo/internal/core/domain"
)

// Deprecated: HintProvider will be removed in sensei-first v2.
// Replaced by SenseiService's agentic Socratic tutoring.
type HintProvider interface {
	// GetHint requests a hint for a failed exercise. Returns channels for async delivery.
	// The hint channel delivers the hint when ready, the error channel signals failures.
	GetHint(ctx context.Context, exercise *domain.Exercise, testOutput string) (<-chan *domain.Hint, <-chan error)
}
