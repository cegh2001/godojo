package main

import (
	"os"
	"testing"
	"time"

	"godojo/internal/adapters/gemini"
	"godojo/internal/adapters/repository"
	"godojo/internal/adapters/runner"
	"godojo/internal/adapters/store"
	"godojo/internal/adapters/tui"
	"godojo/internal/core/services"
)

func TestWiring_NoPanic(t *testing.T) {
	// Test that the full wiring doesn't panic.
	// This verifies all constructors work and type assertions pass.

	// 1. Create adapters (real implementations)
	// Use temp dir for progress store
	tmpDir := t.TempDir()
	progressPath := tmpDir + "/progress.json"
	progressStore := store.NewJSONProgressStore(progressPath)
	testRunner := runner.NewGoTestRunner(30 * time.Second)
	exerciseRepo := repository.NewEmbedExerciseRepo()
	hintProvider := gemini.NewHintProvider()

	// 2. Create services (inject adapters)
	roadmapSvc := services.NewRoadmapService()
	exerciseSvc := services.NewExerciseService(exerciseRepo)
	progressSvc := services.NewProgressService(progressStore)
	hintSvc := services.NewHintService(hintProvider)

	// Verify services are created
	if roadmapSvc == nil {
		t.Fatal("roadmapSvc is nil")
	}
	if exerciseSvc == nil {
		t.Fatal("exerciseSvc is nil")
	}
	if progressSvc == nil {
		t.Fatal("progressSvc is nil")
	}
	if hintSvc == nil {
		t.Fatal("hintSvc is nil")
	}

	// 3. Create TUI model — should not panic
	model := tui.NewModel(roadmapSvc, exerciseSvc, progressSvc, hintSvc, testRunner, nil, t.TempDir())

	// 4. Verify model is initialized
	if model.Init() == nil {
		t.Fatal("model.Init() returned nil command")
	}
}

func TestWiring_MissingGeminiAPIKey_DoesNotCrash(t *testing.T) {
	// Remove GEMINI_API_KEY to simulate missing key
	oldKey := os.Getenv("GEMINI_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")
	defer func() {
		if oldKey != "" {
			os.Setenv("GEMINI_API_KEY", oldKey)
		}
	}()

	// Creating HintProvider without API key should NOT panic
	hintProvider := gemini.NewHintProvider()
	if hintProvider == nil {
		t.Fatal("hintProvider should not be nil even without API key")
	}

	// Creating HintService with the provider should NOT panic
	hintSvc := services.NewHintService(hintProvider)
	if hintSvc == nil {
		t.Fatal("hintSvc should not be nil")
	}

	// The full TUI model wiring should NOT panic
	tmpDir := t.TempDir()
	progressStore := store.NewJSONProgressStore(tmpDir + "/progress.json")
	testRunner := runner.NewGoTestRunner(30 * time.Second)
	exerciseRepo := repository.NewEmbedExerciseRepo()

	roadmapSvc := services.NewRoadmapService()
	exerciseSvc := services.NewExerciseService(exerciseRepo)
	progressSvc := services.NewProgressService(progressStore)

	model := tui.NewModel(roadmapSvc, exerciseSvc, progressSvc, hintSvc, testRunner, nil, t.TempDir())

	// Init should still work (hint degradation is expected)
	cmd := model.Init()
	if cmd == nil {
		t.Fatal("model.Init() returned nil command even without API key")
	}
}

func TestRoadmapLoading_Works(t *testing.T) {
	roadmapSvc := services.NewRoadmapService()
	rm, err := roadmapSvc.GetRoadmap()
	if err != nil {
		t.Fatalf("GetRoadmap() error: %v", err)
	}
	if rm == nil {
		t.Fatal("roadmap is nil")
	}
	if len(rm.Phases) < 2 {
		t.Errorf("expected at least 2 phases, got %d", len(rm.Phases))
	}
}

func TestExerciseRepo_HasExercises(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	// The repository should have exercises under real topic slugs matching RoadmapService
	refs, err := repo.ListByTopic(nil, "variables")
	if err != nil {
		t.Fatalf("ListByTopic error: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("expected at least one exercise in the repository")
	}
}
