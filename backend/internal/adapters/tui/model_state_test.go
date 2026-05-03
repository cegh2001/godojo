package tui

import (
	"testing"
)

// State and message handler tests for the simplified 3-state model.
// Old tests for roadmapLoadedMsg, topicSelectedMsg, exerciseSelectedMsg,
// testResultMsg, hintResultMsg, and hintErrorMsg have been removed as
// those message types were eliminated in the TUI model simplification.
// State transitions for RoadmapView, TopicDetail, ExerciseView, TestRunning,
// TestResults, and HintDisplay are no longer applicable.

func TestModel_View_DefaultState(t *testing.T) {
	m := newModelTest()
	m.state = 999 // invalid state

	view := m.View()
	if view != "Cargando..." {
		t.Errorf("default view = %q, want %q", view, "Cargando...")
	}
}

func TestModel_StateConstants(t *testing.T) {
	// Verify the 3 states are distinct and correctly ordered
	if stateSenseiChat != 0 {
		t.Errorf("stateSenseiChat = %d, want 0", stateSenseiChat)
	}
	if stateToolRunning != 1 {
		t.Errorf("stateToolRunning = %d, want 1", stateToolRunning)
	}
	if stateSessionSelector != 2 {
		t.Errorf("stateSessionSelector = %d, want 2", stateSessionSelector)
	}
}
