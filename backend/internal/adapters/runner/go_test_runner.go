package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"godojo/internal/core/domain"
)

// GoTestRunner implements ports.TestRunner using os/exec to run go test -json.
type GoTestRunner struct {
	timeout time.Duration
}

// NewGoTestRunner creates a new GoTestRunner with a timeout for test execution.
// If timeout is <= 0, it defaults to 30 seconds.
func NewGoTestRunner(timeout time.Duration) *GoTestRunner {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &GoTestRunner{timeout: timeout}
}

// goTestEvent represents a single JSON line from go test -json output.
type goTestEvent struct {
	Time    *time.Time `json:"Time"`
	Action  string     `json:"Action"`
	Package string     `json:"Package"`
	Test    string     `json:"Test"`
	Output  string     `json:"Output"`
	Elapsed float64    `json:"Elapsed"`
}

// Run executes go test -json in the given directory and returns parsed results.
func (r *GoTestRunner) Run(ctx context.Context, exerciseDir string) (*domain.TestResult, error) {
	// Apply our internal timeout to the context
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "test", "-json", "./...")
	cmd.Dir = exerciseDir

	output, err := cmd.CombinedOutput()

	// Context cancellation (timeout or explicit cancel)
	if ctx.Err() != nil {
		return domain.NewTestResult(false,
			"⏱️ El test tardó demasiado tiempo y fue cancelado.\n"+string(output),
			r.timeout)
	}

	// Infrastructure errors: directory not found, go binary not found, etc.
	// err will be an *exec.ExitError for test failures, and *os.PathError or *exec.Error for infra issues.
	if err != nil {
		// Check if it's an exit error (test/compile failed) vs infrastructure error
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Check for missing/invalid module — this is an infrastructure error
			if strings.Contains(string(output), "cannot find main module") ||
				strings.Contains(string(output), "go.mod file not found") ||
				strings.Contains(string(output), "does not contain main module") ||
				strings.Contains(string(output), "[setup failed]") {
				return nil, fmt.Errorf("error al ejecutar go test: no se encontró go.mod (%w)", exitErr)
			}
			// Non-zero exit code: test failure or compile error — parse the output
			return r.ParseJSONOutput(string(output))
		}
		// Infrastructure error: return as error
		return nil, fmt.Errorf("error al ejecutar go test: %w", err)
	}

	return r.ParseJSONOutput(string(output))
}

// isCompileError detects if the output indicates a Go compilation failure rather than a test failure.
func isCompileError(raw string) bool {
	// Compile errors have lines starting with "# package" and contain build failure markers
	return strings.Contains(raw, "[build failed]") ||
		(strings.Contains(raw, "syntax error") && !strings.Contains(raw, `"Action":"fail"`))
}

// ParseJSONOutput parses go test -json output and builds a TestResult.
// This is exported for testing with golden files.
func (r *GoTestRunner) ParseJSONOutput(raw string) (*domain.TestResult, error) {
	start := time.Now()

	if raw == "" {
		return domain.NewTestResult(false,
			"⚠️ No se recibió salida de go test. El comando no produjo resultado (sin salida).",
			time.Since(start))
	}

	// If the output doesn't contain JSON lines, it's likely a compile error
	if !strings.Contains(raw, `"Action"`) {
		return domain.NewTestResult(false,
			fmt.Sprintf("🔴 El código no compila. Revisá los errores:\n\n%s", raw),
			time.Since(start))
	}

	var totalTests int
	var failedTests int
	var outputLines []string
	passed := true

	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event goTestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Not a valid JSON line — could be stderr from go tool itself
			outputLines = append(outputLines, line)
			continue
		}

		// Only count test-level events (events with a Test field)
		if event.Test == "" {
			// Package-level output — collect for summary IF it's meaningful
			if event.Output != "" {
				trimmed := strings.TrimRight(event.Output, "\n")
				// Filter out the "ok/fail  package" summary lines since we build our own
				if !strings.HasPrefix(trimmed, "ok  \t") && !strings.HasPrefix(trimmed, "FAIL\t") {
					outputLines = append(outputLines, trimmed)
				}
			}
			continue
		}

		switch event.Action {
		case "run":
			// Skip — we don't include per-test run lines in summary
		case "pass":
			totalTests++
		case "fail":
			totalTests++
			failedTests++
			passed = false
			if event.Output != "" {
				outputLines = append(outputLines, strings.TrimRight(event.Output, "\n"))
			}
		case "output":
			if event.Output != "" {
				outputLines = append(outputLines, strings.TrimRight(event.Output, "\n"))
			}
		case "skip":
			totalTests++
		}
	}

	// Detect compile errors: no test-level events AND output contains error markers
	if totalTests == 0 && isCompileError(raw) {
		return domain.NewTestResult(false,
			fmt.Sprintf("🔴 El código no compila. Revisá los errores:\n\n%s", strings.Join(outputLines, "\n")),
			time.Since(start))
	}

	if totalTests == 0 {
		// No test-level events found — tests may be cached or module has no tests
		passed = false
		outputLines = append(outputLines, "⚠️ No se detectaron tests individuales en la salida.")
	}

	fullOutput := strings.Join(outputLines, "\n")

	// Build a clear summary
	summary := fmt.Sprintf("Total: %d | Pasaron: %d | Fallaron: %d\n\n%s",
		totalTests, totalTests-failedTests, failedTests, fullOutput)

	return domain.NewTestResult(passed, summary, time.Since(start))
}
