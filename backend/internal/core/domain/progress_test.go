package domain_test

import (
	"testing"
	"time"

	"godojo/internal/core/domain"
)

func TestNewProgress(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		exerciseID  string
		topicSlug   string
		status      string
		attempts    int
		completedAt *time.Time
		wantErr     bool
	}{
		{
			name:        "valid not_started progress",
			exerciseID:  "variables-basicas",
			topicSlug:   "variables",
			status:      "not_started",
			attempts:    0,
			completedAt: nil,
			wantErr:     false,
		},
		{
			name:        "valid in_progress",
			exerciseID:  "funciones",
			topicSlug:   "funciones",
			status:      "in_progress",
			attempts:    1,
			completedAt: nil,
			wantErr:     false,
		},
		{
			name:        "valid completed with timestamp",
			exerciseID:  "slices",
			topicSlug:   "slices",
			status:      "completed",
			attempts:    2,
			completedAt: &now,
			wantErr:     false,
		},
		{
			name:        "valid skipped",
			exerciseID:  "maps",
			topicSlug:   "maps",
			status:      "skipped",
			attempts:    0,
			completedAt: nil,
			wantErr:     false,
		},
		{
			name:        "empty exercise id",
			exerciseID:  "",
			topicSlug:   "topic",
			status:      "not_started",
			attempts:    0,
			completedAt: nil,
			wantErr:     true,
		},
		{
			name:        "empty topic slug is valid (populated later)",
			exerciseID:  "ex",
			topicSlug:   "",
			status:      "not_started",
			attempts:    0,
			completedAt: nil,
			wantErr:     false,
		},
		{
			name:        "invalid status",
			exerciseID:  "ex",
			topicSlug:   "topic",
			status:      "pending",
			attempts:    0,
			completedAt: nil,
			wantErr:     true,
		},
		{
			name:        "negative attempts",
			exerciseID:  "ex",
			topicSlug:   "topic",
			status:      "not_started",
			attempts:    -1,
			completedAt: nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := domain.NewProgress(tt.exerciseID, tt.topicSlug, tt.status, tt.attempts, tt.completedAt)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil, p=%+v", p)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if p.ExerciseID != tt.exerciseID {
				t.Errorf("exerciseID = %q, want %q", p.ExerciseID, tt.exerciseID)
			}
			if p.TopicSlug != tt.topicSlug {
				t.Errorf("topicSlug = %q, want %q", p.TopicSlug, tt.topicSlug)
			}
			if string(p.Status) != tt.status {
				t.Errorf("status = %q, want %q", p.Status, tt.status)
			}
			if p.Attempts != tt.attempts {
				t.Errorf("attempts = %d, want %d", p.Attempts, tt.attempts)
			}
		})
	}
}
