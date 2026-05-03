// Deprecated: Hint will be removed in sensei-first v2.
// Replaced by SenseiService's conversational Socratic guidance.
package domain

import (
	"fmt"
	"time"
)

// Hint is a value object representing a Socratic hint from the AI sensei.
type Hint struct {
	ExerciseID  string
	Content     string
	RequestedAt time.Time
	ReceivedAt  time.Time
}

// NewHint creates a validated Hint.
func NewHint(exerciseID, content string, requestedAt, receivedAt time.Time) (*Hint, error) {
	if exerciseID == "" {
		return nil, fmt.Errorf("exercise id is required")
	}
	if content == "" {
		return nil, fmt.Errorf("hint content is required")
	}
	if receivedAt.Before(requestedAt) {
		return nil, fmt.Errorf("received time cannot be before requested time")
	}
	return &Hint{
		ExerciseID:  exerciseID,
		Content:     content,
		RequestedAt: requestedAt,
		ReceivedAt:  receivedAt,
	}, nil
}
