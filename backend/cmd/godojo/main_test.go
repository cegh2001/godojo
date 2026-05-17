package main

import (
	"testing"
	"time"

	"godojo/internal/adapters/genai"
	"godojo/internal/adapters/repository"
	"godojo/internal/adapters/runner"
	"godojo/internal/adapters/store"
	"godojo/internal/adapters/tui"
	"godojo/internal/adapters/workspace"
	"godojo/internal/core"
	"godojo/internal/core/ports"
	"godojo/internal/core/services"
)

func TestWiring_NoPanic(t *testing.T) {
	// Test that the full wiring doesn't panic.
	// This verifies all constructors work and type assertions pass.

	// 1. Create adapters (real implementations)
	tmpDir := t.TempDir()
	progressPath := tmpDir + "/progress.json"
	progressStore := store.NewJSONProgressStore(progressPath)
	exerciseRepo := repository.NewEmbedExerciseRepo()

	// 2. Create services (inject adapters)
	roadmapSvc := services.NewRoadmapService()
	exerciseSvc := services.NewExerciseService(exerciseRepo)
	progressSvc := services.NewProgressService(progressStore)

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

	// 3. Create SenseiService
	chatProviderInstance := genai.NewGenaiProvider()
	testRunner := runner.NewGoTestRunner(30 * time.Second)
	toolRegistry := core.NewToolRegistry()
	workspacePath := t.TempDir()
	workspaceManager := workspace.NewWorkspaceManager(workspacePath)
	senseiSvc := services.NewSenseiService(chatProviderInstance, toolRegistry, workspaceManager, roadmapSvc, testRunner)

	// 4. Create TUI model — should not panic
	model := tui.NewModel(senseiSvc, nil, "")

	// 5. Verify model is initialized
	if model.Init() == nil {
		t.Fatal("model.Init() returned nil command")
	}
}

func TestWiring_UsesGenaiProvider(t *testing.T) {
	// Verifies that NewGenaiProvider() returns a valid SenseiProvider implementation.
	provider := genai.NewGenaiProvider()

	// Type assertion: provider should implement ports.SenseiProvider
	senseiProvider, ok := interface{}(provider).(ports.SenseiProvider)
	if !ok {
		t.Fatal("NewGenaiProvider() does not implement ports.SenseiProvider")
	}

	// Structural check: provider must not be nil
	if senseiProvider == nil {
		t.Fatal("SenseiProvider is nil")
	}
}

func TestWiring_UsesGoTestRunner(t *testing.T) {
	// Verifies that a real GoTestRunner is wired in and implements ports.TestRunner.
	testRunner := runner.NewGoTestRunner(30 * time.Second)

	testRunnerInterface, ok := interface{}(testRunner).(ports.TestRunner)
	if !ok {
		t.Fatal("NewGoTestRunner() does not implement ports.TestRunner")
	}
	if testRunnerInterface == nil {
		t.Fatal("TestRunner is nil")
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
	refs, err := repo.ListByTopic(nil, "variables")
	if err != nil {
		t.Fatalf("ListByTopic error: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("expected at least one exercise in the repository")
	}
}
