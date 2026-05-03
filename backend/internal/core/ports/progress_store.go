package ports

import (
	"context"

	"godojo/internal/core/domain"
)

// Deprecated: ProgressStore will be removed in sensei-first v2.
// Replaced by chat session history as the source of truth for learning progress.
type ProgressStore interface {
	// Load returns all progress entries from persistent storage.
	Load(ctx context.Context) (map[string]*domain.Progress, error)

	// Save persists all progress entries to storage.
	Save(ctx context.Context, progress map[string]*domain.Progress) error

	// GetBySlug retrieves progress for a specific exercise.
	GetBySlug(ctx context.Context, slug string) (*domain.Progress, error)
}
