package domain

import (
	"fmt"
	"time"
)

// ProgressStatus represents the status of an exercise attempt.
type ProgressStatus string

const (
	ProgressNotStarted ProgressStatus = "not_started"
	ProgressInProgress ProgressStatus = "in_progress"
	ProgressCompleted  ProgressStatus = "completed"
	ProgressSkipped    ProgressStatus = "skipped"
)

// validStatuses maps valid progress statuses.
var validStatuses = map[ProgressStatus]bool{
	ProgressNotStarted: true,
	ProgressInProgress: true,
	ProgressCompleted:  true,
	ProgressSkipped:    true,
}

// Progress tracks user progress on a specific exercise.
type Progress struct {
	ExerciseID  string         `json:"exercise_id"`
	TopicSlug   string         `json:"topic_slug"`
	Status      ProgressStatus `json:"status"`
	Attempts    int            `json:"attempts"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}

// NewProgress creates a validated Progress record.
func NewProgress(exerciseID, topicSlug, status string, attempts int, completedAt *time.Time) (*Progress, error) {
	if exerciseID == "" {
		return nil, fmt.Errorf("exercise id is required")
	}
	// topicSlug is optional at progress creation time — populated later by roadmap context
	s := ProgressStatus(status)
	if !validStatuses[s] {
		return nil, fmt.Errorf("invalid progress status %q: must be not_started, in_progress, completed, or skipped", status)
	}
	if attempts < 0 {
		return nil, fmt.Errorf("attempts cannot be negative")
	}
	return &Progress{
		ExerciseID:  exerciseID,
		TopicSlug:   topicSlug,
		Status:      s,
		Attempts:    attempts,
		CompletedAt: completedAt,
	}, nil
}
