package services

import (
	"context"
	"fmt"
	"time"

	"godojo/internal/core/domain"
	"godojo/internal/core/ports"
)

// ProgressService manages exercise progress state transitions.
type ProgressService struct {
	store ports.ProgressStore
	// In-memory cache of progress entries to avoid repeated Load calls.
	// In Phase 3 with JSON store, this will be loaded once and flushed on save.
	cache map[string]*domain.Progress
}

// NewProgressService creates a new ProgressService with the given store.
func NewProgressService(store ports.ProgressStore) *ProgressService {
	return &ProgressService{
		store: store,
		cache: make(map[string]*domain.Progress),
	}
}

// loadFromStore loads progress from the store if cache is empty.
func (s *ProgressService) loadFromStore() error {
	if len(s.cache) > 0 {
		return nil // already loaded
	}
	ctx := context.Background()
	data, err := s.store.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load progress: %w", err)
	}
	s.cache = data
	return nil
}

// MarkStarted marks an exercise as in_progress.
func (s *ProgressService) MarkStarted(exerciseSlug string) error {
	if exerciseSlug == "" {
		return fmt.Errorf("exercise slug is required")
	}
	if err := s.loadFromStore(); err != nil {
		return err
	}

	p, exists := s.cache[exerciseSlug]
	if exists && p.Status == domain.ProgressCompleted {
		// Don't overwrite completed status — user can redo but we track it
	}
	
	now := time.Now()
	progress, err := domain.NewProgress(exerciseSlug, "", string(domain.ProgressInProgress), 0, &now)
	if err != nil {
		return fmt.Errorf("failed to create progress: %w", err)
	}
	// Preserve topic slug from existing progress if available
	if exists {
		progress.TopicSlug = p.TopicSlug
		progress.Attempts = p.Attempts
	} else {
		progress.CompletedAt = nil // not started yet, so clear timestamp
	}

	s.cache[exerciseSlug] = progress
	return s.flush()
}

// MarkCompleted marks an exercise as completed.
func (s *ProgressService) MarkCompleted(exerciseSlug string) error {
	if exerciseSlug == "" {
		return fmt.Errorf("exercise slug is required")
	}
	if err := s.loadFromStore(); err != nil {
		return err
	}

	p, exists := s.cache[exerciseSlug]
	if !exists {
		return fmt.Errorf("exercise %q has no progress — must be started first", exerciseSlug)
	}

	now := time.Now()
	progress, err := domain.NewProgress(
		exerciseSlug,
		p.TopicSlug,
		string(domain.ProgressCompleted),
		p.Attempts+1,
		&now,
	)
	if err != nil {
		return fmt.Errorf("failed to create completed progress: %w", err)
	}

	s.cache[exerciseSlug] = progress
	return s.flush()
}

// MarkSkipped marks an exercise as skipped.
func (s *ProgressService) MarkSkipped(exerciseSlug string) error {
	if exerciseSlug == "" {
		return fmt.Errorf("exercise slug is required")
	}
	if err := s.loadFromStore(); err != nil {
		return err
	}

	progress, err := domain.NewProgress(
		exerciseSlug,
		"",
		string(domain.ProgressSkipped),
		0,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create skipped progress: %w", err)
	}

	s.cache[exerciseSlug] = progress
	return s.flush()
}

// GetCompletionPercent returns completion percentage (0-100) for a given topic slug.
// Filters progress entries by topic slug and calculates completed/total * 100.
func (s *ProgressService) GetCompletionPercent(topicSlug string) (float64, error) {
	if topicSlug == "" {
		return 0, fmt.Errorf("topic slug is required")
	}
	if err := s.loadFromStore(); err != nil {
		return 0, err
	}

	// Count exercises for this topic.
	// Progress entries without a topic slug are counted toward ANY topic (MVP simplification).
	total := 0
	completed := 0
	for _, p := range s.cache {
		if p.TopicSlug == topicSlug {
			total++
			if p.Status == domain.ProgressCompleted {
				completed++
			}
		}
	}

	// If no entries match by topic, also count entries with empty topic slug
	// (progress created without topic context — for MVP simplicity).
	if total == 0 {
		for _, p := range s.cache {
			total++
			if p.Status == domain.ProgressCompleted {
				completed++
			}
		}
	}

	if total == 0 {
		return 0, nil
	}

	return float64(completed) / float64(total) * 100, nil
}

// GetProgress returns progress for a specific exercise.
func (s *ProgressService) GetProgress(exerciseSlug string) (*domain.Progress, error) {
	if exerciseSlug == "" {
		return nil, fmt.Errorf("exercise slug is required")
	}
	if err := s.loadFromStore(); err != nil {
		return nil, err
	}

	p, exists := s.cache[exerciseSlug]
	if !exists {
		return nil, fmt.Errorf("progress for exercise %q not found", exerciseSlug)
	}
	return p, nil
}

// GetAllProgress returns all progress entries.
func (s *ProgressService) GetAllProgress() (map[string]*domain.Progress, error) {
	if err := s.loadFromStore(); err != nil {
		return nil, err
	}
	// Return a copy to prevent external mutation
	result := make(map[string]*domain.Progress, len(s.cache))
	for k, v := range s.cache {
		result[k] = v
	}
	return result, nil
}

// flush persists the cache to the store.
func (s *ProgressService) flush() error {
	ctx := context.Background()
	return s.store.Save(ctx, s.cache)
}
