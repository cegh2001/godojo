package ports

import (
	"context"

	"godojo/internal/core/domain"
)

// ExerciseRepository provides access to exercise definitions and file generation.
type ExerciseRepository interface {
	// GetBySlug retrieves a single exercise by its slug within a topic.
	GetBySlug(ctx context.Context, topicSlug, exerciseSlug string) (*domain.Exercise, error)

	// ListByTopic returns all exercise references for a given topic.
	ListByTopic(ctx context.Context, topicSlug string) ([]*domain.ExerciseRef, error)

	// GenerateFiles writes ejercicio.go, ejercicio_test.go, and go.mod to the workspace.
	GenerateFiles(ctx context.Context, exercise *domain.Exercise, workspacePath string) error
}
