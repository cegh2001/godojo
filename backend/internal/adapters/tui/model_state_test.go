package tui

import (
	"fmt"
	"testing"

	"godojo/internal/core/domain"

	tea "github.com/charmbracelet/bubbletea"
)

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
	m.state = stateTestRunning
	m.hintRequested = true

	h := &domain.Hint{Content: "Probaste usando un loop?"}
	newM, _ := m.Update(hintResultMsg{hint: h})
	updated := newM.(Model)

	if updated.hintResult == nil {
		t.Fatal("hintResult should be set")
	}
	if updated.hintResult.Content != "Probaste usando un loop?" {
		t.Errorf("hint content = %q, want %q", updated.hintResult.Content, "Probaste usando un loop?")
	}
	if updated.hintRequested {
		t.Error("hintRequested should be cleared after receiving result")
	}
}

func TestModel_HandleHintErrorMsg(t *testing.T) {
	m := newModelTest()
	m.state = stateTestRunning
	m.hintRequested = true

	newM, _ := m.Update(hintErrorMsg{err: fmt.Errorf("timeout")})
	updated := newM.(Model)

	if updated.hintError == nil {
		t.Error("hintError should be set")
	}
	if updated.hintRequested {
		t.Error("hintRequested should be cleared after receiving error")
	}
}

func TestModel_CursorNavigation_Down(t *testing.T) {
	m := newModelTest()
	m.state = stateRoadmapView
	m.topics = []*domain.Topic{
		{Slug: "t1"}, {Slug: "t2"}, {Slug: "t3"},
	}
	m.cursor = 0

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := newM.(Model)
	if updated.cursor != 1 {
		t.Errorf("cursor after one down = %d, want 1", updated.cursor)
	}

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
	m.cursor = 1

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
	m.cursor = 0

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

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := newM.(Model)
	if updated.cursor != 1 {
		t.Errorf("cursor after 'j' = %d, want 1", updated.cursor)
	}

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
	m := newModelTest()
	m.state = stateTestResults
	m.testResult = &domain.TestResult{Passed: true, Output: "PASS"}

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	updated := newM.(Model)

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

func TestHandleCtrlT_ReturnsCommand_WhenTestRunnerSet(t *testing.T) {
	runner := &mockTestRunner{
		result: &domain.TestResult{Passed: true, Output: "PASS"},
	}
	m := newModelTest()
	m.testRunner = runner
	m.workspacePath = "/tmp/workspace"
	m.state = stateExerciseView
	m.currentExercise = &domain.Exercise{Slug: "ex-1", Title: "Test Exercise"}

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})

	updated := newM.(Model)
	if updated.state != stateTestRunning {
		t.Errorf("state after ctrl+t = %v, want %v", updated.state, stateTestRunning)
	}

	if cmd == nil {
		t.Fatal("ctrl+t should return a non-nil command when testRunner is set")
	}
}

func TestHandleCtrlT_ReturnsNilCommand_WhenTestRunnerNil(t *testing.T) {
	m := newModelTest()
	m.state = stateExerciseView
	m.currentExercise = &domain.Exercise{Slug: "ex-1", Title: "Test Exercise"}

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})

	updated := newM.(Model)
	if updated.state != stateTestRunning {
		t.Errorf("state after ctrl+t = %v, want %v", updated.state, stateTestRunning)
	}

	if cmd != nil {
		t.Errorf("ctrl+t should return nil command when testRunner is nil, got non-nil")
	}
}

func TestHandleCtrlH_ReturnsCommand_WhenHintAvailable(t *testing.T) {
	hintSvc := &mockHintService{
		hint: &domain.Hint{Content: "pista útil"},
	}
	m := newModelTest()
	m.hintSvc = hintSvc
	m.state = stateTestResults
	m.testResult = &domain.TestResult{Passed: false, Output: "FAIL: expected X got Y"}
	m.currentExercise = &domain.Exercise{Slug: "ex-1"}

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})

	updated := newM.(Model)
	if updated.state != stateHintDisplay {
		t.Errorf("state after ctrl+h = %v, want %v", updated.state, stateHintDisplay)
	}

	if cmd == nil {
		t.Fatal("ctrl+h should return a non-nil command when hintSvc is set")
	}
}

func TestHandleCtrlH_ReturnsNil_WhenTestPassed(t *testing.T) {
	hintSvc := &mockHintService{
		hint: &domain.Hint{Content: "pista"},
	}
	m := newModelTest()
	m.hintSvc = hintSvc
	m.state = stateTestResults
	m.testResult = &domain.TestResult{Passed: true, Output: "PASS"}
	m.currentExercise = &domain.Exercise{Slug: "ex-1"}

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})

	updated := newM.(Model)
	if updated.state == stateHintDisplay {
		t.Error("should not transition to hint display when test passed")
	}

	if cmd != nil {
		t.Errorf("ctrl+h should return nil command when test passed, got non-nil")
	}

	_ = updated
}
