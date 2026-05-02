package services_test

import (
	"context"
	"errors"
	"testing"

	"godojo/internal/core/domain"
	"godojo/internal/core/services"
)

// mockExerciseRepo implements ports.ExerciseRepository for testing.
type mockExerciseRepo struct {
	exercises     map[string]*domain.Exercise // key: "topicSlug/exerciseSlug"
	exercisesByTopic map[string][]*domain.ExerciseRef
	generateErr  error
}

func newMockExerciseRepo() *mockExerciseRepo {
	return &mockExerciseRepo{
		exercises:     make(map[string]*domain.Exercise),
		exercisesByTopic: make(map[string][]*domain.ExerciseRef),
	}
}

func (m *mockExerciseRepo) GetBySlug(ctx context.Context, topicSlug, exerciseSlug string) (*domain.Exercise, error) {
	// Match by exercise slug alone (global uniqueness assumed for MVP)
	for _, ex := range m.exercises {
		if ex.Slug == exerciseSlug {
			return ex, nil
		}
	}
	return nil, errors.New("exercise not found")
}

func (m *mockExerciseRepo) ListByTopic(ctx context.Context, topicSlug string) ([]*domain.ExerciseRef, error) {
	refs, ok := m.exercisesByTopic[topicSlug]
	if !ok {
		return nil, errors.New("topic not found")
	}
	return refs, nil
}

func (m *mockExerciseRepo) GenerateFiles(ctx context.Context, exercise *domain.Exercise, workspacePath string) error {
	return nil
}

func TestExerciseService_StartExercise(t *testing.T) {
	mockRepo := newMockExerciseRepo()
	validEx := mustExercise("variables-basicas", "Variables Básicas", "variables",
		"package main", "package main", "")
	mockRepo.exercises["variables-basicas"] = validEx

	svc := services.NewExerciseService(mockRepo)

	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{
			name:    "start existing exercise",
			slug:    "variables-basicas",
			wantErr: false,
		},
		{
			name:    "start non-existent exercise",
			slug:    "no-existe",
			wantErr: true,
		},
		{
			name:    "empty slug",
			slug:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ex, err := svc.StartExercise(tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil, ex=%+v", ex)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if ex.Slug != tt.slug {
				t.Errorf("slug = %q, want %q", ex.Slug, tt.slug)
			}
		})
	}
}

func TestExerciseService_ValidateExercise(t *testing.T) {
	mockRepo := newMockExerciseRepo()
	svc := services.NewExerciseService(mockRepo)

	passResult, _ := domain.NewTestResult(true, "PASS", 0)
	failResult, _ := domain.NewTestResult(false, "FAIL", 0)

	tests := []struct {
		name       string
		slug       string
		testResult *domain.TestResult
		wantPass   bool
		wantErr    bool
	}{
		{
			name:       "exercise passes",
			slug:       "ex-1",
			testResult: passResult,
			wantPass:   true,
			wantErr:    false,
		},
		{
			name:       "exercise fails",
			slug:       "ex-1",
			testResult: failResult,
			wantPass:   false,
			wantErr:    false,
		},
		{
			name:       "nil test result",
			slug:       "ex-1",
			testResult: nil,
			wantPass:   false,
			wantErr:    true,
		},
		{
			name:       "empty slug",
			slug:       "",
			testResult: passResult,
			wantPass:   false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, err := svc.ValidateExercise(tt.slug, tt.testResult)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got passed=%v", passed)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if passed != tt.wantPass {
				t.Errorf("passed = %v, want %v", passed, tt.wantPass)
			}
		})
	}
}

func TestExerciseService_GetExercisesByTopic(t *testing.T) {
	mockRepo := newMockExerciseRepo()
	ref1 := mustExerciseRef("ex-1", "Ejercicio 1", "easy")
	ref2 := mustExerciseRef("ex-2", "Ejercicio 2", "medium")
	mockRepo.exercisesByTopic["variables"] = []*domain.ExerciseRef{ref1, ref2}

	svc := services.NewExerciseService(mockRepo)

	tests := []struct {
		name      string
		topicSlug string
		wantLen   int
		wantErr   bool
	}{
		{
			name:      "topic with exercises",
			topicSlug: "variables",
			wantLen:   2,
			wantErr:   false,
		},
		{
			name:      "non-existent topic",
			topicSlug: "no-existe",
			wantLen:   0,
			wantErr:   true,
		},
		{
			name:      "empty slug",
			topicSlug: "",
			wantLen:   0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exercises, err := svc.GetExercisesByTopic(tt.topicSlug)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got %d exercises", len(exercises))
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(exercises) != tt.wantLen {
				t.Errorf("got %d exercises, want %d", len(exercises), tt.wantLen)
			}
		})
	}
}

// Helper functions for tests.

func mustExercise(slug, title, topicSlug, templateCode, testCode, solutionCode string) *domain.Exercise {
	ex, err := domain.NewExercise(slug, title, topicSlug, templateCode, testCode, solutionCode)
	if err != nil {
		panic(err)
	}
	return ex
}

func mustExerciseRef(slug, title, difficulty string) *domain.ExerciseRef {
	ref, err := domain.NewExerciseRef(slug, title, difficulty)
	if err != nil {
		panic(err)
	}
	return ref
}
