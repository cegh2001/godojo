package tui

import (
	"context"
	"fmt"
	"time"

	"godojo/internal/core/domain"
	"godojo/internal/core/ports"

	tea "github.com/charmbracelet/bubbletea"
)

// runTestsCmd returns a Bubbletea command that executes tests and sends back results.
// It accepts a TestRunner interface to actually run the tests.
func runTestsCmd(runner ports.TestRunner, workspace string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		result, err := runner.Run(ctx, workspace)
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

		timeout := time.NewTimer(30 * time.Second)
		defer timeout.Stop()

		// Wait for an actual hint or error value, ignoring closed channels with no payload.
		for hintCh != nil || errCh != nil {
			select {
			case hint, ok := <-hintCh:
				if ok && hint != nil {
					return hintResultMsg{hint: hint}
				}
				hintCh = nil
			case err, ok := <-errCh:
				if ok && err != nil {
					return hintErrorMsg{err: err}
				}
				errCh = nil
			case <-timeout.C:
				return hintErrorMsg{err: fmt.Errorf("timeout: el sensei no respondió a tiempo")}
			}
		}

		return hintErrorMsg{err: fmt.Errorf("el sensei no respondió")}
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
