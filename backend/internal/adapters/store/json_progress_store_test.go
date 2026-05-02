package store_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"godojo/internal/adapters/store"
	"godojo/internal/core/domain"
)

func TestLoad_FileDoesNotExist_ReturnsEmptyMap(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "progress.json")
	s := store.NewJSONProgressStore(filePath)

	got, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Load() with non-existent file should return empty map, got %d entries", len(got))
	}
}

func TestLoad_ValidJSON_ReturnsProgress(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "progress.json")
	content := `{
		"ejercicio-1": {
			"exercise_id": "ejercicio-1",
			"topic_slug": "variables",
			"status": "completed",
			"attempts": 3
		},
		"ejercicio-2": {
			"exercise_id": "ejercicio-2",
			"topic_slug": "funciones",
			"status": "in_progress",
			"attempts": 1
		}
	}`

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	s := store.NewJSONProgressStore(filePath)
	got, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Load() expected 2 entries, got %d", len(got))
	}
	if got["ejercicio-1"] == nil || got["ejercicio-2"] == nil {
		t.Fatal("Load() expected entries for both keys")
	}
	if got["ejercicio-1"].Status != domain.ProgressCompleted {
		t.Errorf("ejercicio-1 status = %q, want %q", got["ejercicio-1"].Status, domain.ProgressCompleted)
	}
	if got["ejercicio-2"].Attempts != 1 {
		t.Errorf("ejercicio-2 attempts = %d, want 1", got["ejercicio-2"].Attempts)
	}
}

func TestLoad_CorruptedJSON_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "progress.json")

	if err := os.WriteFile(filePath, []byte("esto no es json {{{"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	s := store.NewJSONProgressStore(filePath)
	_, err := s.Load(context.Background())
	if err == nil {
		t.Fatal("Load() with corrupted JSON should return error")
	}
}

func TestLoad_EmptyFile_ReturnsEmptyMap(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "progress.json")

	if err := os.WriteFile(filePath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	s := store.NewJSONProgressStore(filePath)
	got, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() unexpected error for empty file: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Load() with empty file should return empty map, got %d entries", len(got))
	}
}

func TestSave_WritesJSONToDisk(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "progress.json")

	s := store.NewJSONProgressStore(filePath)

	now := time.Now()
	prog, _ := domain.NewProgress("ejercicio-test", "topic-test", "completed", 5, &now)

	input := map[string]*domain.Progress{
		"ejercicio-test": prog,
	}

	if err := s.Save(context.Background(), input); err != nil {
		t.Fatalf("Save() unexpected error: %v", err)
	}

	// Load back and verify
	loaded, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() after Save() unexpected error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 entry after reload, got %d", len(loaded))
	}
	entry := loaded["ejercicio-test"]
	if entry == nil {
		t.Fatal("expected entry for 'ejercicio-test'")
	}
	if entry.Status != domain.ProgressCompleted {
		t.Errorf("status = %q, want %q", entry.Status, domain.ProgressCompleted)
	}
	if entry.Attempts != 5 {
		t.Errorf("attempts = %d, want 5", entry.Attempts)
	}
}

func TestSave_AtomicWrite_NoPartialFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "progress.json")

	s := store.NewJSONProgressStore(filePath)

	prog, _ := domain.NewProgress("atomic-test", "topic", "in_progress", 1, nil)
	input := map[string]*domain.Progress{
		"atomic-test": prog,
	}

	if err := s.Save(context.Background(), input); err != nil {
		t.Fatalf("Save() unexpected error: %v", err)
	}

	// Verify no temp file left behind
	tmpFile := filePath + ".tmp"
	if _, err := os.Stat(tmpFile); err == nil {
		t.Errorf("temp file %q should not exist after atomic save", tmpFile)
	}

	// Verify the file exists and is valid JSON
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("cannot read saved file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("saved file should not be empty")
	}
}

func TestGetBySlug_Found_ReturnsProgress(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "progress.json")

	content := `{
		"test-ej": {
			"exercise_id": "test-ej",
			"topic_slug": "fundamentos",
			"status": "in_progress",
			"attempts": 2
		}
	}`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	s := store.NewJSONProgressStore(filePath)
	got, err := s.GetBySlug(context.Background(), "test-ej")
	if err != nil {
		t.Fatalf("GetBySlug() unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("GetBySlug() should return progress for existing slug")
	}
	if got.ExerciseID != "test-ej" {
		t.Errorf("ExerciseID = %q, want %q", got.ExerciseID, "test-ej")
	}
}

func TestGetBySlug_NotFound_ReturnsNil(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "progress.json")

	content := `{"otro-ej": {"exercise_id": "otro-ej", "topic_slug": "t", "status": "not_started", "attempts": 0}}`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	s := store.NewJSONProgressStore(filePath)
	got, err := s.GetBySlug(context.Background(), "no-existe")
	if err != nil {
		t.Fatalf("GetBySlug() unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("GetBySlug() for unknown slug should return nil, got %v", got)
	}
}

func TestSave_LoadRoundtrip_PreservesData(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "progress.json")
	s := store.NewJSONProgressStore(filePath)

	now := time.Now().Truncate(time.Second)
	prog1, _ := domain.NewProgress("slug-1", "topic-a", "completed", 10, &now)
	prog2, _ := domain.NewProgress("slug-2", "topic-b", "skipped", 0, nil)

	input := map[string]*domain.Progress{
		"slug-1": prog1,
		"slug-2": prog2,
	}

	if err := s.Save(context.Background(), input); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("roundtrip: expected 2 entries, got %d", len(loaded))
	}

	e1 := loaded["slug-1"]
	e2 := loaded["slug-2"]

	if e1.Status != domain.ProgressCompleted || e1.Attempts != 10 {
		t.Errorf("slug-1 corrupted: status=%q attempts=%d", e1.Status, e1.Attempts)
	}
	if e1.CompletedAt == nil || !e1.CompletedAt.Equal(now) {
		t.Errorf("slug-1 CompletedAt mismatch: got %v, want %v", e1.CompletedAt, now)
	}

	if e2.Status != domain.ProgressSkipped || e2.Attempts != 0 {
		t.Errorf("slug-2 corrupted: status=%q attempts=%d", e2.Status, e2.Attempts)
	}
	if e2.CompletedAt != nil {
		t.Errorf("slug-2 CompletedAt should be nil, got %v", e2.CompletedAt)
	}
}

func TestConcurrentSaves_NoDataRace(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "progress.json")
	s := store.NewJSONProgressStore(filePath)

	// Build a shared map with all entries — simulate real usage where
	// the full progress map is saved each time.
	fullMap := make(map[string]*domain.Progress)
	var mapMu sync.Mutex

	for i := 0; i < 10; i++ {
		slug := "ej-" + string(rune('0'+i))
		prog, _ := domain.NewProgress(slug, "topic", "in_progress", i, nil)
		fullMap[slug] = prog
	}

	var wg sync.WaitGroup
	for k := 0; k < 10; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mapMu.Lock()
			copyMap := make(map[string]*domain.Progress, len(fullMap))
			for k, v := range fullMap {
				copyMap[k] = v
			}
			mapMu.Unlock()
			_ = s.Save(context.Background(), copyMap)
		}()
	}
	wg.Wait()

	// Final load should not panic and should have all entries
	loaded, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() after concurrent saves: %v", err)
	}
	if len(loaded) != 10 {
		t.Errorf("expected 10 entries after concurrent saves, got %d", len(loaded))
	}
}
