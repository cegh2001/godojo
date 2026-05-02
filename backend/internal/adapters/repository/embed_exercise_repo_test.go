package repository_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"godojo/internal/adapters/repository"
	"godojo/internal/core/domain"
)

func TestGetBySlug_HolaMundo_ReturnsExercise(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	ex, err := repo.GetBySlug(context.Background(), "variables", "hola-mundo")
	if err != nil {
		t.Fatalf("GetBySlug() unexpected error: %v", err)
	}
	if ex == nil {
		t.Fatal("GetBySlug() should return exercise for 'hola-mundo'")
	}
	if ex.Slug != "hola-mundo" {
		t.Errorf("slug = %q, want %q", ex.Slug, "hola-mundo")
	}
	if ex.Title == "" {
		t.Error("title should not be empty")
	}
	if ex.TopicSlug != "variables" {
		t.Errorf("topicSlug = %q, want %q", ex.TopicSlug, "variables")
	}
	if ex.TemplateCode == "" {
		t.Error("template code should not be empty")
	}
	if ex.TestCode == "" {
		t.Error("test code should not be empty")
	}
}

func TestGetBySlug_Variables_ReturnsExercise(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	ex, err := repo.GetBySlug(context.Background(), "variables", "variables")
	if err != nil {
		t.Fatalf("GetBySlug() unexpected error: %v", err)
	}
	if ex.Slug != "variables" {
		t.Errorf("slug = %q, want %q", ex.Slug, "variables")
	}
	if ex.TemplateCode == "" || ex.TestCode == "" {
		t.Error("exercise should have non-empty template and test code")
	}
}

func TestGetBySlug_Funciones_ReturnsExercise(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	ex, err := repo.GetBySlug(context.Background(), "funciones", "funciones")
	if err != nil {
		t.Fatalf("GetBySlug() unexpected error: %v", err)
	}
	if ex.TemplateCode == "" || ex.TestCode == "" {
		t.Error("exercise should have non-empty template and test code")
	}
}

func TestGetBySlug_NotFound_ReturnsError(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	_, err := repo.GetBySlug(context.Background(), "variables", "no-existe")
	if err == nil {
		t.Fatal("GetBySlug() should return error for non-existent exercise")
	}
}

func TestGetBySlug_InvalidTopic_ReturnsError(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	_, err := repo.GetBySlug(context.Background(), "tema-inexistente", "hola-mundo")
	if err == nil {
		t.Fatal("GetBySlug() should return error for invalid topic")
	}
}

func TestListByTopic_Variables_ReturnsMultipleExercises(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	refs, err := repo.ListByTopic(context.Background(), "variables")
	if err != nil {
		t.Fatalf("ListByTopic() unexpected error: %v", err)
	}
	if len(refs) < 2 {
		t.Errorf("expected at least 2 exercises for variables topic, got %d", len(refs))
	}
	// Verify each ref has required fields
	for _, ref := range refs {
		if ref.Slug == "" {
			t.Error("exercise ref slug should not be empty")
		}
		if ref.Title == "" {
			t.Error("exercise ref title should not be empty")
		}
		if ref.Difficulty == "" {
			t.Error("exercise ref difficulty should not be empty")
		}
	}
}

func TestListByTopic_Funciones_ReturnsExercises(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	refs, err := repo.ListByTopic(context.Background(), "funciones")
	if err != nil {
		t.Fatalf("ListByTopic() unexpected error: %v", err)
	}
	if len(refs) < 1 {
		t.Errorf("expected at least 1 exercise for funciones topic, got %d", len(refs))
	}
}

func TestListByTopic_Arrays_ReturnsExercises(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	refs, err := repo.ListByTopic(context.Background(), "arrays")
	if err != nil {
		t.Fatalf("ListByTopic() unexpected error: %v", err)
	}
	if len(refs) < 1 {
		t.Errorf("expected at least 1 exercise for arrays topic, got %d", len(refs))
	}
}

func TestListByTopic_Structs_ReturnsExercises(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	refs, err := repo.ListByTopic(context.Background(), "structs")
	if err != nil {
		t.Fatalf("ListByTopic() unexpected error: %v", err)
	}
	if len(refs) < 1 {
		t.Errorf("expected at least 1 exercise for structs topic, got %d", len(refs))
	}
}

func TestListByTopic_UnknownTopic_ReturnsError(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	_, err := repo.ListByTopic(context.Background(), "no-existe")
	if err == nil {
		t.Fatal("ListByTopic() should return error for unknown topic")
	}
}

func TestListByTopic_AllHaveUniqueSlugs(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()

	// Test a few topics
	for _, topic := range []string{"variables", "tipos", "funciones", "arrays", "slices", "maps", "structs"} {
		refs, err := repo.ListByTopic(context.Background(), topic)
		if err != nil {
			t.Fatalf("ListByTopic(%q) unexpected error: %v", topic, err)
		}

		seen := make(map[string]bool)
		for _, ref := range refs {
			if seen[ref.Slug] {
				t.Errorf("duplicate slug %q in topic %q", ref.Slug, topic)
			}
			seen[ref.Slug] = true
		}
	}
}

func TestGenerateFiles_CreatesExpectedStructure(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	ex, err := repo.GetBySlug(context.Background(), "variables", "variables")
	if err != nil {
		t.Fatalf("GetBySlug() failed: %v", err)
	}

	tmpDir := t.TempDir()
	workspace := filepath.Join(tmpDir, "variables")

	if err := repo.GenerateFiles(context.Background(), ex, workspace); err != nil {
		t.Fatalf("GenerateFiles() unexpected error: %v", err)
	}

	// Verify files exist
	for _, filename := range []string{"ejercicio.go", "ejercicio_test.go", "go.mod"} {
		path := filepath.Join(workspace, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", filename)
		}
	}

	// Verify ejercicio.go content matches template
	content, err := os.ReadFile(filepath.Join(workspace, "ejercicio.go"))
	if err != nil {
		t.Fatalf("cannot read generated file: %v", err)
	}
	if len(content) == 0 {
		t.Error("ejercicio.go should not be empty")
	}
}

func TestGenerateFiles_Overwrite_ReturnsError(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	ex, err := repo.GetBySlug(context.Background(), "variables", "variables")
	if err != nil {
		t.Fatalf("GetBySlug() failed: %v", err)
	}

	tmpDir := t.TempDir()
	workspace := filepath.Join(tmpDir, "variables")

	// First generation should succeed
	if err := repo.GenerateFiles(context.Background(), ex, workspace); err != nil {
		t.Fatalf("first GenerateFiles() unexpected error: %v", err)
	}

	// Second generation should fail because files exist
	err = repo.GenerateFiles(context.Background(), ex, workspace)
	if err == nil {
		t.Fatal("GenerateFiles() should return error when files already exist")
	}
	if !strings.Contains(err.Error(), "ya existen") && !strings.Contains(err.Error(), "sobrescribir") {
		t.Errorf("error message should mention existing files, got: %v", err)
	}
}

func TestGenerateFiles_CreatesGoMod(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	ex, err := repo.GetBySlug(context.Background(), "variables", "variables")
	if err != nil {
		t.Fatalf("GetBySlug() failed: %v", err)
	}

	tmpDir := t.TempDir()
	workspace := filepath.Join(tmpDir, "variables")

	if err := repo.GenerateFiles(context.Background(), ex, workspace); err != nil {
		t.Fatalf("GenerateFiles() unexpected error: %v", err)
	}

	modContent, err := os.ReadFile(filepath.Join(workspace, "go.mod"))
	if err != nil {
		t.Fatalf("cannot read go.mod: %v", err)
	}
	if !strings.Contains(string(modContent), "module") {
		t.Errorf("go.mod should contain 'module' directive, got: %s", string(modContent))
	}
}

func TestGenerateFiles_PartialOverwrite_ReturnsError(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	ex, err := repo.GetBySlug(context.Background(), "variables", "variables")
	if err != nil {
		t.Fatalf("GetBySlug() failed: %v", err)
	}

	tmpDir := t.TempDir()
	workspace := filepath.Join(tmpDir, "variables")

	// Create only one file manually to simulate partial state
	os.MkdirAll(workspace, 0755)
	os.WriteFile(filepath.Join(workspace, "ejercicio.go"), []byte("código del usuario"), 0644)

	err = repo.GenerateFiles(context.Background(), ex, workspace)
	if err == nil {
		t.Fatal("GenerateFiles() should return error when any file already exists")
	}
}

func TestGenerateFiles_InvalidWorkspace_ReturnsError(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	ex, err := repo.GetBySlug(context.Background(), "variables", "variables")
	if err != nil {
		t.Fatalf("GetBySlug() failed: %v", err)
	}

	// Use an invalid path (e.g., a file path where a dir should be)
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "not-a-dir")
	os.WriteFile(filePath, []byte("i am a file"), 0644)

	err = repo.GenerateFiles(context.Background(), ex, filePath)
	if err == nil {
		t.Fatal("GenerateFiles() should return error for invalid workspace path")
	}
}

func TestGenerateFiles_Paquetes_CreatesExtraFiles(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	ex, err := repo.GetBySlug(context.Background(), "packages", "paquetes")
	if err != nil {
		t.Fatalf("GetBySlug() failed: %v", err)
	}

	tmpDir := t.TempDir()
	workspace := filepath.Join(tmpDir, "paquetes")

	if err := repo.GenerateFiles(context.Background(), ex, workspace); err != nil {
		t.Fatalf("GenerateFiles() unexpected error: %v", err)
	}

	// Check for extra file calculadora/calculadora.go
	calcPath := filepath.Join(workspace, "calculadora", "calculadora.go")
	content, err := os.ReadFile(calcPath)
	if err != nil {
		t.Fatalf("expected calculadora/calculadora.go to exist: %v", err)
	}
	if len(content) == 0 {
		t.Error("calculadora/calculadora.go should not be empty")
	}
	if !strings.Contains(string(content), "package calculadora") {
		t.Error("calculadora/calculadora.go should have package calculadora declaration")
	}
}

func TestGetBySlug_Tipos_ReturnsExercise(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	ex, err := repo.GetBySlug(context.Background(), "tipos", "tipos")
	if err != nil {
		t.Fatalf("GetBySlug() unexpected error: %v", err)
	}
	if ex.TemplateCode == "" || ex.TestCode == "" {
		t.Error("exercise should have non-empty template and test code")
	}
}

func TestGetBySlug_ControlFlujo_ReturnsExercise(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()
	ex, err := repo.GetBySlug(context.Background(), "control-de-flujo", "control-flujo")
	if err != nil {
		t.Fatalf("GetBySlug() unexpected error: %v", err)
	}
	if ex.TemplateCode == "" || ex.TestCode == "" {
		t.Error("exercise should have non-empty template and test code")
	}
}

func TestGetBySlug_AllPhase2Exercises(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()

	exercises := map[string]string{
		"arrays":  "arrays",
		"slices":  "slices",
		"maps":    "maps",
		"structs": "structs",
	}

	for topic, slug := range exercises {
		ex, err := repo.GetBySlug(context.Background(), topic, slug)
		if err != nil {
			t.Errorf("GetBySlug(%q, %q) unexpected error: %v", topic, slug, err)
			continue
		}
		if ex.TemplateCode == "" || ex.TestCode == "" {
			t.Errorf("exercise %q should have non-empty template and test code", slug)
		}
	}
}

func TestExerciseRegistry_HasAllTopics(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()

	// All 9 topics from RoadmapService must have at least 1 exercise
	requiredTopics := []string{
		"variables", "tipos", "funciones", "packages", "control-de-flujo",
		"arrays", "slices", "maps", "structs",
	}

	for _, topic := range requiredTopics {
		_, err := repo.ListByTopic(context.Background(), topic)
		if err != nil {
			t.Errorf("topic %q should have exercises but got error: %v", topic, err)
		}
	}
}

func TestGenerateFiles_AllExercises(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()

	// Test that GenerateFiles works for every exercise
	tests := []struct {
		topicSlug    string
		exerciseSlug string
	}{
		{"variables", "hola-mundo"},
		{"variables", "variables"},
		{"tipos", "tipos"},
		{"funciones", "funciones"},
		{"packages", "paquetes"},
		{"control-de-flujo", "control-flujo"},
		{"arrays", "arrays"},
		{"slices", "slices"},
		{"maps", "maps"},
		{"structs", "structs"},
	}

	for _, tt := range tests {
		t.Run(tt.exerciseSlug, func(t *testing.T) {
			ex, err := repo.GetBySlug(context.Background(), tt.topicSlug, tt.exerciseSlug)
			if err != nil {
				t.Fatalf("GetBySlug() failed: %v", err)
			}

			tmpDir := t.TempDir()
			workspace := filepath.Join(tmpDir, tt.exerciseSlug)

			if err := repo.GenerateFiles(context.Background(), ex, workspace); err != nil {
				t.Fatalf("GenerateFiles() failed: %v", err)
			}

			// Verify core files exist
			for _, f := range []string{"ejercicio.go", "ejercicio_test.go", "go.mod"} {
				if _, err := os.Stat(filepath.Join(workspace, f)); os.IsNotExist(err) {
					t.Errorf("expected file %s to exist for exercise %s", f, tt.exerciseSlug)
				}
			}
		})
	}
}

// Test that generated exercise files reference domain concepts correctly
func TestExerciseRef_AllHaveDifficulty(t *testing.T) {
	repo := repository.NewEmbedExerciseRepo()

	for _, topic := range []string{"variables", "tipos", "funciones", "packages", "control-de-flujo", "arrays", "slices", "maps", "structs"} {
		refs, _ := repo.ListByTopic(context.Background(), topic)
		for _, ref := range refs {
			if ref.Difficulty != "easy" && ref.Difficulty != "medium" && ref.Difficulty != "hard" {
				t.Errorf("exercise %q in topic %q has invalid difficulty: %q", ref.Slug, topic, ref.Difficulty)
			}
		}
	}
}

// Helper interface for testing
var _ domain.Difficulty = "easy"
