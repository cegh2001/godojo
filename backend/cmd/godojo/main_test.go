package main

import (
	"testing"

	"godojo/internal/adapters/gemini"
	"godojo/internal/adapters/repository"
	"godojo/internal/adapters/store"
	"godojo/internal/adapters/tui"
	"godojo/internal/adapters/workspace"
	"godojo/internal/core"
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
	chatProviderInstance := gemini.NewChatProvider()
	toolRegistry := core.NewToolRegistry()
	workspacePath := t.TempDir()
	workspaceManager := workspace.NewWorkspaceManager(workspacePath)
	senseiSvc := services.NewSenseiService(chatProviderInstance, toolRegistry, workspaceManager, roadmapSvc)

	// 4. Create TUI model — should not panic
	model := tui.NewModel(senseiSvc, nil, "")

	// 5. Verify model is initialized
	if model.Init() == nil {
		t.Fatal("model.Init() returned nil command")
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
