package domain_test

import (
	"testing"
	"time"

	"godojo/internal/core/domain"
)

func TestNewHint(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		exerciseID string
		content    string
		requested  time.Time
		received   time.Time
		wantErr    bool
	}{
		{
			name:       "valid hint",
			exerciseID: "variables-basicas",
			content:    "¿Probaste a declarar la variable con := en vez de var?",
			requested:  now,
			received:   now.Add(2 * time.Second),
			wantErr:    false,
		},
		{
			name:       "empty exercise id",
			exerciseID: "",
			content:    "contenido",
			requested:  now,
			received:   now,
			wantErr:    true,
		},
		{
			name:       "empty content",
			exerciseID: "ex",
			content:    "",
			requested:  now,
			received:   now,
			wantErr:    true,
		},
		{
			name:       "received before requested",
			exerciseID: "ex",
			content:    "contenido",
			requested:  now,
			received:   now.Add(-1 * time.Hour),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := domain.NewHint(tt.exerciseID, tt.content, tt.requested, tt.received)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil, h=%+v", h)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if h.ExerciseID != tt.exerciseID {
				t.Errorf("exerciseID = %q, want %q", h.ExerciseID, tt.exerciseID)
			}
			if h.Content != tt.content {
				t.Errorf("content = %q, want %q", h.Content, tt.content)
			}
			if !h.RequestedAt.Equal(tt.requested) {
				t.Errorf("requestedAt mismatch")
			}
			if !h.ReceivedAt.Equal(tt.received) {
				t.Errorf("receivedAt mismatch")
			}
		})
	}
}
