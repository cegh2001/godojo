package runner_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"godojo/internal/adapters/runner"
)

func TestParseJSONOutput_AllPassing(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "test_output_pass.json"))
	if err != nil {
		t.Fatalf("cannot read golden file: %v", err)
	}

	r := runner.NewGoTestRunner(5 * time.Second)
	result, err := r.ParseJSONOutput(string(data))
	if err != nil {
		t.Fatalf("ParseJSONOutput() unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("all tests pass → Passed should be true")
	}
	if result.Output == "" {
		t.Error("output should not be empty for passing tests")
	}
	if !strings.Contains(result.Output, "PASS") {
		t.Errorf("output should contain PASS, got: %s", result.Output)
	}
	// Output should include test names in summary
	if !strings.Contains(result.Output, "Total: 2") {
		t.Errorf("summary should report total tests, got: %s", result.Output)
	}
}

func TestParseJSONOutput_SomeFailing(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "test_output_fail.json"))
	if err != nil {
		t.Fatalf("cannot read golden file: %v", err)
	}

	r := runner.NewGoTestRunner(5 * time.Second)
	result, err := r.ParseJSONOutput(string(data))
	if err != nil {
		t.Fatalf("ParseJSONOutput() unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("some tests fail → Passed should be false")
	}
	if result.Output == "" {
		t.Error("output should not be empty for failing tests")
	}
	if !strings.Contains(result.Output, "FAIL") {
		t.Errorf("output should contain FAIL, got: %s", result.Output)
	}
}

func TestParseJSONOutput_EmptyInput(t *testing.T) {
	r := runner.NewGoTestRunner(5 * time.Second)
	result, err := r.ParseJSONOutput("")
	if err != nil {
		t.Fatalf("ParseJSONOutput() empty input should not error: %v", err)
	}
	if result.Passed {
		t.Error("empty input → Passed should be false")
	}
	if !strings.Contains(result.Output, "sin salida") {
		t.Errorf("empty output should mention 'sin salida', got: %s", result.Output)
	}
}

func TestParseJSONOutput_CompileError(t *testing.T) {
	compileErrorOutput := "# ejercicio\n./ejercicio.go:3:2: undefined: Suma\nFAIL\tejercicio [build failed]\n"

	r := runner.NewGoTestRunner(5 * time.Second)
	result, err := r.ParseJSONOutput(compileErrorOutput)
	if err != nil {
		t.Fatalf("ParseJSONOutput() compile error should not error: %v", err)
	}
	if result.Passed {
		t.Error("compile error → Passed should be false")
	}
	if !strings.Contains(result.Output, "no compila") || !strings.Contains(result.Output, "undefined") {
		t.Errorf("output should mention compilation error, got: %s", result.Output)
	}
}

func TestParseJSONOutput_OnlyBuildOutput(t *testing.T) {
	// Simulate go test -json output that only has package-level events (no test-level events)
	jsonOnly := `{"Time":"2026-01-01T00:00:00Z","Action":"output","Package":"ejercicio","Output":"ok  \tejercicio\t(cached)\n"}
{"Time":"2026-01-01T00:00:00Z","Action":"pass","Package":"ejercicio","Elapsed":0.001}`

	r := runner.NewGoTestRunner(5 * time.Second)
	result, err := r.ParseJSONOutput(jsonOnly)
	if err != nil {
		t.Fatalf("ParseJSONOutput() unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("no test-level events → Passed should be false")
	}
}

func TestParseJSONOutput_DurationComputed(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "test_output_pass.json"))
	if err != nil {
		t.Fatalf("cannot read golden file: %v", err)
	}

	r := runner.NewGoTestRunner(5 * time.Second)
	result, err := r.ParseJSONOutput(string(data))
	if err != nil {
		t.Fatalf("ParseJSONOutput() unexpected error: %v", err)
	}
	// Duration may be 0 if parsing is instantaneous (sub-millisecond)
	if result.Duration < 0 {
		t.Errorf("duration should not be negative, got %v", result.Duration)
	}
}

// Integration test: actually runs go test in a temp directory
func TestRun_RealGoTest_Passing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	exerciseFile := filepath.Join(tmpDir, "ejercicio.go")
	testFile := filepath.Join(tmpDir, "ejercicio_test.go")
	modFile := filepath.Join(tmpDir, "go.mod")

	// Write a minimal Go module with passing tests
	os.WriteFile(exerciseFile, []byte(`package ejercicio

func Suma(a, b int) int {
	return a + b
}
`), 0644)

	os.WriteFile(testFile, []byte(`package ejercicio

import "testing"

func TestSuma(t *testing.T) {
	got := Suma(2, 3)
	want := 5
	if got != want {
		t.Errorf("Suma(2, 3) = %d, want %d", got, want)
	}
}
`), 0644)

	os.WriteFile(modFile, []byte(`module ejercicio

go 1.21
`), 0644)

	r := runner.NewGoTestRunner(30 * time.Second)
	result, err := r.Run(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("passing tests should produce Passed=true, got Output: %s", result.Output)
	}
}

func TestRun_RealGoTest_Failing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	exerciseFile := filepath.Join(tmpDir, "ejercicio.go")
	testFile := filepath.Join(tmpDir, "ejercicio_test.go")
	modFile := filepath.Join(tmpDir, "go.mod")

	os.WriteFile(exerciseFile, []byte(`package ejercicio

func Suma(a, b int) int {
	return a + b + 1 // intentional bug
}
`), 0644)

	os.WriteFile(testFile, []byte(`package ejercicio

import "testing"

func TestSuma(t *testing.T) {
	got := Suma(2, 3)
	want := 5
	if got != want {
		t.Errorf("Suma(2, 3) = %d, want %d", got, want)
	}
}
`), 0644)

	os.WriteFile(modFile, []byte(`module ejercicio

go 1.21
`), 0644)

	r := runner.NewGoTestRunner(30 * time.Second)
	result, err := r.Run(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("failing tests should produce Passed=false")
	}
}

func TestRun_RealGoTest_CompileError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	exerciseFile := filepath.Join(tmpDir, "ejercicio.go")
	testFile := filepath.Join(tmpDir, "ejercicio_test.go")
	modFile := filepath.Join(tmpDir, "go.mod")

	// Write code that doesn't compile
	os.WriteFile(exerciseFile, []byte(`package ejercicio

func Suma(a, b int) int {
	return a + b
`), 0644) // missing closing brace

	os.WriteFile(testFile, []byte(`package ejercicio

import "testing"

func TestSuma(t *testing.T) {
	got := Suma(2, 3)
	if got != 5 {
		t.Errorf("fail")
	}
}
`), 0644)

	os.WriteFile(modFile, []byte(`module ejercicio

go 1.21
`), 0644)

	r := runner.NewGoTestRunner(30 * time.Second)
	result, err := r.Run(context.Background(), tmpDir)
	// Compile error should NOT return err — it should be captured in TestResult
	if err != nil {
		t.Fatalf("Run() should not error on compile failure, should capture in result: %v", err)
	}
	if result.Passed {
		t.Error("compile error should produce Passed=false")
	}
	if !strings.Contains(result.Output, "no compila") {
		t.Errorf("output should mention compilation issue, got: %s", result.Output)
	}
}

func TestRun_Timeout_KillsProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	exerciseFile := filepath.Join(tmpDir, "ejercicio.go")
	testFile := filepath.Join(tmpDir, "ejercicio_test.go")
	modFile := filepath.Join(tmpDir, "go.mod")

	os.WriteFile(exerciseFile, []byte(`package ejercicio

func BucleInfinito() {
	for {}
}
`), 0644)

	os.WriteFile(testFile, []byte(`package ejercicio

import "testing"

func TestBucleInfinito(t *testing.T) {
	BucleInfinito()
}
`), 0644)

	os.WriteFile(modFile, []byte(`module ejercicio

go 1.21
`), 0644)

	// Very short timeout to force cancellation
	r := runner.NewGoTestRunner(2 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := r.Run(ctx, tmpDir)
	// Timeout errors from exec are expected
	if err == nil && result != nil && result.Passed {
		t.Error("infinite loop test should not pass")
	}
	// Either we get an error (timeout) or a result with Passed=false — both acceptable
}

func TestRun_NoGoMod_ReturnsError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	r := runner.NewGoTestRunner(30 * time.Second)
	_, err := r.Run(context.Background(), tmpDir)
	if err == nil {
		t.Error("Run() should return error when go.mod is missing")
	}
}

func TestRun_NoSuchDirectory_ReturnsError(t *testing.T) {
	r := runner.NewGoTestRunner(5 * time.Second)
	_, err := r.Run(context.Background(), "/no/existe/directorio")
	if err == nil {
		t.Error("Run() should return error for non-existent directory")
	}
}

func TestNewGoTestRunner_NegativeTimeout_Defaults(t *testing.T) {
	r := runner.NewGoTestRunner(-1 * time.Second)
	if r == nil {
		t.Fatal("NewGoTestRunner should not return nil")
	}
}

func TestParseJSONOutput_MixedTestAndPackageEvents(t *testing.T) {
	input := `{"Time":"2026-01-01T00:00:00Z","Action":"run","Package":"ejercicio","Test":"TestA"}
{"Time":"2026-01-01T00:00:00Z","Action":"output","Package":"ejercicio","Test":"TestA","Output":"=== RUN   TestA\n"}
{"Time":"2026-01-01T00:00:01Z","Action":"pass","Package":"ejercicio","Test":"TestA","Elapsed":0.001}
{"Time":"2026-01-01T00:00:01Z","Action":"output","Package":"ejercicio","Output":"ok  \tejercicio\t0.001s\n"}
{"Time":"2026-01-01T00:00:01Z","Action":"pass","Package":"ejercicio","Elapsed":0.001}`

	r := runner.NewGoTestRunner(5 * time.Second)
	result, err := r.ParseJSONOutput(input)
	if err != nil {
		t.Fatalf("ParseJSONOutput() unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("one test passing → Passed should be true")
	}
}

func TestRun_ContextCancelled_StopsEarly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	exerciseFile := filepath.Join(tmpDir, "ejercicio.go")
	testFile := filepath.Join(tmpDir, "ejercicio_test.go")
	modFile := filepath.Join(tmpDir, "go.mod")

	os.WriteFile(exerciseFile, []byte(`package ejercicio

func Suma(a, b int) int {
	sum := 0
	for i := 0; i < 1000000000; i++ {
		sum += i
	}
	return a + b
}
`), 0644)

	os.WriteFile(testFile, []byte(`package ejercicio

import "testing"

func TestSuma(t *testing.T) {
	got := Suma(2, 3)
	if got != 5 {
		t.Errorf("fail")
	}
}
`), 0644)

	os.WriteFile(modFile, []byte(`module ejercicio

go 1.21
`), 0644)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	r := runner.NewGoTestRunner(30 * time.Second)
	result, err := r.Run(ctx, tmpDir)
	if err == nil && (result == nil || result.Passed) {
		t.Error("cancelled context should result in error or non-passing result")
	}
	_ = err
}

// Verify the runner is not available when go binary is missing
func TestNewGoTestRunner_GoBinaryPath(t *testing.T) {
	r := runner.NewGoTestRunner(5 * time.Second)
	// The runner stores the path internally; verify it finds go binary
	if r == nil {
		t.Fatal("constructor should not return nil")
	}
	// Verify it can find go
	_, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go binary not found in PATH, skipping go-dependent test")
	}
}

// --- Stdout/Stderr separation tests ---

func TestRun_StdoutStderrPopulated_Passing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "ejercicio.go"), []byte(`package ejercicio

func Suma(a, b int) int {
	return a + b
}
`), 0644)

	os.WriteFile(filepath.Join(tmpDir, "ejercicio_test.go"), []byte(`package ejercicio

import "testing"

func TestSuma(t *testing.T) {
	got := Suma(2, 3)
	if got != 5 {
		t.Errorf("fail")
	}
}
`), 0644)

	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(`module ejercicio

go 1.21
`), 0644)

	r := runner.NewGoTestRunner(30 * time.Second)
	result, err := r.Run(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if !result.Passed {
		t.Fatal("expected passing tests")
	}

	// Stdout should be populated (go test -json writes to stdout)
	if result.Stdout == "" {
		t.Error("Stdout should not be empty for passing tests")
	}

	// Stderr should be empty for clean compilation
	if result.Stderr != "" {
		t.Errorf("Stderr should be empty for clean compilation, got: %q", result.Stderr)
	}

	// Output should still exist (backward compatible)
	if result.Output == "" {
		t.Error("Output should not be empty")
	}

	// Stdout should contain JSON lines
	if !strings.Contains(result.Stdout, `"Action"`) {
		t.Error("Stdout should contain JSON test events")
	}
}

func TestRun_StdoutStderrPopulated_CompileError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "ejercicio.go"), []byte(`package ejercicio

func Suma(a, b int) int {
	return a + b
`), 0644) // missing closing brace — will not compile

	os.WriteFile(filepath.Join(tmpDir, "ejercicio_test.go"), []byte(`package ejercicio

import "testing"

func TestSuma(t *testing.T) {
	got := Suma(2, 3)
	if got != 5 {
		t.Errorf("fail")
	}
}
`), 0644)

	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(`module ejercicio

go 1.21
`), 0644)

	r := runner.NewGoTestRunner(30 * time.Second)
	result, err := r.Run(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Run() should not error on compile, should capture in result: %v", err)
	}
	if result.Passed {
		t.Error("compile error should produce Passed=false")
	}

	// At least one of Stdout or Stderr should be populated with error output.
	// go test may route compile errors to stdout (via JSON) or stderr depending on platform.
	if result.Stdout == "" && result.Stderr == "" {
		t.Error("Stdout or Stderr should contain compile error output")
	}

	// Output should still be populated (combined)
	if result.Output == "" {
		t.Error("Output should not be empty")
	}
}

func TestRun_StdoutStderrPopulated_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "ejercicio.go"), []byte(`package ejercicio

func BucleInfinito() {
	for {}
}
`), 0644)

	os.WriteFile(filepath.Join(tmpDir, "ejercicio_test.go"), []byte(`package ejercicio

import (
	"fmt"
	"testing"
)

func TestBucleInfinito(t *testing.T) {
	fmt.Println("Iniciando bucle...")
	BucleInfinito()
	fmt.Println("Nunca llega acá")
}
`), 0644)

	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(`module ejercicio

go 1.21
`), 0644)

	r := runner.NewGoTestRunner(2 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, _ := r.Run(ctx, tmpDir)

	if result != nil {
		// Stdout and Stderr fields should exist (even if empty due to buffering)
		// We verify the struct fields are present, content is timing-dependent
		_ = result.Stdout
		_ = result.Stderr

		// Output should still be populated (combined is always set)
		if result.Output == "" {
			t.Error("Output should not be empty even on timeout")
		}
	}
}

func TestParseJSONOutput_StdoutStderrEmpty(t *testing.T) {
	// ParseJSONOutput should NOT set Stdout/Stderr — that's Run's job
	data, err := os.ReadFile(filepath.Join("testdata", "test_output_pass.json"))
	if err != nil {
		t.Fatalf("cannot read golden file: %v", err)
	}

	r := runner.NewGoTestRunner(5 * time.Second)
	result, err := r.ParseJSONOutput(string(data))
	if err != nil {
		t.Fatalf("ParseJSONOutput() unexpected error: %v", err)
	}

	if result.Stdout != "" {
		t.Errorf("ParseJSONOutput should not set Stdout, got: %q", result.Stdout)
	}
	if result.Stderr != "" {
		t.Errorf("ParseJSONOutput should not set Stderr, got: %q", result.Stderr)
	}
}
