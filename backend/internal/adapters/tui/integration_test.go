package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"godojo/internal/adapters/repository"
	"godojo/internal/adapters/store"
	"godojo/internal/core/domain"
	"godojo/internal/core/services"

	tea "github.com/charmbracelet/bubbletea"
)

// updateModel is a helper that calls model.Update and returns concrete Model.
func updateModel(m Model, msg tea.Msg) Model {
	nm, _ := m.Update(msg)
	return nm.(Model)
}

func updateModelCmd(m Model, msg tea.Msg) (Model, tea.Cmd) {
	nm, cmd := m.Update(msg)
	return nm.(Model), cmd
}

// TestIntegration_ModelCreation tests that the model can be created.
func TestIntegration_ModelCreation(t *testing.T) {
	model := NewModel(nil, nil, "")

	if model.state != stateSenseiChat {
		t.Errorf("initial state = %d, want stateSenseiChat", model.state)
	}

	initCmd := model.Init()
	if initCmd == nil {
		t.Fatal("Init() should return a command")
	}
}

// TestIntegration_QuitCommands verifies ctrl+c and q quit the model.
func TestIntegration_QuitCommands(t *testing.T) {
	model := NewModel(nil, nil, "")

	// ctrl+c should quit
	_, cmd := updateModelCmd(model, tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("ctrl+c should return quit command")
	}
}

// TestIntegration_ProgressUpdates verifies progress tracking integration.
func TestIntegration_ProgressUpdates(t *testing.T) {
	_, _, progressSvc, _ := setupIntegrationServices(t)

	err := progressSvc.MarkStarted("hola-mundo")
	if err != nil {
		t.Fatalf("MarkStarted() failed: %v", err)
	}

	err = progressSvc.MarkCompleted("hola-mundo")
	if err != nil {
		t.Fatalf("MarkCompleted() failed: %v", err)
	}

	prog, err := progressSvc.GetProgress("hola-mundo")
	if err != nil {
		t.Fatalf("GetProgress() failed: %v", err)
	}
	if prog == nil {
		t.Fatal("progress should not be nil")
	}
	if prog.Status != domain.ProgressCompleted {
		t.Errorf("status = %q, want %q", prog.Status, domain.ProgressCompleted)
	}

	all, err := progressSvc.GetAllProgress()
	if err != nil {
		t.Fatalf("GetAllProgress() failed: %v", err)
	}
	if len(all) < 1 {
		t.Error("expected at least 1 progress entry")
	}
}

// TestIntegration_ExerciseService_LoadsExercises tests ExerciseService with real repo.
func TestIntegration_ExerciseService_LoadsExercises(t *testing.T) {
	_, exerciseSvc, _, _ := setupIntegrationServices(t)

	topics := []string{"variables", "tipos", "funciones", "packages", "control-de-flujo"}
	for _, topic := range topics {
		refs, err := exerciseSvc.GetExercisesByTopic(topic)
		if err != nil {
			t.Errorf("GetExercisesByTopic(%q) error: %v", topic, err)
			continue
		}
		if len(refs) < 1 {
			t.Errorf("expected at least 1 exercise for topic %q, got %d", topic, len(refs))
		}
		for _, ref := range refs {
			if ref.Slug == "" {
				t.Errorf("exercise in topic %q has empty slug", topic)
			}
		}
	}
}

// TestIntegration_Roadmap_HasSevenPhases verifies roadmap structure.
func TestIntegration_Roadmap_HasSevenPhases(t *testing.T) {
	roadmapSvc, _, _, _ := setupIntegrationServices(t)

	rm, err := roadmapSvc.GetRoadmap()
	if err != nil {
		t.Fatalf("GetRoadmap() error: %v", err)
	}

	if len(rm.Phases) != 7 {
		t.Errorf("expected 7 phases, got %d", len(rm.Phases))
	}
}

// TestIntegration_AllTopicsHaveExercises verifies repo ↔ roadmap consistency.
func TestIntegration_AllTopicsHaveExercises(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	roadmapSvc := services.NewRoadmapService()

	rm, err := roadmapSvc.GetRoadmap()
	if err != nil {
		t.Fatalf("GetRoadmap() failed: %v", err)
	}

	for _, phase := range rm.Phases {
		for _, topic := range phase.Topics {
			refs, err := repo.ListByTopic(nil, topic.Slug)
			if err != nil {
				t.Errorf("topic %q: ListByTopic error: %v", topic.Slug, err)
				continue
			}
			if len(refs) == 0 {
				t.Errorf("topic %q: no exercises found", topic.Slug)
			}
		}
	}
}

// mockHintProviderIntegration implements ports.HintProvider for integration tests.
type mockHintProviderIntegration struct {
	available bool
}

func (m *mockHintProviderIntegration) GetHint(ctx context.Context, exercise *domain.Exercise, testOutput string) (<-chan *domain.Hint, <-chan error) {
	errCh := make(chan error, 1)
	hintCh := make(chan *domain.Hint, 1)

	if !m.available {
		errCh <- fmt.Errorf("hints unavailable")
	} else {
		hintCh <- &domain.Hint{Content: "pista de integración"}
	}
	close(hintCh)
	close(errCh)

	return hintCh, errCh
}

// setupIntegrationServices creates real services for integration testing.
func setupIntegrationServices(t *testing.T) (roadmapSvc *services.RoadmapService, exerciseSvc *services.ExerciseService, progressSvc *services.ProgressService, hintSvc *services.HintService) {
	t.Helper()

	exerciseRepo := repository.NewEmbedExerciseRepo()
	tmpDir := t.TempDir()
	progressStore := store.NewJSONProgressStore(filepath.Join(tmpDir, "progress.json"))

	roadmapSvc = services.NewRoadmapService()
	exerciseSvc = services.NewExerciseService(exerciseRepo)
	progressSvc = services.NewProgressService(progressStore)

	mockHintProvider := &mockHintProviderIntegration{available: false}
	hintSvc = services.NewHintService(mockHintProvider)

	return
}
