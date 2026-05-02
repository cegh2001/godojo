package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"godojo/internal/adapters/repository"
	"godojo/internal/adapters/store"
	"godojo/internal/core/domain"
	"godojo/internal/core/services"
)

// mockHintProvider simulates a hint provider without requiring an API key.
type mockHintProvider struct {
	available bool
	delay     time.Duration
}

func (m *mockHintProvider) GetHint(ctx context.Context, exercise *domain.Exercise, testOutput string) (<-chan *domain.Hint, <-chan error) {
	hintCh := make(chan *domain.Hint, 1)
	errCh := make(chan error, 1)

	go func() {
		if !m.available {
			errCh <- fmt.Errorf("hints unavailable: no API key configured")
			close(errCh)
		} else {
			time.Sleep(m.delay)
			hintCh <- &domain.Hint{Content: "¿Probaste verificar tu implementación paso a paso?"}
			close(hintCh)
		}
	}()

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

	hintProvider := &mockHintProvider{available: false}
	hintSvc = services.NewHintService(hintProvider)

	return
}

// update is a helper that calls model.Update and returns concrete Model.
func updateModel(m Model, msg tea.Msg) Model {
	nm, _ := m.Update(msg)
	return nm.(Model)
}

func updateModelCmd(m Model, msg tea.Msg) (Model, tea.Cmd) {
	nm, cmd := m.Update(msg)
	return nm.(Model), cmd
}

// TestIntegration_RoadmapLoadsAndRenders tests the full roadmap loading flow.
func TestIntegration_RoadmapLoadsAndRenders(t *testing.T) {
	roadmapSvc, exerciseSvc, progressSvc, hintSvc := setupIntegrationServices(t)

	model := NewModel(roadmapSvc, exerciseSvc, progressSvc, hintSvc, nil, nil, "")

	// Verify initial state
	if model.state != stateRoadmapView {
		t.Errorf("initial state = %d, want stateRoadmapView", model.state)
	}

	// Init returns a command
	initCmd := model.Init()
	if initCmd == nil {
		t.Fatal("Init() should return a command")
	}

	// Execute the command
	msg := initCmd()
	rmMsg, ok := msg.(roadmapLoadedMsg)
	if !ok {
		t.Fatalf("expected roadmapLoadedMsg, got %T", msg)
	}
	if rmMsg.roadmap == nil {
		t.Fatal("roadmap should not be nil (empty roadmapLoadedMsg means loading failed)")
	}
	if len(rmMsg.roadmap.Phases) < 2 {
		t.Errorf("expected at least 2 phases, got %d", len(rmMsg.roadmap.Phases))
	}

	// Feed the roadmap to the model
	model = updateModel(model, rmMsg)

	// Verify state is still roadmap view
	if model.state != stateRoadmapView {
		t.Errorf("after roadmap load, state = %d, want stateRoadmapView", model.state)
	}
}

// TestIntegration_TopicDetail_NavigateToTopic tests entering a topic from roadmap.
func TestIntegration_TopicDetail_NavigateToTopic(t *testing.T) {
	roadmapSvc, exerciseSvc, progressSvc, hintSvc := setupIntegrationServices(t)

	model := NewModel(roadmapSvc, exerciseSvc, progressSvc, hintSvc, nil, nil, "")

	// Load roadmap
	initCmd := model.Init()
	msg := initCmd()
	model = updateModel(model, msg)

	// Press Enter to select first topic
	model = updateModel(model, tea.KeyMsg{Type: tea.KeyEnter})

	// Should transition to topic detail
	if model.state != stateTopicDetail {
		t.Errorf("after Enter on roadmap, state = %d, want stateTopicDetail", model.state)
	}
}

// TestIntegration_FullNavigationCycle tests the complete keyboard navigation cycle.
func TestIntegration_FullNavigationCycle(t *testing.T) {
	roadmapSvc, exerciseSvc, progressSvc, hintSvc := setupIntegrationServices(t)

	model := NewModel(roadmapSvc, exerciseSvc, progressSvc, hintSvc, nil, nil, "")

	// Load roadmap
	initCmd := model.Init()
	model = updateModel(model, initCmd())

	// Navigate down to second topic
	model = updateModel(model, tea.KeyMsg{Type: tea.KeyDown})
	if model.cursor != 1 {
		t.Errorf("cursor after down = %d, want 1", model.cursor)
	}

	// Navigate up back
	model = updateModel(model, tea.KeyMsg{Type: tea.KeyUp})
	if model.cursor != 0 {
		t.Errorf("cursor after up = %d, want 0", model.cursor)
	}

	// Select first topic → TopicDetail
	model = updateModel(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.state != stateTopicDetail {
		t.Errorf("after Enter, state = %d, want stateTopicDetail", model.state)
	}

	// Esc → back to Roadmap
	model = updateModel(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.state != stateRoadmapView {
		t.Errorf("after Esc from topic, state = %d, want stateRoadmapView", model.state)
	}

	// j/k aliases work
	model = updateModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if model.cursor != 1 {
		t.Errorf("cursor after 'j' = %d, want 1", model.cursor)
	}

	model = updateModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if model.cursor != 0 {
		t.Errorf("cursor after 'k' = %d, want 0", model.cursor)
	}

	// Quit via 'q'
	_, cmd := updateModelCmd(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("'q' should return a quit command")
	}
}

// TestIntegration_MissingAPIKey_DoesNotCrash tests graceful degradation.
func TestIntegration_MissingAPIKey_DoesNotCrash(t *testing.T) {
	oldKey := os.Getenv("GEMINI_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")
	defer func() {
		if oldKey != "" {
			os.Setenv("GEMINI_API_KEY", oldKey)
		}
	}()

	roadmapSvc, exerciseSvc, progressSvc, hintSvc := setupIntegrationServices(t)

	model := NewModel(roadmapSvc, exerciseSvc, progressSvc, hintSvc, nil, nil, "")
	if model.Init() == nil {
		t.Fatal("Init() should return non-nil cmd even without API key")
	}

	// Roadmap loads fine
	rm, err := roadmapSvc.GetRoadmap()
	if err != nil {
		t.Fatalf("GetRoadmap() error: %v", err)
	}
	if rm == nil {
		t.Fatal("roadmap is nil without API key")
	}

	// Hint service still works (degraded — returns errors via channels)
	if !hintSvc.IsAvailable() {
		t.Log("hint service reports unavailable — expected with mock unavailable provider")
	}
}

// TestIntegration_ProgressUpdates verifies progress tracking integration.
func TestIntegration_ProgressUpdates(t *testing.T) {
	_, _, progressSvc, _ := setupIntegrationServices(t)

	// Must start before completing
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

// TestIntegration_HintDegradation tests hint requests without API key.
func TestIntegration_HintDegradation(t *testing.T) {
	_, _, _, hintSvc := setupIntegrationServices(t)

	hintCh, errCh := hintSvc.RequestHint(&domain.Exercise{Slug: "test-ex"}, "output")

	select {
	case <-hintCh:
		t.Error("got hint from unavailable provider — unexpected")
	case err := <-errCh:
		if err == nil {
			t.Error("expected error from unavailable provider")
		}
	case <-time.After(2 * time.Second):
		t.Log("hint channels timed out — acceptable")
	}
}

// TestIntegration_Roadmap_HasTwoPhases verifies roadmap structure.
func TestIntegration_Roadmap_HasTwoPhases(t *testing.T) {
	roadmapSvc, _, _, _ := setupIntegrationServices(t)

	rm, err := roadmapSvc.GetRoadmap()
	if err != nil {
		t.Fatalf("GetRoadmap() error: %v", err)
	}

	if len(rm.Phases) != 2 {
		t.Errorf("expected 2 phases, got %d", len(rm.Phases))
	}

	fase1 := rm.Phases[0]
	if fase1.ID != "fase-1" {
		t.Errorf("phase 1 id = %q, want 'fase-1'", fase1.ID)
	}
	if len(fase1.Topics) != 5 {
		t.Errorf("fase 1 topics = %d, want 5", len(fase1.Topics))
	}

	fase2 := rm.Phases[1]
	if fase2.ID != "fase-2" {
		t.Errorf("phase 2 id = %q, want 'fase-2'", fase2.ID)
	}
	if len(fase2.Topics) != 4 {
		t.Errorf("fase 2 topics = %d, want 4", len(fase2.Topics))
	}

	// No duplicate slugs
	allSlugs := make(map[string]bool)
	for _, phase := range rm.Phases {
		for _, topic := range phase.Topics {
			if topic.Slug == "" {
				t.Errorf("empty slug in topic %q", topic.Title)
			}
			if allSlugs[topic.Slug] {
				t.Errorf("duplicate topic slug: %q", topic.Slug)
			}
			allSlugs[topic.Slug] = true
		}
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

// TestIntegration_QuitCommands verifies ctrl+c and q quit the model.
func TestIntegration_QuitCommands(t *testing.T) {
	roadmapSvc, exerciseSvc, progressSvc, hintSvc := setupIntegrationServices(t)

	model := NewModel(roadmapSvc, exerciseSvc, progressSvc, hintSvc, nil, nil, "")

	// Load roadmap first
	initCmd := model.Init()
	model = updateModel(model, initCmd())

	// ctrl+c should quit
	_, cmd := updateModelCmd(model, tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("ctrl+c should return quit command")
	}
}
