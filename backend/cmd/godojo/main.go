package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"godojo/internal/adapters/chatstore"
	"godojo/internal/adapters/gemini"
	"godojo/internal/adapters/repository"
	"godojo/internal/adapters/store"
	"godojo/internal/adapters/tui"
	"godojo/internal/adapters/workspace"
	"godojo/internal/core"
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
		fmt.Fprintf(os.Stderr, "   Crea un archivo .env en la raíz del proyecto con: GEMINI_API_KEY=tu-key\n")
	}

	// 1. Set up data directory
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

	// 2. Create adapters
	progressStore := store.NewJSONProgressStore(progressPath)
	exerciseRepo := repository.NewEmbedExerciseRepo()
	chatProviderInstance := gemini.NewChatProvider()
	chatStore := chatstore.NewChatStore(filepath.Join(homeDir, ".godojo", "sessions"))

	// 3. Create services
	roadmapSvc := services.NewRoadmapService()
	exerciseSvc := services.NewExerciseService(exerciseRepo)
	progressSvc := services.NewProgressService(progressStore)
	hintProvider := gemini.NewHintProvider()
	hintSvc := services.NewHintService(hintProvider)

	// 4. Create SenseiService (agentic AI loop)
	toolRegistry := core.NewToolRegistry()
	workspacePath := filepath.Join(homeDir, ".godojo", "workspace")
	workspaceManager := workspace.NewWorkspaceManager(workspacePath)
	senseiSvc := services.NewSenseiService(chatProviderInstance, toolRegistry, workspaceManager, roadmapSvc)

	// 5. Create TUI model (simplified: only sensei + chat)
	senseiSystemPrompt := `Sos un sensei de Go, un maestro experto en programación Go.
Ayudás a estudiantes a aprender Go con paciencia, ejemplos claros y preguntas socráticas.
Usá español rioplatense (voseo).
Cuando crees ejercicios de un tema, usá topic_slug con el slug de ese tema para guardar los archivos en una carpeta temática; si ya hay varias clases, anidá subcarpetas debajo de esa carpeta.
Cuando generes el contenido de un ejercicio, preferí un scaffold con pistas dentro del mismo archivo: comentarios TODO, funciones incompletas, ayudas graduales y mensajes que inviten a pensar. No des la solución completa si no te la piden explícitamente.`

	model := tui.NewModel(senseiSvc, chatStore, senseiSystemPrompt)

	// Suppress unused variable warnings for services still wired but unused in TUI
	_ = exerciseSvc
	_ = progressSvc
	_ = hintSvc

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
