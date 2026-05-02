package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"godojo/internal/adapters/gemini"
	"godojo/internal/adapters/repository"
	"godojo/internal/adapters/runner"
	"godojo/internal/adapters/store"
	"godojo/internal/adapters/tui"
	"godojo/internal/core/services"
)

func main() {
	// 1. Set up progress directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: no se pudo determinar el directorio home: %v\n", err)
		os.Exit(1)
	}
	godojoDir := filepath.Join(homeDir, ".godojo")
	if err := os.MkdirAll(godojoDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: no se pudo crear %s: %v\n", godojoDir, err)
		os.Exit(1)
	}
	progressPath := filepath.Join(godojoDir, "progress.json")

	// 2. Create adapters (real implementations)
	progressStore := store.NewJSONProgressStore(progressPath)
	testRunner := runner.NewGoTestRunner(30 * time.Second)
	exerciseRepo := repository.NewEmbedExerciseRepo()
	hintProvider := gemini.NewHintProvider()

	// 3. Create services (inject adapters)
	roadmapSvc := services.NewRoadmapService()
	exerciseSvc := services.NewExerciseService(exerciseRepo)
	progressSvc := services.NewProgressService(progressStore)
	hintSvc := services.NewHintService(hintProvider)

	// 4. Create workspace path for exercises
	workspacePath := filepath.Join(homeDir, "godojo", "exercises")

	// 5. Create TUI model
	model := tui.NewModel(roadmapSvc, exerciseSvc, progressSvc, hintSvc, testRunner, workspacePath)

	// 6. Run Bubbletea
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
