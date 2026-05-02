package store

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"godojo/internal/core/domain"
)

// JSONProgressStore implements ports.ProgressStore using a local JSON file.
type JSONProgressStore struct {
	filePath string
	cache    map[string]*domain.Progress
	mu       sync.RWMutex
	loaded   bool
}

// NewJSONProgressStore creates a new JSONProgressStore that persists to the given file path.
func NewJSONProgressStore(filePath string) *JSONProgressStore {
	return &JSONProgressStore{
		filePath: filePath,
	}
}

// Load reads and deserializes progress from the JSON file.
// If the file doesn't exist or is empty, returns an empty map.
// If the file is corrupted, returns an error.
func (s *JSONProgressStore) Load(ctx context.Context) (map[string]*domain.Progress, error) {
	s.mu.RLock()
	if s.loaded {
		// Return a copy of the cache
		result := make(map[string]*domain.Progress, len(s.cache))
		for k, v := range s.cache {
			result[k] = v
		}
		s.mu.RUnlock()
		return result, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			s.cache = make(map[string]*domain.Progress)
			s.loaded = true
			return make(map[string]*domain.Progress), nil
		}
		return nil, err
	}

	// Handle empty file
	if len(data) == 0 {
		s.cache = make(map[string]*domain.Progress)
		s.loaded = true
		return make(map[string]*domain.Progress), nil
	}

	var progress map[string]*domain.Progress
	if err := json.Unmarshal(data, &progress); err != nil {
		return nil, err
	}

	s.cache = progress
	s.loaded = true

	// Return a copy
	result := make(map[string]*domain.Progress, len(s.cache))
	for k, v := range s.cache {
		result[k] = v
	}
	return result, nil
}

// Save atomically writes progress to the JSON file.
// It writes to a temporary file first, then renames to the target path.
func (s *JSONProgressStore) Save(ctx context.Context, progress map[string]*domain.Progress) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update cache
	s.cache = make(map[string]*domain.Progress, len(progress))
	for k, v := range progress {
		s.cache[k] = v
	}
	s.loaded = true

	data, err := json.MarshalIndent(progress, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, s.filePath); err != nil {
		// Try to clean up temp file
		os.Remove(tmpPath)
		return err
	}

	return nil
}

// GetBySlug retrieves progress for a specific exercise slug by checking the cache.
func (s *JSONProgressStore) GetBySlug(ctx context.Context, slug string) (*domain.Progress, error) {
	// Ensure cache is loaded
	_, err := s.Load(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.cache[slug]
	if !ok {
		return nil, nil
	}
	return p, nil
}
