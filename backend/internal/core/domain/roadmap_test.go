package domain_test

import (
	"testing"

	"godojo/internal/core/domain"
)

func TestNewExerciseRef(t *testing.T) {
	tests := []struct {
		name       string
		slug       string
		title      string
		difficulty string
		wantErr    bool
	}{
		{
			name:       "valid easy exercise",
			slug:       "variables-basicas",
			title:      "Variables Básicas",
			difficulty: "easy",
			wantErr:    false,
		},
		{
			name:       "valid medium exercise",
			slug:       "funciones-avanzadas",
			title:      "Funciones Avanzadas",
			difficulty: "medium",
			wantErr:    false,
		},
		{
			name:       "valid hard exercise",
			slug:       "concurrencia",
			title:      "Concurrencia",
			difficulty: "hard",
			wantErr:    false,
		},
		{
			name:       "empty slug",
			slug:       "",
			title:      "Título",
			difficulty: "easy",
			wantErr:    true,
		},
		{
			name:       "empty title",
			slug:       "slug",
			title:      "",
			difficulty: "easy",
			wantErr:    true,
		},
		{
			name:       "empty difficulty",
			slug:       "slug",
			title:      "Título",
			difficulty: "",
			wantErr:    true,
		},
		{
			name:       "invalid difficulty",
			slug:       "slug",
			title:      "Título",
			difficulty: "impossible",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := domain.NewExerciseRef(tt.slug, tt.title, tt.difficulty)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil, ref=%+v", ref)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if ref.Slug != tt.slug {
				t.Errorf("slug = %q, want %q", ref.Slug, tt.slug)
			}
			if ref.Title != tt.title {
				t.Errorf("title = %q, want %q", ref.Title, tt.title)
			}
			if string(ref.Difficulty) != tt.difficulty {
				t.Errorf("difficulty = %q, want %q", ref.Difficulty, tt.difficulty)
			}
		})
	}
}

func TestNewPhase(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		title   string
		topics  []*domain.Topic
		wantErr bool
	}{
		{
			name:  "valid phase with topics",
			id:    "fase-1",
			title: "Fundamentos",
			topics: []*domain.Topic{
				mustTopic(t, "variables", "Variables", "Aprender variables"),
			},
			wantErr: false,
		},
		{
			name:    "valid phase with no topics",
			id:      "fase-2",
			title:   "Estructuras",
			topics:  nil,
			wantErr: false,
		},
		{
			name:    "empty id",
			id:      "",
			title:   "Título",
			topics:  nil,
			wantErr: true,
		},
		{
			name:    "empty title",
			id:      "id",
			title:   "",
			topics:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase, err := domain.NewPhase(tt.id, tt.title, tt.topics)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil, phase=%+v", phase)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if phase.ID != tt.id {
				t.Errorf("id = %q, want %q", phase.ID, tt.id)
			}
			if phase.Title != tt.title {
				t.Errorf("title = %q, want %q", phase.Title, tt.title)
			}
		})
	}
}

func TestNewTopic(t *testing.T) {
	tests := []struct {
		name      string
		slug      string
		title     string
		desc      string
		exercises []*domain.ExerciseRef
		wantErr   bool
	}{
		{
			name:  "valid topic with exercises",
			slug:  "variables",
			title: "Variables",
			desc:  "Aprender a declarar variables",
			exercises: []*domain.ExerciseRef{
				mustExerciseRef(t, "var-basicas", "Variables Básicas", "easy"),
			},
			wantErr: false,
		},
		{
			name:      "valid topic with no exercises",
			slug:      "intro",
			title:     "Introducción",
			desc:      "Introducción a Go",
			exercises: nil,
			wantErr:   false,
		},
		{
			name:      "empty slug",
			slug:      "",
			title:     "Título",
			desc:      "Descripción",
			exercises: nil,
			wantErr:   true,
		},
		{
			name:      "empty title",
			slug:      "slug",
			title:     "",
			desc:      "Descripción",
			exercises: nil,
			wantErr:   true,
		},
		{
			name:      "empty description ok",
			slug:      "slug",
			title:     "Título",
			desc:      "",
			exercises: nil,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic, err := domain.NewTopic(tt.slug, tt.title, tt.desc, tt.exercises, nil)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil, topic=%+v", topic)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if topic.Slug != tt.slug {
				t.Errorf("slug = %q, want %q", topic.Slug, tt.slug)
			}
			if topic.Title != tt.title {
				t.Errorf("title = %q, want %q", topic.Title, tt.title)
			}
		})
	}
}

func TestNewRoadmap(t *testing.T) {
	emptyPhases := []*domain.Phase{}
	validPhase := mustPhase(t, "fase-1", "Fundamentos", nil)

	tests := []struct {
		name    string
		phases  []*domain.Phase
		wantErr bool
	}{
		{
			name:    "valid roadmap with phases",
			phases:  []*domain.Phase{validPhase},
			wantErr: false,
		},
		{
			name:    "empty phases slice",
			phases:  emptyPhases,
			wantErr: true,
		},
		{
			name:    "nil phases",
			phases:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm, err := domain.NewRoadmap(tt.phases)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil, roadmap=%+v", rm)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(rm.Phases) != len(tt.phases) {
				t.Errorf("phases count = %d, want %d", len(rm.Phases), len(tt.phases))
			}
		})
	}
}

// Helpers to create valid entities in test setup (panics on error — only for test data).

func mustExerciseRef(t *testing.T, slug, title, difficulty string) *domain.ExerciseRef {
	t.Helper()
	ref, err := domain.NewExerciseRef(slug, title, difficulty)
	if err != nil {
		t.Fatalf("mustExerciseRef: %v", err)
	}
	return ref
}

func mustTopic(t *testing.T, slug, title, desc string) *domain.Topic {
	t.Helper()
	topic, err := domain.NewTopic(slug, title, desc, nil, nil)
	if err != nil {
		t.Fatalf("mustTopic: %v", err)
	}
	return topic
}

func mustPhase(t *testing.T, id, title string, topics []*domain.Topic) *domain.Phase {
	t.Helper()
	phase, err := domain.NewPhase(id, title, topics)
	if err != nil {
		t.Fatalf("mustPhase: %v", err)
	}
	return phase
}
