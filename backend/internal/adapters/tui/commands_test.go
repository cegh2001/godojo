package tui

import (
	"context"
	"errors"
	"testing"
	"time"

	"godojo/internal/core/domain"
)

// --- Mock TestRunner ---

type mockTestRunner struct {
	result *domain.TestResult
	err    error
}

func (m *mockTestRunner) Run(ctx context.Context, workspace string) (*domain.TestResult, error) {
	return m.result, m.err
}

// --- Mock HintService ---

type mockHintService struct {
	hint   *domain.Hint
	err    error
	rateLimited bool
}

func (m *mockHintService) RequestHint(exercise *domain.Exercise, testOutput string) (<-chan *domain.Hint, <-chan error) {
	if m.rateLimited {
		return nil, nil
	}
	hintCh := make(chan *domain.Hint, 1)
	errCh := make(chan error, 1)

	go func() {
		if m.err != nil {
			errCh <- m.err
		} else if m.hint != nil {
			hintCh <- m.hint
		}
		close(hintCh)
		close(errCh)
	}()

	return hintCh, errCh
}

func (m *mockHintService) IsAvailable() bool {
	return !m.rateLimited
}

// --- Mock ProgressService ---

type mockProgressService struct {
	progress map[string]*domain.Progress
	err      error
}

func (m *mockProgressService) MarkStarted(exerciseSlug string) error           { return nil }
func (m *mockProgressService) MarkCompleted(exerciseSlug string) error         { return nil }
func (m *mockProgressService) GetCompletionPercent(topicSlug string) (float64, error) { return 0, nil }
func (m *mockProgressService) GetProgress(exerciseSlug string) (*domain.Progress, error) {
	return nil, nil
}
func (m *mockProgressService) GetAllProgress() (map[string]*domain.Progress, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.progress, nil
}

// --- Mock ExerciseService ---

type mockExerciseService struct {
	exercises []*domain.ExerciseRef
	exercise  *domain.Exercise
	err       error
}

func (m *mockExerciseService) StartExercise(slug string) (*domain.Exercise, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.exercise, nil
}
func (m *mockExerciseService) GetExercisesByTopic(topicSlug string) ([]*domain.ExerciseRef, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.exercises, nil
}
func (m *mockExerciseService) ValidateExercise(slug string, testResult *domain.TestResult) (bool, error) {
	return testResult.Passed, nil
}

// --- Tests ---

func TestRunTestsCmd_ReturnsTestResultMsg(t *testing.T) {
	runner := &mockTestRunner{
		result: &domain.TestResult{Passed: true, Output: "PASS", Duration: 100 * time.Millisecond},
	}

	cmd := runTestsCmd(runner, "/workspace")
	msg := cmd()

	trMsg, ok := msg.(testResultMsg)
	if !ok {
		t.Fatalf("expected testResultMsg, got %T", msg)
	}
	if trMsg.result == nil {
		t.Fatal("result should not be nil")
	}
	if !trMsg.result.Passed {
		t.Error("result.Passed should be true")
	}
	if trMsg.err != nil {
		t.Errorf("unexpected error: %v", trMsg.err)
	}
}

func TestRunTestsCmd_RunnerError_ReturnsTestResultMsgWithError(t *testing.T) {
	runner := &mockTestRunner{
		err: errors.New("test failed to run"),
	}

	cmd := runTestsCmd(runner, "/workspace")
	msg := cmd()

	trMsg, ok := msg.(testResultMsg)
	if !ok {
		t.Fatalf("expected testResultMsg, got %T", msg)
	}
	if trMsg.err == nil {
		t.Fatal("expected error but got nil")
	}
	if trMsg.result == nil {
		t.Fatal("result should not be nil even on error")
	}
	if trMsg.result.Passed {
		t.Error("result.Passed should be false on error")
	}
}

func TestFetchHintCmd_ReturnsHintResultMsg(t *testing.T) {
	hintSvc := &mockHintService{
		hint: &domain.Hint{Content: "¿Probaste con un bucle?"},
	}

	cmd := fetchHintCmd(hintSvc, &domain.Exercise{Slug: "ex-1"}, "FAIL output")
	msg := cmd()

	hMsg, ok := msg.(hintResultMsg)
	if !ok {
		t.Fatalf("expected hintResultMsg, got %T", msg)
	}
	if hMsg.hint == nil {
		t.Fatal("hint should not be nil")
	}
	if hMsg.hint.Content != "¿Probaste con un bucle?" {
		t.Errorf("hint content = %q, want %q", hMsg.hint.Content, "¿Probaste con un bucle?")
	}
}

func TestFetchHintCmd_ReturnsHintErrorMsg(t *testing.T) {
	hintSvc := &mockHintService{
		err: errors.New("API error"),
	}

	cmd := fetchHintCmd(hintSvc, &domain.Exercise{Slug: "ex-1"}, "FAIL output")
	msg := cmd()

	_, ok := msg.(hintErrorMsg)
	if !ok {
		t.Fatalf("expected hintErrorMsg, got %T", msg)
	}
}

func TestFetchHintCmd_RateLimited_ReturnsError(t *testing.T) {
	hintSvc := &mockHintService{
		rateLimited: true,
	}

	cmd := fetchHintCmd(hintSvc, &domain.Exercise{Slug: "ex-1"}, "FAIL output")
	msg := cmd()

	_, ok := msg.(hintErrorMsg)
	if !ok {
		t.Fatalf("expected hintErrorMsg when rate limited, got %T", msg)
	}
}

func TestFetchHintCmd_NilService_ReturnsError(t *testing.T) {
	cmd := fetchHintCmd(nil, &domain.Exercise{Slug: "ex-1"}, "FAIL output")
	msg := cmd()

	_, ok := msg.(hintErrorMsg)
	if !ok {
		t.Fatalf("expected hintErrorMsg for nil service, got %T", msg)
	}
}

func TestFetchHintCmd_NilExercise_ReturnsError(t *testing.T) {
	hintSvc := &mockHintService{
		hint: &domain.Hint{Content: "pista"},
	}

	cmd := fetchHintCmd(hintSvc, nil, "FAIL output")
	msg := cmd()

	_, ok := msg.(hintErrorMsg)
	if !ok {
		t.Fatalf("expected hintErrorMsg for nil exercise, got %T", msg)
	}
}

func TestLoadProgressCmd_ReturnsProgressLoadedMsg(t *testing.T) {
	progressSvc := &mockProgressService{
		progress: map[string]*domain.Progress{
			"ex-1": {ExerciseID: "ex-1", Status: domain.ProgressCompleted},
		},
	}

	cmd := loadProgressCmd(progressSvc)
	msg := cmd()

	plMsg, ok := msg.(progressLoadedMsg)
	if !ok {
		t.Fatalf("expected progressLoadedMsg, got %T", msg)
	}
	if plMsg.err != nil {
		t.Errorf("unexpected error: %v", plMsg.err)
	}
	if plMsg.progress == nil {
		t.Fatal("progress should not be nil")
	}
	if len(plMsg.progress) != 1 {
		t.Errorf("progress count = %d, want 1", len(plMsg.progress))
	}
}

func TestLoadProgressCmd_Error_ReturnsProgressLoadedMsgWithError(t *testing.T) {
	progressSvc := &mockProgressService{
		err: errors.New("store error"),
	}

	cmd := loadProgressCmd(progressSvc)
	msg := cmd()

	plMsg, ok := msg.(progressLoadedMsg)
	if !ok {
		t.Fatalf("expected progressLoadedMsg, got %T", msg)
	}
	if plMsg.err == nil {
		t.Fatal("expected error but got nil")
	}
}

func TestLoadTopicExercisesCmd_ReturnsTopicExercisesLoadedMsg(t *testing.T) {
	exerciseSvc := &mockExerciseService{
		exercises: []*domain.ExerciseRef{
			{Slug: "ex-1", Title: "Hola Mundo", Difficulty: domain.DifficultyEasy},
			{Slug: "ex-2", Title: "Variables", Difficulty: domain.DifficultyMedium},
		},
	}

	cmd := loadTopicExercisesCmd(exerciseSvc, "variables")
	msg := cmd()

	teMsg, ok := msg.(topicExercisesLoadedMsg)
	if !ok {
		t.Fatalf("expected topicExercisesLoadedMsg, got %T", msg)
	}
	if teMsg.err != nil {
		t.Errorf("unexpected error: %v", teMsg.err)
	}
	if len(teMsg.exercises) != 2 {
		t.Errorf("exercises count = %d, want 2", len(teMsg.exercises))
	}
	if teMsg.exercises[0].Title != "Hola Mundo" {
		t.Errorf("first exercise title = %q, want %q", teMsg.exercises[0].Title, "Hola Mundo")
	}
}

func TestLoadTopicExercisesCmd_Error_ReturnsTopicExercisesLoadedMsgWithError(t *testing.T) {
	exerciseSvc := &mockExerciseService{
		err: errors.New("topic not found"),
	}

	cmd := loadTopicExercisesCmd(exerciseSvc, "inexistente")
	msg := cmd()

	teMsg, ok := msg.(topicExercisesLoadedMsg)
	if !ok {
		t.Fatalf("expected topicExercisesLoadedMsg, got %T", msg)
	}
	if teMsg.err == nil {
		t.Fatal("expected error but got nil")
	}
}

func TestProgressLoadedMsg_Struct(t *testing.T) {
	prog := map[string]*domain.Progress{
		"ex-1": {ExerciseID: "ex-1", Status: domain.ProgressCompleted},
	}
	msg := progressLoadedMsg{progress: prog, err: nil}
	if msg.progress["ex-1"].Status != domain.ProgressCompleted {
		t.Error("progress message should preserve status")
	}
}

func TestTopicExercisesLoadedMsg_Struct(t *testing.T) {
	exercises := []*domain.ExerciseRef{
		{Slug: "ex-1", Title: "Test", Difficulty: domain.DifficultyEasy},
	}
	msg := topicExercisesLoadedMsg{exercises: exercises, err: nil}
	if len(msg.exercises) != 1 {
		t.Error("should have 1 exercise")
	}
}
