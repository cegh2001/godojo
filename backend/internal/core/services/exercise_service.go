package services

import (
	"context"
	"fmt"

	"godojo/internal/core/domain"
	"godojo/internal/core/ports"
)

// ExerciseService orchestrates the exercise lifecycle.
type ExerciseService struct {
	repo ports.ExerciseRepository
}

// NewExerciseService creates a new ExerciseService with the given repository.
func NewExerciseService(repo ports.ExerciseRepository) *ExerciseService {
	return &ExerciseService{repo: repo}
}

// StartExercise fetches an exercise by slug.
// The slug is globally unique across all topics for MVP simplicity.
func (s *ExerciseService) StartExercise(slug string) (*domain.Exercise, error) {
	if slug == "" {
		return nil, fmt.Errorf("exercise slug is required")
	}

	// For the MVP, search across topics. The repository implementation
	// in Phase 3 will handle efficient lookup.
	// We use an empty topicSlug since the mock repo matches by exercise slug alone.
	ctx := context.Background()
	ex, err := s.repo.GetBySlug(ctx, "", slug)
	if err != nil {
		return nil, fmt.Errorf("exercise %q not found: %w", slug, err)
	}
	return ex, nil
}

// ValidateExercise checks if the test result indicates a passing exercise.
func (s *ExerciseService) ValidateExercise(slug string, testResult *domain.TestResult) (bool, error) {
	if slug == "" {
		return false, fmt.Errorf("exercise slug is required")
	}
	if testResult == nil {
		return false, fmt.Errorf("test result is required")
	}
	return testResult.Passed, nil
}

// GetExercisesByTopic returns all exercises for a given topic.
func (s *ExerciseService) GetExercisesByTopic(topicSlug string) ([]*domain.ExerciseRef, error) {
	if topicSlug == "" {
		return nil, fmt.Errorf("topic slug is required")
	}
	ctx := context.Background()
	refs, err := s.repo.ListByTopic(ctx, topicSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to list exercises for topic %q: %w", topicSlug, err)
	}
	return refs, nil
}
