package services

import (
	"context"
	"sync"

	"godojo/internal/core/domain"
	"godojo/internal/core/ports"
)

// HintService coordinates async hint requests with rate limiting.
type HintService struct {
	provider ports.HintProvider
	// Rate limiting: max 3 hints per exercise per session.
	mu           sync.Mutex
	hintCounts   map[string]int // exercise slug → hint count
	maxHints     int
}

// NewHintService creates a new HintService with the given provider.
func NewHintService(provider ports.HintProvider) *HintService {
	return &HintService{
		provider:   provider,
		hintCounts: make(map[string]int),
		maxHints:   3,
	}
}

// RequestHint requests a Socratic hint for a failed exercise.
// Returns nil channels if rate limited or exercise is nil.
func (s *HintService) RequestHint(exercise *domain.Exercise, testOutput string) (<-chan *domain.Hint, <-chan error) {
	if exercise == nil {
		return nil, nil
	}

	s.mu.Lock()
	count := s.hintCounts[exercise.Slug]
	if count >= s.maxHints {
		s.mu.Unlock()
		return nil, nil // rate limited
	}
	s.hintCounts[exercise.Slug] = count + 1
	s.mu.Unlock()

	ctx := context.Background()
	return s.provider.GetHint(ctx, exercise, testOutput)
}

// IsAvailable checks if the hint provider is reachable.
// For the mock, we check a simple flag. Real implementation will probe Gemini.
func (s *HintService) IsAvailable() bool {
	// Simple availability check: if provider is not nil, it's "available"
	// Real implementation in Phase 3 will add health checks
	return s.provider != nil
}
