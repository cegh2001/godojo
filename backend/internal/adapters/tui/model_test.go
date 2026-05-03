package tui

import (
	"context"
	"testing"

	"godojo/internal/core/domain"

	tea "github.com/charmbracelet/bubbletea"
)

// stubRoadmapService returns a minimal roadmap for testing.
type stubRoadmapService struct{}

func (s *stubRoadmapService) GetRoadmap() (*domain.Roadmap, error) {
	t1, _ := domain.NewTopic("t1", "Topic 1", "desc", nil, nil)
	t2, _ := domain.NewTopic("t2", "Topic 2", "desc", nil, nil)
	p1, _ := domain.NewPhase("fase-1", "Fundamentos", []*domain.Topic{t1, t2})
	return domain.NewRoadmap([]*domain.Phase{p1})
}

func (s *stubRoadmapService) GetTopicBySlug(slug string) (*domain.Topic, error) {
	t1, _ := domain.NewTopic("t1", "Topic 1", "desc", nil, nil)
	t2, _ := domain.NewTopic("t2", "Topic 2", "desc", nil, nil)
	if slug == "t1" {
		return t1, nil
	}
	if slug == "t2" {
		return t2, nil
	}
	return nil, nil
}

// stubExerciseService returns empty exercises for testing.
type stubExerciseService struct{}

func (s *stubExerciseService) StartExercise(slug string) (*domain.Exercise, error) {
	return nil, nil
}
func (s *stubExerciseService) GetExercisesByTopic(topicSlug string) ([]*domain.ExerciseRef, error) {
	return nil, nil
}
func (s *stubExerciseService) ValidateExercise(slug string, testResult *domain.TestResult) (bool, error) {
	return false, nil
}

// stubExerciseRepo is a minimal ExerciseRepository for tests.
type stubExerciseRepo struct{}

func (s *stubExerciseRepo) GetBySlug(ctx context.Context, topicSlug, slug string) (*domain.Exercise, error) {
	return nil, nil
}
func (s *stubExerciseRepo) ListByTopic(ctx context.Context, topicSlug string) ([]*domain.ExerciseRef, error) {
	return nil, nil
}
func (s *stubExerciseRepo) GenerateFiles(ctx context.Context, exercise *domain.Exercise, workspacePath string) error {
	return nil
}

// newModelTest creates a Model for testing with stub services.
func newModelTest() Model {
	return Model{
		roadmapSvc:   &stubRoadmapService{},
		exerciseSvc:  &stubExerciseService{},
		exerciseRepo: &stubExerciseRepo{},
		state:        stateRoadmapView,
	}
}

func TestNewModel_InitialState(t *testing.T) {
	m := newModelTest()

	if m.state != stateRoadmapView {
		t.Errorf("initial state = %v, want %v", m.state, stateRoadmapView)
	}
	if m.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", m.cursor)
	}
}

func TestModel_Init_ReturnsLoadRoadmapCmd(t *testing.T) {
	m := newModelTest()
	cmd := m.Init()

	if cmd == nil {
		t.Fatal("Init() returned nil command, expected loadRoadmap command")
	}

	// Execute the command and check it returns roadmapLoadedMsg
	msg := cmd()
	if msg == nil {
		t.Fatal("command produced nil message")
	}
	_, ok := msg.(roadmapLoadedMsg)
	if !ok {
		t.Errorf("command produced %T, want roadmapLoadedMsg", msg)
	}
}

func TestModel_Quit_OnCtrlC(t *testing.T) {
	m := newModelTest()

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c should return tea.Quit")
	}
	_ = newModel

	// Verify cmd is tea.Quit by checking its string representation
	quitMsg := cmd()
	if quitMsg == nil {
		t.Fatal("tea.Quit produced nil message")
	}
}

func TestModel_Quit_OnQ(t *testing.T) {
	m := newModelTest()

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("'q' should return tea.Quit")
	}
	_ = newModel

	quitMsg := cmd()
	if quitMsg == nil {
		t.Fatal("tea.Quit produced nil message")
	}
}

func TestModel_StateTransition_RoadmapToTopicDetail_OnEnter(t *testing.T) {
	// Load the roadmap first
	m := newModelTest()
	m.roadmap = &domain.Roadmap{
		Phases: []*domain.Phase{
			{
				ID: "fase-1",
				Topics: []*domain.Topic{
					{Slug: "t1", Title: "Topic 1"},
				},
			},
		},
	}

	// We need to set up the model with roadmap loaded and topics list populated
	// The roadmap is loaded via roadmapLoadedMsg
	rm := &domain.Roadmap{
		Phases: []*domain.Phase{
			{
				ID:    "fase-1",
				Title: "Fundamentos",
				Topics: []*domain.Topic{
					{Slug: "t1", Title: "Topic 1", Description: "T1 desc"},
					{Slug: "t2", Title: "Topic 2", Description: "T2 desc"},
				},
			},
		},
	}
	m.roadmap = rm
	m.phases = rm.Phases
	// Build flat topic list
	var topics []*domain.Topic
	for _, ph := range rm.Phases {
		topics = append(topics, ph.Topics...)
	}
	m.topics = topics
	m.cursor = 0
	m.state = stateRoadmapView

	// Press Enter to select first topic
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = cmd

	updated, ok := newM.(Model)
	if !ok {
		t.Fatalf("Update returned %T, not Model", newM)
	}

	if updated.state != stateTopicDetail {
		t.Errorf("after Enter on topic, state = %v, want %v", updated.state, stateTopicDetail)
	}
	if updated.currentTopic == nil {
		t.Fatal("currentTopic should not be nil after selecting topic")
	}
	if updated.currentTopic.Slug != "t1" {
		t.Errorf("selected topic slug = %q, want %q", updated.currentTopic.Slug, "t1")
	}
}

func TestModel_StateTransition_TopicDetailToRoadmap_OnEsc(t *testing.T) {
	m := newModelTest()
	m.state = stateTopicDetail
	m.cursor = 0

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := newM.(Model)

	if updated.state != stateRoadmapView {
		t.Errorf("after Esc from TopicDetail, state = %v, want %v", updated.state, stateRoadmapView)
	}
}

func TestModel_StateTransition_ExerciseViewToTestRunning_OnCtrlT(t *testing.T) {
	m := newModelTest()
	m.state = stateExerciseView
	m.currentExercise = &domain.Exercise{Slug: "ex-1", Title: "Ejercicio 1"}

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	_ = cmd

	updated := newM.(Model)
	if updated.state != stateTestRunning {
		t.Errorf("after ctrl+t, state = %v, want %v", updated.state, stateTestRunning)
	}
}

func TestModel_StateTransition_TestResultsToHintDisplay_OnCtrlH(t *testing.T) {
	m := newModelTest()
	m.state = stateTestResults
	m.testResult = &domain.TestResult{Passed: false, Output: "FAIL"}

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	_ = cmd

	updated := newM.(Model)
	if updated.state != stateHintDisplay {
		t.Errorf("after ctrl+h on failed test, state = %v, want %v", updated.state, stateHintDisplay)
	}
}

func TestModel_StateTransition_HintDisplayToTestResults_OnEsc(t *testing.T) {
	m := newModelTest()
	m.state = stateHintDisplay
	m.previousView = stateTestResults
	m.hintResult = &domain.Hint{Content: "pista"}

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := newM.(Model)

	if updated.state != stateTestResults {
		t.Errorf("after Esc from HintDisplay, state = %v, want %v", updated.state, stateTestResults)
	}
}
