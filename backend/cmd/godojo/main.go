package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"godojo/internal/adapters/chatstore"
	"godojo/internal/adapters/gemini"
	"godojo/internal/adapters/repository"
	"godojo/internal/adapters/runner"
	"godojo/internal/adapters/store"
	"godojo/internal/adapters/tui"
	"godojo/internal/core/services"
)

func main() {
	// 0. Load .env file from project root (one level up from backend/)
	loadEnv("..")

	// Check if Gemini API key is loaded
	if os.Getenv("GEMINI_API_KEY") != "" {
		fmt.Fprintf(os.Stderr, "🔑 Gemini API key cargada del .env\n")
	} else {
		fmt.Fprintf(os.Stderr, "⚠️  GEMINI_API_KEY no encontrada. Las pistas del sensei no estarán disponibles.\n")
		fmt.Fprintf(os.Stderr, "   Creá un archivo .env en la raíz del proyecto con: GEMINI_API_KEY=tu-key\n")
	}

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
	chatStore := chatstore.NewChatStore(filepath.Join(homeDir, ".godojo", "sessions"))
	chatProviderInstance := gemini.NewChatProvider()

	// 3. Create services (inject adapters)
	roadmapSvc := services.NewRoadmapService()
	exerciseSvc := services.NewExerciseService(exerciseRepo)
	progressSvc := services.NewProgressService(progressStore)
	hintSvc := services.NewHintService(hintProvider)

	// 4. Create workspace path for exercises
	workspacePath := filepath.Join(homeDir, "godojo", "exercises")

	// 5. Create TUI model
	model := tui.NewModel(roadmapSvc, exerciseSvc, progressSvc, hintSvc, testRunner, exerciseRepo, workspacePath, chatStore, chatProviderInstance)

	// 6. Run Bubbletea
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// loadEnv reads a .env file and sets environment variables.
// Only sets vars that aren't already set in the environment.
func loadEnv(envDir string) {
	envPath := filepath.Join(envDir, ".env")
	f, err := os.Open(envPath)
	if err != nil {
		return // .env is optional
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Only set if not already in environment
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}
