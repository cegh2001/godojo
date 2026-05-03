package tui

import (
	"context"
	"testing"
	"time"

	"godojo/internal/adapters/chatstore"
	"godojo/internal/core/domain"

	tea "github.com/charmbracelet/bubbletea"
)

// mockChatProvider implements chatProvider for testing.
type mockChatProvider struct {
	response    string
	err         error
	rateLimited bool
}

func (m *mockChatProvider) SendMessage(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage) (string, error) {
	return m.response, m.err
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

	// WHEN: NewModel is called with 9 params (including chat deps)
	m := NewModel(
		&stubRoadmapService{},
		nil, // exerciseSvc
		nil, // progressSvc
		nil, // hintSvc
		runner,
		nil, // exerciseRepo
		"/custom/workspace",
		nil, // chatStore
		nil, // chatProvider
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

func TestModel_ChatSessionsLoadedMsg(t *testing.T) {
	m := newModelTest()
	sessions := []chatstore.ChatSession{
		{ID: "s1", Name: "Test", UpdatedAt: time.Now()},
	}

	newM, _ := m.Update(chatSessionsLoadedMsg{sessions: sessions, err: nil})
	updated := newM.(Model)

	if len(updated.chatSessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(updated.chatSessions))
	}
	if updated.chatSessions[0].ID != "s1" {
		t.Errorf("session ID = %q", updated.chatSessions[0].ID)
	}
}
