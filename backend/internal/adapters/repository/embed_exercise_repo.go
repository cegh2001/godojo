package repository

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"godojo/internal/core/domain"
)

//go:embed data
var exerciseFS embed.FS

// exerciseMeta defines metadata for each exercise embedded in the FS.
type exerciseMeta struct {
	slug       string
	title      string
	topicSlug  string
	difficulty domain.Difficulty
	dir        string // relative path within embedded FS, e.g. "data/fundamentos/variables"
}

// exerciseManifest lists all exercises available in the platform.
var exerciseManifest = []exerciseMeta{
	// Fase 1: Fundamentos
	{slug: "hola-mundo", title: "¡Hola, Gopher!", topicSlug: "variables", difficulty: domain.DifficultyEasy, dir: "data/fundamentos/hola-mundo"},
	{slug: "variables", title: "Declaración de Variables", topicSlug: "variables", difficulty: domain.DifficultyEasy, dir: "data/fundamentos/variables"},
	{slug: "tipos", title: "Tipos Básicos y Conversiones", topicSlug: "tipos", difficulty: domain.DifficultyEasy, dir: "data/fundamentos/tipos"},
	{slug: "funciones", title: "Funciones y Múltiples Retornos", topicSlug: "funciones", difficulty: domain.DifficultyMedium, dir: "data/fundamentos/funciones"},
	{slug: "paquetes", title: "Paquetes y Visibilidad", topicSlug: "packages", difficulty: domain.DifficultyMedium, dir: "data/fundamentos/paquetes"},
	{slug: "control-flujo", title: "Control de Flujo", topicSlug: "control-de-flujo", difficulty: domain.DifficultyMedium, dir: "data/fundamentos/control-flujo"},
	// Fase 2: Estructuras de Datos
	{slug: "arrays", title: "Arrays en Go", topicSlug: "arrays", difficulty: domain.DifficultyEasy, dir: "data/estructuras/arrays"},
	{slug: "slices", title: "Slices y Operaciones", topicSlug: "slices", difficulty: domain.DifficultyMedium, dir: "data/estructuras/slices"},
	{slug: "maps", title: "Maps en Go", topicSlug: "maps", difficulty: domain.DifficultyMedium, dir: "data/estructuras/maps"},
	{slug: "structs", title: "Structs y Métodos", topicSlug: "structs", difficulty: domain.DifficultyHard, dir: "data/estructuras/structs"},
}

// extraFiles defines additional files to write alongside the main ejercicio.go
// and ejercicio_test.go for specific exercises (e.g., sub-packages).
var extraFiles = map[string]map[string]string{
	"paquetes": {
		"calculadora/calculadora.go": "data/fundamentos/paquetes/calculadora/calculadora.go",
	},
}

// EmbedExerciseRepo implements ports.ExerciseRepository using Go's embed package.
type EmbedExerciseRepo struct {
	registry map[string][]*domain.Exercise // topicSlug → exercises
	cache    map[string]*domain.Exercise   // topicSlug/exerciseSlug → exercise
}

// NewEmbedExerciseRepo creates a new EmbedExerciseRepo and loads all exercises
// from the embedded filesystem.
func NewEmbedExerciseRepo() *EmbedExerciseRepo {
	repo := &EmbedExerciseRepo{
		registry: make(map[string][]*domain.Exercise),
		cache:    make(map[string]*domain.Exercise),
	}

	for _, meta := range exerciseManifest {
		tmplPath := path.Join(meta.dir, "ejercicio.go")
		tmplCode, err := exerciseFS.ReadFile(tmplPath)
		if err != nil {
			panic(fmt.Sprintf("embedded template %s not found: %v", tmplPath, err))
		}

		testPath := path.Join(meta.dir, "ejercicio_test.go")
		testCode, err := exerciseFS.ReadFile(testPath)
		if err != nil {
			panic(fmt.Sprintf("embedded test %s not found: %v", testPath, err))
		}

		ex, err := domain.NewExercise(
			meta.slug,
			meta.title,
			meta.topicSlug,
			string(tmplCode),
			string(testCode),
			"",
		)
		if err != nil {
			panic(fmt.Sprintf("invalid exercise %q: %v", meta.slug, err))
		}

		repo.registry[meta.topicSlug] = append(repo.registry[meta.topicSlug], ex)
		cacheKey := meta.topicSlug + "/" + meta.slug
		repo.cache[cacheKey] = ex
	}

	return repo
}

// GetBySlug retrieves an exercise by topic slug and exercise slug.
// If topicSlug is empty, it searches all topics for a matching exercise.
func (r *EmbedExerciseRepo) GetBySlug(ctx context.Context, topicSlug, exerciseSlug string) (*domain.Exercise, error) {
	// If topicSlug is provided, do a direct lookup
	if topicSlug != "" {
		cacheKey := topicSlug + "/" + exerciseSlug
		if ex, ok := r.cache[cacheKey]; ok {
			return ex, nil
		}
		return nil, fmt.Errorf("ejercicio %q no encontrado en el tema %q", exerciseSlug, topicSlug)
	}

	// Search all topics for the exercise
	for _, exercises := range r.registry {
		for _, ex := range exercises {
			if ex.Slug == exerciseSlug {
				return ex, nil
			}
		}
	}
	return nil, fmt.Errorf("ejercicio %q no encontrado", exerciseSlug)
}

// ListByTopic returns exercise references for a given topic.
func (r *EmbedExerciseRepo) ListByTopic(ctx context.Context, topicSlug string) ([]*domain.ExerciseRef, error) {
	exercises, ok := r.registry[topicSlug]
	if !ok || len(exercises) == 0 {
		return nil, fmt.Errorf("tema %q no encontrado", topicSlug)
	}

	refs := make([]*domain.ExerciseRef, 0, len(exercises))
	for _, ex := range exercises {
		// Determine difficulty from manifest
		diff := domain.DifficultyEasy
		for _, meta := range exerciseManifest {
			if meta.slug == ex.Slug && meta.topicSlug == ex.TopicSlug {
				diff = meta.difficulty
				break
			}
		}
		ref, err := domain.NewExerciseRef(ex.Slug, ex.Title, string(diff))
		if err != nil {
			return nil, fmt.Errorf("error al cargar referencia %q: %w", ex.Slug, err)
		}
		refs = append(refs, ref)
	}

	return refs, nil
}

// GenerateFiles writes ejercicio.go, ejercicio_test.go, go.mod, and any extra
// files to the workspace directory.
func (r *EmbedExerciseRepo) GenerateFiles(ctx context.Context, exercise *domain.Exercise, workspacePath string) error {
	// Collect all files to write
	filesToWrite := map[string]string{
		"ejercicio.go":      exercise.TemplateCode,
		"ejercicio_test.go": exercise.TestCode,
	}

	// Check for extra files (e.g., calculadora/calculadora.go for paquetes exercise)
	if extra, ok := extraFiles[exercise.Slug]; ok {
		for dstName, srcPath := range extra {
			content, err := exerciseFS.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("archivo extra %q no encontrado: %w", srcPath, err)
			}
			filesToWrite[dstName] = string(content)
		}
	}

	// Check which files already exist — don't overwrite user's work
	existingFiles := []string{}
	for filename := range filesToWrite {
		p := filepath.Join(workspacePath, filename)
		if _, err := os.Stat(p); err == nil {
			existingFiles = append(existingFiles, filename)
		}
	}
	modPath := filepath.Join(workspacePath, "go.mod")
	if _, err := os.Stat(modPath); err == nil {
		existingFiles = append(existingFiles, "go.mod")
	}

	if len(existingFiles) > 0 {
		return fmt.Errorf(
			"los siguientes archivos ya existen y no se van a sobrescribir: %s. Borralos manualmente si querés empezar de nuevo.",
			strings.Join(existingFiles, ", "),
		)
	}

	// Validate/create workspace directory
	info, err := os.Stat(workspacePath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(workspacePath, 0755); err != nil {
				return fmt.Errorf("no se pudo crear el directorio de trabajo: %w", err)
			}
			// Create subdirectories for extra files
			for filename := range filesToWrite {
				if dir := filepath.Dir(filepath.Join(workspacePath, filename)); dir != workspacePath {
					if err := os.MkdirAll(dir, 0755); err != nil {
						return fmt.Errorf("no se pudo crear el subdirectorio %s: %w", dir, err)
					}
				}
			}
		} else {
			return fmt.Errorf("no se pudo acceder al directorio de trabajo: %w", err)
		}
	} else if !info.IsDir() {
		return fmt.Errorf("la ruta de trabajo %q no es un directorio", workspacePath)
	}

	// Write all files
	for filename, content := range filesToWrite {
		p := filepath.Join(workspacePath, filename)
		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			return fmt.Errorf("no se pudo crear el directorio para %s: %w", filename, err)
		}
		// Strip //go:build ignore so the exercise can compile
		cleaned := stripBuildIgnore(content)
		if err := os.WriteFile(p, []byte(cleaned), 0644); err != nil {
			return fmt.Errorf("no se pudo escribir %s: %w", filename, err)
		}
	}

	// Write go.mod
	modContent := fmt.Sprintf("module %s\n\ngo 1.21\n", exercise.Slug)
	if err := os.WriteFile(modPath, []byte(modContent), 0644); err != nil {
		return fmt.Errorf("no se pudo escribir go.mod: %w", err)
	}

	return nil
}

// stripBuildIgnore removes the //go:build ignore directive from generated files.
// The embedded templates need this directive to avoid being compiled as part of GoDojo,
// but exercise files generated for the user must be compilable.
func stripBuildIgnore(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "//go:build ignore" {
		// Remove the build tag and the following empty line if present
		if len(lines) > 1 && lines[1] == "" {
			return strings.Join(lines[2:], "\n")
		}
		return strings.Join(lines[1:], "\n")
	}
	return content
}
