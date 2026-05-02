package tui

import (
	"strings"
	"testing"

	"godojo/internal/core/domain"
)

func TestViewRoadmap_ShowsTopics(t *testing.T) {
	m := newModelTest()
	m.state = stateRoadmapView
	m.topics = []*domain.Topic{
		{Slug: "t1", Title: "Variables"},
		{Slug: "t2", Title: "Funciones"},
	}
	m.cursor = 0

	view := m.View()
	if !strings.Contains(view, "GoDojo") {
		t.Error("roadmap view should contain 'GoDojo'")
	}
	if !strings.Contains(view, "Variables") {
		t.Error("roadmap view should contain 'Variables' topic")
	}
	if !strings.Contains(view, "Funciones") {
		t.Error("roadmap view should contain 'Funciones' topic")
	}
	if !strings.Contains(view, "j/k navegar") {
		t.Error("roadmap view should contain navigation help")
	}
}

func TestViewRoadmap_ShowsCursor(t *testing.T) {
	m := newModelTest()
	m.state = stateRoadmapView
	m.topics = []*domain.Topic{
		{Slug: "t1", Title: "Topic 1"},
		{Slug: "t2", Title: "Topic 2"},
	}
	m.cursor = 1 // second topic selected

	view := m.View()
	if !strings.Contains(view, "→") {
		t.Error("roadmap view should show cursor indicator '→'")
	}
}

func TestViewTopicDetail_ShowsTopicAndExercises(t *testing.T) {
	m := newModelTest()
	m.state = stateTopicDetail
	m.currentTopic = &domain.Topic{
		Slug:        "variables",
		Title:       "Variables",
		Description: "Aprendé sobre variables en Go",
	}
	m.exercises = []*domain.ExerciseRef{
		{Slug: "ex-1", Title: "Hola Mundo", Difficulty: domain.DifficultyEasy},
		{Slug: "ex-2", Title: "Tipos", Difficulty: domain.DifficultyMedium},
	}
	m.cursor = 0

	view := m.View()
	if !strings.Contains(view, "Variables") {
		t.Error("topic detail should show topic title")
	}
	if !strings.Contains(view, "Aprendé sobre variables en Go") {
		t.Error("topic detail should show topic description")
	}
	if !strings.Contains(view, "Hola Mundo") {
		t.Error("topic detail should list exercises")
	}
	if !strings.Contains(view, "Tipos") {
		t.Error("topic detail should list all exercises")
	}
}

func TestViewTopicDetail_ShowsDifficultyDots(t *testing.T) {
	m := newModelTest()
	m.state = stateTopicDetail
	m.currentTopic = &domain.Topic{Slug: "t", Title: "T"}
	m.exercises = []*domain.ExerciseRef{
		{Slug: "e1", Title: "Easy", Difficulty: domain.DifficultyEasy},
		{Slug: "e2", Title: "Medium", Difficulty: domain.DifficultyMedium},
		{Slug: "e3", Title: "Hard", Difficulty: domain.DifficultyHard},
	}
	m.cursor = 0

	view := m.View()
	if !strings.Contains(view, "●○○") {
		t.Error("should show '●○○' for easy difficulty")
	}
	if !strings.Contains(view, "●●○") {
		t.Error("should show '●●○' for medium difficulty")
	}
	if !strings.Contains(view, "●●●") {
		t.Error("should show '●●●' for hard difficulty")
	}
}

func TestViewTopicDetail_ShowsCursor(t *testing.T) {
	m := newModelTest()
	m.state = stateTopicDetail
	m.currentTopic = &domain.Topic{Slug: "t", Title: "T"}
	m.exercises = []*domain.ExerciseRef{
		{Slug: "e1", Title: "First", Difficulty: domain.DifficultyEasy},
	}
	m.cursor = 0

	view := m.View()
	if !strings.Contains(view, "→") {
		t.Error("topic detail should show cursor on selected exercise")
	}
}

func TestViewExercise_ShowsInstructions(t *testing.T) {
	m := newModelTest()
	m.state = stateExerciseView
	m.currentExercise = &domain.Exercise{
		Slug:      "hola-mundo",
		Title:     "Hola Mundo",
		TopicSlug: "variables",
	}

	view := m.View()
	if !strings.Contains(view, "Hola Mundo") {
		t.Error("exercise view should show exercise title")
	}
	if !strings.Contains(view, "ctrl+t") {
		t.Error("exercise view should show ctrl+t shortcut")
	}
	if !strings.Contains(view, "ctrl+h") {
		t.Error("exercise view should show ctrl+h shortcut")
	}
}

func TestViewExercise_NilExercise(t *testing.T) {
	m := newModelTest()
	m.state = stateExerciseView
	m.currentExercise = nil

	view := m.View()
	if !strings.Contains(view, "Cargando") {
		t.Error("nil exercise should show loading message")
	}
}

func TestViewTestRunning_ShowsSpinner(t *testing.T) {
	m := newModelTest()
	m.state = stateTestRunning

	view := m.View()
	if !strings.Contains(view, "Ejecutando tests") {
		t.Error("test running view should show 'Ejecutando tests'")
	}
}

func TestViewTestResults_Passed(t *testing.T) {
	m := newModelTest()
	m.state = stateTestResults
	m.testResult = &domain.TestResult{
		Passed: true,
		Output: "ok  godojo/exercises/variables/hola-mundo  0.123s",
	}

	view := m.View()
	if !strings.Contains(view, "Tests pasaron") {
		t.Error("passed tests should show success message")
	}
	if !strings.Contains(view, "ok  godojo") {
		t.Error("passed tests should show test output")
	}
}

func TestViewTestResults_Failed(t *testing.T) {
	m := newModelTest()
	m.state = stateTestResults
	m.testResult = &domain.TestResult{
		Passed: false,
		Output: "--- FAIL: TestSuma (0.00s)\n    ejercicio_test.go:10: expected 5, got 0",
	}

	view := m.View()
	if !strings.Contains(view, "Tests fallaron") {
		t.Error("failed tests should show failure message")
	}
	if !strings.Contains(view, "ctrl+h") {
		t.Error("failed tests should show hint shortcut")
	}
	if !strings.Contains(view, "--- FAIL") {
		t.Error("failed tests should show test output")
	}
}

func TestViewTestResults_NilResult(t *testing.T) {
	m := newModelTest()
	m.state = stateTestResults
	m.testResult = nil

	view := m.View()
	if !strings.Contains(view, "Sin resultados") {
		t.Error("nil test result should show 'Sin resultados'")
	}
}

func TestViewHintDisplay_ShowsHint(t *testing.T) {
	m := newModelTest()
	m.state = stateHintDisplay
	m.hintResult = &domain.Hint{
		Content: "¿Probaste usar un bucle for para iterar?",
	}

	view := m.View()
	if !strings.Contains(view, "Sensei dice") {
		t.Error("hint view should show 'Sensei dice'")
	}
	if !strings.Contains(view, "bucle for") {
		t.Error("hint view should show hint content")
	}
}

func TestViewHintDisplay_ShowsError(t *testing.T) {
	m := newModelTest()
	m.state = stateHintDisplay
	m.hintError = fmtError("timeout")

	view := m.View()
	if !strings.Contains(view, "no está disponible") {
		t.Error("hint error should show 'no está disponible'")
	}
}

func TestViewHintDisplay_Waiting(t *testing.T) {
	m := newModelTest()
	m.state = stateHintDisplay
	// Neither hintResult nor hintError set

	view := m.View()
	if !strings.Contains(view, "Consultando") {
		t.Error("waiting hint should show 'Consultando'")
	}
}

func TestDifficultyDots_AllLevels(t *testing.T) {
	tests := []struct {
		name       string
		difficulty domain.Difficulty
		want       string
	}{
		{"easy", domain.DifficultyEasy, "●○○"},
		{"medium", domain.DifficultyMedium, "●●○"},
		{"hard", domain.DifficultyHard, "●●●"},
		{"unknown", domain.Difficulty("unknown"), "○○○"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := difficultyDots(tt.difficulty)
			if !strings.Contains(got, tt.want) {
				t.Errorf("difficultyDots(%v) contains %q, want it to contain %q",
					tt.difficulty, got, tt.want)
			}
		})
	}
}

// fmtError creates a simple error for testing.
type fmtError string

func (e fmtError) Error() string { return string(e) }
