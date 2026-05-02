package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"godojo/internal/core/domain"
	"godojo/internal/core/ports"
)

// runTestsCmd returns a Bubbletea command that executes tests and sends back results.
// It accepts a TestRunner interface to actually run the tests.
func runTestsCmd(runner ports.TestRunner, workspace string) tea.Cmd {
	return func() tea.Msg {
		// Use background context with timeout
		result, err := runner.Run(nil, workspace)
		if err != nil {
			return testResultMsg{
				result: &domain.TestResult{
					Passed: false,
					Output: fmt.Sprintf("Error al ejecutar tests: %v", err),
				},
				err: err,
			}
		}
		return testResultMsg{result: result, err: nil}
	}
}

// fetchHintCmd returns a Bubbletea command that requests a hint asynchronously.
// It reads from the channels returned by HintService.RequestHint.
func fetchHintCmd(hintSvc hintService, exercise *domain.Exercise, testOutput string) tea.Cmd {
	return func() tea.Msg {
		if hintSvc == nil || exercise == nil {
			return hintErrorMsg{err: fmt.Errorf("servicio de pistas no disponible")}
		}

		hintCh, errCh := hintSvc.RequestHint(exercise, testOutput)
		if hintCh == nil && errCh == nil {
			// Rate limited or unavailable
			return hintErrorMsg{err: fmt.Errorf("pistas no disponibles en este momento (límite alcanzado)")}
		}

		// Wait for either hint or error
		select {
		case hint, ok := <-hintCh:
			if ok && hint != nil {
				return hintResultMsg{hint: hint}
			}
			return hintErrorMsg{err: fmt.Errorf("el sensei no respondió")}
		case err, ok := <-errCh:
			if ok && err != nil {
				return hintErrorMsg{err: err}
			}
			return hintErrorMsg{err: fmt.Errorf("error desconocido al consultar al sensei")}
		case <-time.After(30 * time.Second):
			return hintErrorMsg{err: fmt.Errorf("timeout: el sensei no respondió a tiempo")}
		}
	}
}

// loadProgressCmd returns a command that loads all progress from the ProgressService.
func loadProgressCmd(progressSvc progressService) tea.Cmd {
	return func() tea.Msg {
		progress, err := progressSvc.GetAllProgress()
		if err != nil {
			return progressLoadedMsg{progress: nil, err: err}
		}
		return progressLoadedMsg{progress: progress, err: nil}
	}
}

// loadTopicExercisesCmd returns a command that loads exercises for a given topic.
func loadTopicExercisesCmd(exerciseSvc exerciseService, topicSlug string) tea.Cmd {
	return func() tea.Msg {
		exercises, err := exerciseSvc.GetExercisesByTopic(topicSlug)
		if err != nil {
			return topicExercisesLoadedMsg{exercises: nil, err: err}
		}
		return topicExercisesLoadedMsg{exercises: exercises, err: nil}
	}
}

// progressLoadedMsg is sent when progress is loaded.
type progressLoadedMsg struct {
	progress map[string]*domain.Progress
	err      error
}

// topicExercisesLoadedMsg is sent when exercises for a topic are loaded.
type topicExercisesLoadedMsg struct {
	exercises []*domain.ExerciseRef
	err       error
}
