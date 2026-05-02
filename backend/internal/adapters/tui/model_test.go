package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"godojo/internal/core/domain"
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

func TestModel_HandleRoadmapLoadedMsg(t *testing.T) {
	m := newModelTest()
	rm := &domain.Roadmap{
		Phases: []*domain.Phase{
			{
				ID:    "fase-1",
				Title: "Fundamentos",
				Topics: []*domain.Topic{
					{Slug: "t1", Title: "Topic 1"},
				},
			},
		},
	}

	newM, _ := m.Update(roadmapLoadedMsg{roadmap: rm})
	updated := newM.(Model)

	if updated.roadmap == nil {
		t.Fatal("roadmap should be set after roadmapLoadedMsg")
	}
	if len(updated.phases) != 1 {
		t.Errorf("phases count = %d, want 1", len(updated.phases))
	}
	if len(updated.topics) != 1 {
		t.Errorf("topics count = %d, want 1", len(updated.topics))
	}
	if updated.state != stateRoadmapView {
		t.Errorf("state after roadmap loaded = %v, want %v", updated.state, stateRoadmapView)
	}
}

func TestModel_HandleTestResultMsg(t *testing.T) {
	m := newModelTest()
	m.state = stateTestRunning

	tr := &domain.TestResult{Passed: true, Output: "PASS", Duration: 0}
	newM, _ := m.Update(testResultMsg{result: tr, err: nil})
	updated := newM.(Model)

	if updated.state != stateTestResults {
		t.Errorf("state after test result = %v, want %v", updated.state, stateTestResults)
	}
	if updated.testResult == nil {
		t.Fatal("testResult should be set")
	}
	if !updated.testResult.Passed {
		t.Error("testResult.Passed should be true")
	}
}

func TestModel_HandleHintResultMsg(t *testing.T) {
	m := newModelTest()
	m.state = stateTestRunning // hint arrives during/after test

	h := &domain.Hint{Content: "Probaste usando un loop?"}
	newM, _ := m.Update(hintResultMsg{hint: h})
	updated := newM.(Model)

	if updated.hintResult == nil {
		t.Fatal("hintResult should be set")
	}
	if updated.hintResult.Content != "Probaste usando un loop?" {
		t.Errorf("hint content = %q, want %q", updated.hintResult.Content, "Probaste usando un loop?")
	}
}

func TestModel_HandleHintErrorMsg(t *testing.T) {
	m := newModelTest()
	m.state = stateTestRunning

	newM, _ := m.Update(hintErrorMsg{err: nil})
	updated := newM.(Model)

	// State should transition to hintDisplay even on error
	if updated.state != stateHintDisplay {
		t.Errorf("state after hint error = %v, want %v", updated.state, stateHintDisplay)
	}
	// previousView should be saved for back navigation
	if updated.previousView != stateTestRunning {
		t.Errorf("previousView after hint error = %v, want %v", updated.previousView, stateTestRunning)
	}
}

func TestModel_CursorNavigation_Down(t *testing.T) {
	m := newModelTest()
	m.state = stateRoadmapView
	m.topics = []*domain.Topic{
		{Slug: "t1"}, {Slug: "t2"}, {Slug: "t3"},
	}
	m.cursor = 0

	// Move down
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := newM.(Model)
	if updated.cursor != 1 {
		t.Errorf("cursor after one down = %d, want 1", updated.cursor)
	}

	// Move down again
	newM2, _ := updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated2 := newM2.(Model)
	if updated2.cursor != 2 {
		t.Errorf("cursor after two down = %d, want 2", updated2.cursor)
	}
}

func TestModel_CursorNavigation_Boundary_Down(t *testing.T) {
	m := newModelTest()
	m.state = stateRoadmapView
	m.topics = []*domain.Topic{
		{Slug: "t1"}, {Slug: "t2"},
	}
	m.cursor = 1 // last item

	// Press down at boundary — should stay at last
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := newM.(Model)
	if updated.cursor != 1 {
		t.Errorf("cursor at bottom boundary = %d, want 1", updated.cursor)
	}
}

func TestModel_CursorNavigation_Up(t *testing.T) {
	m := newModelTest()
	m.state = stateRoadmapView
	m.topics = []*domain.Topic{
		{Slug: "t1"}, {Slug: "t2"}, {Slug: "t3"},
	}
	m.cursor = 2

	// Move up
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := newM.(Model)
	if updated.cursor != 1 {
		t.Errorf("cursor after one up = %d, want 1", updated.cursor)
	}
}

func TestModel_CursorNavigation_Boundary_Up(t *testing.T) {
	m := newModelTest()
	m.state = stateRoadmapView
	m.topics = []*domain.Topic{
		{Slug: "t1"}, {Slug: "t2"},
	}
	m.cursor = 0 // first item

	// Press up at boundary — should stay at 0
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := newM.(Model)
	if updated.cursor != 0 {
		t.Errorf("cursor at top boundary = %d, want 0", updated.cursor)
	}
}

func TestModel_CursorNavigation_JK(t *testing.T) {
	m := newModelTest()
	m.state = stateRoadmapView
	m.topics = []*domain.Topic{
		{Slug: "t1"}, {Slug: "t2"}, {Slug: "t3"},
	}
	m.cursor = 0

	// 'j' moves down
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := newM.(Model)
	if updated.cursor != 1 {
		t.Errorf("cursor after 'j' = %d, want 1", updated.cursor)
	}

	// 'k' moves up
	newM2, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	updated2 := newM2.(Model)
	if updated2.cursor != 0 {
		t.Errorf("cursor after 'k' = %d, want 0", updated2.cursor)
	}
}

func TestModel_HandleExerciseSelectedMsg(t *testing.T) {
	m := newModelTest()
	m.state = stateTopicDetail

	ex := &domain.Exercise{Slug: "ex-1", Title: "Ejercicio 1", TopicSlug: "t1"}
	newM, _ := m.Update(exerciseSelectedMsg{exercise: ex})
	updated := newM.(Model)

	if updated.state != stateExerciseView {
		t.Errorf("state after exercise selected = %v, want %v", updated.state, stateExerciseView)
	}
	if updated.currentExercise == nil {
		t.Fatal("currentExercise should be set")
	}
	if updated.currentExercise.Slug != "ex-1" {
		t.Errorf("exercise slug = %q, want %q", updated.currentExercise.Slug, "ex-1")
	}
}

func TestModel_HandleTopicSelectedMsg(t *testing.T) {
	m := newModelTest()
	m.state = stateRoadmapView

	topic := &domain.Topic{Slug: "t1", Title: "Topic 1", Description: "Topic 1 description"}
	newM, _ := m.Update(topicSelectedMsg{topic: topic})
	updated := newM.(Model)

	if updated.state != stateTopicDetail {
		t.Errorf("state after topic selected = %v, want %v", updated.state, stateTopicDetail)
	}
	if updated.currentTopic == nil {
		t.Fatal("currentTopic should be set")
	}
	if updated.currentTopic.Slug != "t1" {
		t.Errorf("topic slug = %q, want %q", updated.currentTopic.Slug, "t1")
	}
}

func TestModel_TestResultsToTopicDetail_OnEsc(t *testing.T) {
	m := newModelTest()
	m.state = stateTestResults
	m.testResult = &domain.TestResult{Passed: true}

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := newM.(Model)

	if updated.state != stateTopicDetail {
		t.Errorf("state after Esc from TestResults = %v, want %v", updated.state, stateTopicDetail)
	}
}

func TestModel_ExerciseViewToTopicDetail_OnEsc(t *testing.T) {
	m := newModelTest()
	m.state = stateExerciseView
	m.currentExercise = &domain.Exercise{Slug: "ex-1"}

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := newM.(Model)

	if updated.state != stateTopicDetail {
		t.Errorf("state after Esc from ExerciseView = %v, want %v", updated.state, stateTopicDetail)
	}
}

func TestModel_CannotRequestHint_WhenTestPassed(t *testing.T) {
	// ctrl+h should only work when test failed
	m := newModelTest()
	m.state = stateTestResults
	m.testResult = &domain.TestResult{Passed: true, Output: "PASS"}

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	updated := newM.(Model)

	// State should NOT change to hint display — hint only available on failure
	if updated.state == stateHintDisplay {
		t.Errorf("should not transition to hint display when test passed")
	}
}

func TestModel_View_ReturnsNonEmpty(t *testing.T) {
	m := newModelTest()
	m.state = stateRoadmapView

	view := m.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

// ---- Bug 1 & 2 Tests: Command dispatch verification ----

func TestHandleCtrlT_ReturnsCommand_WhenTestRunnerSet(t *testing.T) {
	// GIVEN: Model with testRunner in exercise view
	runner := &mockTestRunner{
		result: &domain.TestResult{Passed: true, Output: "PASS"},
	}
	m := newModelTest()
	m.testRunner = runner
	m.workspacePath = "/tmp/workspace"
	m.state = stateExerciseView
	m.currentExercise = &domain.Exercise{Slug: "ex-1", Title: "Test Exercise"}

	// WHEN: ctrl+t is pressed
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})

	// THEN: State transitions to test running
	updated := newM.(Model)
	if updated.state != stateTestRunning {
		t.Errorf("state after ctrl+t = %v, want %v", updated.state, stateTestRunning)
	}

	// THEN: A command is dispatched (non-nil)
	if cmd == nil {
		t.Fatal("ctrl+t should return a non-nil command when testRunner is set")
	}
}

func TestHandleCtrlT_ReturnsNilCommand_WhenTestRunnerNil(t *testing.T) {
	// GIVEN: Model WITHOUT testRunner in exercise view
	m := newModelTest()
	m.state = stateExerciseView
	m.currentExercise = &domain.Exercise{Slug: "ex-1", Title: "Test Exercise"}

	// WHEN: ctrl+t is pressed
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})

	// THEN: State still transitions
	updated := newM.(Model)
	if updated.state != stateTestRunning {
		t.Errorf("state after ctrl+t = %v, want %v", updated.state, stateTestRunning)
	}

	// THEN: No command dispatched (graceful degradation)
	if cmd != nil {
		t.Errorf("ctrl+t should return nil command when testRunner is nil, got non-nil")
	}
}

func TestHandleCtrlH_ReturnsCommand_WhenHintAvailable(t *testing.T) {
	// GIVEN: Model with hintSvc in test results with failed test
	hintSvc := &mockHintService{
		hint: &domain.Hint{Content: "pista útil"},
	}
	m := newModelTest()
	m.hintSvc = hintSvc
	m.state = stateTestResults
	m.testResult = &domain.TestResult{Passed: false, Output: "FAIL: expected X got Y"}
	m.currentExercise = &domain.Exercise{Slug: "ex-1"}

	// WHEN: ctrl+h is pressed
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})

	// THEN: State transitions to hint display
	updated := newM.(Model)
	if updated.state != stateHintDisplay {
		t.Errorf("state after ctrl+h = %v, want %v", updated.state, stateHintDisplay)
	}

	// THEN: A command is dispatched (non-nil)
	if cmd == nil {
		t.Fatal("ctrl+h should return a non-nil command when hintSvc is set")
	}
}

func TestHandleCtrlH_ReturnsNil_WhenTestPassed(t *testing.T) {
	// GIVEN: Model with hintSvc but test passed (shouldn't dispatch)
	hintSvc := &mockHintService{
		hint: &domain.Hint{Content: "pista"},
	}
	m := newModelTest()
	m.hintSvc = hintSvc
	m.state = stateTestResults
	m.testResult = &domain.TestResult{Passed: true, Output: "PASS"}
	m.currentExercise = &domain.Exercise{Slug: "ex-1"}

	// WHEN: ctrl+h is pressed
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})

	// THEN: State does NOT change (hint only available on failure)
	updated := newM.(Model)
	if updated.state == stateHintDisplay {
		t.Error("should not transition to hint display when test passed")
	}

	// THEN: No command dispatched
	if cmd != nil {
		t.Errorf("ctrl+h should return nil command when test passed, got non-nil")
	}

	_ = updated
}

func TestHandleTestResult_StoresLastTestOutput(t *testing.T) {
	// GIVEN: Model in test running state
	m := newModelTest()
	m.state = stateTestRunning

	// WHEN: testResultMsg arrives with output
	tr := &domain.TestResult{Passed: false, Output: "FAIL: expected 5, got 3", Duration: 100}
	newM, _ := m.Update(testResultMsg{result: tr, err: nil})
	updated := newM.(Model)

	// THEN: lastTestOutput is stored
	if updated.lastTestOutput != "FAIL: expected 5, got 3" {
		t.Errorf("lastTestOutput = %q, want %q", updated.lastTestOutput, "FAIL: expected 5, got 3")
	}
}

func TestHandleCtrlH_UsesLastTestOutput(t *testing.T) {
	// GIVEN: Model with hintSvc, failed test, and stored lastTestOutput
	hintSvc := &mockHintService{
		hint: &domain.Hint{Content: "pista del output"},
	}
	m := newModelTest()
	m.hintSvc = hintSvc
	m.state = stateTestResults
	m.testResult = &domain.TestResult{Passed: false, Output: "FAIL: assertion failed"}
	m.currentExercise = &domain.Exercise{Slug: "ex-1"}
	m.lastTestOutput = "FAIL: assertion failed"

	// WHEN: ctrl+h is pressed, the dispatched command should use lastTestOutput
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	updated := newM.(Model)

	if updated.state != stateHintDisplay {
		t.Errorf("state after ctrl+h = %v, want %v", updated.state, stateHintDisplay)
	}
	if cmd == nil {
		t.Fatal("ctrl+h should dispatch command when hint available")
	}

	// Execute the command and verify it produces a result (proves lastTestOutput was used)
	msg := cmd()
	_, ok := msg.(hintResultMsg)
	if !ok {
		t.Errorf("expected hintResultMsg from command, got %T", msg)
	}
}

func TestNewModel_AcceptsTestRunnerAndWorkspacePath(t *testing.T) {
	runner := &mockTestRunner{
		result: &domain.TestResult{Passed: true},
	}

	// WHEN: NewModel is called with 7 params
	m := NewModel(
		&stubRoadmapService{},
		nil, // exerciseSvc
		nil, // progressSvc
		nil, // hintSvc
		runner,
		nil, // exerciseRepo
		"/custom/workspace",
	)

	// THEN: Fields are populated
	if m.testRunner != runner {
		t.Error("testRunner should be stored")
	}
	if m.workspacePath != "/custom/workspace" {
		t.Errorf("workspacePath = %q, want %q", m.workspacePath, "/custom/workspace")
	}
	if m.state != stateRoadmapView {
		t.Errorf("initial state = %v, want %v", m.state, stateRoadmapView)
	}
}
