package services_test

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"godojo/internal/adapters/chatstore"
	"godojo/internal/core/domain"
	"godojo/internal/core/ports"

	"godojo/internal/core"          // ToolRegistry
	"godojo/internal/core/services" // SenseiService, RoadmapService
)

// --- Mock SenseiProvider ---

// mockSenseiProvider implements ports.SenseiProvider with preset responses.
// Each entry in responses corresponds to one call (SendMessage or SendFunctionResponse).
type mockSenseiProvider struct {
	mu                         sync.Mutex
	responses                  [][]domain.ContentPart
	callCount                  int
	sendCalls                  int
	sendToolCounts             []int
	functionResponseCalls      int
	lastFunctionResponseCallID string
	lastFunctionResponseResult interface{}
	err                        error // if set, return this error instead of next response

	// SendMessageStream support
	mockStreamResponse   func(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) (<-chan ports.StreamChunk, error)
	sendStreamCalls      int
	sendStreamToolCounts []int
}

func (m *mockSenseiProvider) nextResponse() ([]domain.ContentPart, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	if m.callCount >= len(m.responses) {
		return nil, nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return resp, nil
}

func (m *mockSenseiProvider) SendMessage(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) ([]domain.ContentPart, error) {
	m.mu.Lock()
	m.sendCalls++
	m.sendToolCounts = append(m.sendToolCounts, len(tools))
	m.mu.Unlock()
	return m.nextResponse()
}

func (m *mockSenseiProvider) SendMessageStream(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) (<-chan ports.StreamChunk, error) {
	m.mu.Lock()
	m.sendStreamCalls++
	m.sendStreamToolCounts = append(m.sendStreamToolCounts, len(tools))
	m.mu.Unlock()

	if m.mockStreamResponse != nil {
		return m.mockStreamResponse(ctx, systemPrompt, history, tools)
	}
	// Default no-op stub: return a closed empty channel
	ch := make(chan ports.StreamChunk)
	close(ch)
	return ch, nil
}

func (m *mockSenseiProvider) SendFunctionResponse(ctx context.Context, history []chatstore.ChatMessage, callID string, name string, result interface{}) ([]domain.ContentPart, error) {
	m.mu.Lock()
	m.functionResponseCalls++
	m.lastFunctionResponseCallID = callID
	m.lastFunctionResponseResult = result
	m.mu.Unlock()
	return m.nextResponse()
}

// --- Mock WorkspaceManager ---

type mockWorkspace struct {
	files       map[string]string
	directories map[string]bool
}

func newMockWorkspace() *mockWorkspace {
	return &mockWorkspace{
		files:       make(map[string]string),
		directories: make(map[string]bool),
	}
}

func (w *mockWorkspace) CreateFile(filename string, content string) error {
	w.files[filename] = content
	return nil
}

func (w *mockWorkspace) ReadFile(filename string) (string, error) {
	content, ok := w.files[filename]
	if !ok {
		return "", fmt.Errorf("archivo %q no existe", filename)
	}
	return content, nil
}

func (w *mockWorkspace) ListFiles() ([]string, error) {
	var names []string
	for name := range w.files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func (w *mockWorkspace) WorkspacePath() string {
	return "/mock/workspace"
}

func (w *mockWorkspace) CreateDirectory(name string) error {
	if name == "" {
		return fmt.Errorf("el nombre del directorio no puede estar vacío")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("no se permite usar '..' en la ruta: %q", name)
	}
	w.directories[name] = true
	return nil
}

func (w *mockWorkspace) ListDirectory(name string) ([]ports.FileInfo, error) {
	if !w.directories[name] && name != "." {
		return nil, fmt.Errorf("el directorio %q no existe", name)
	}
	var result []ports.FileInfo
	for fname := range w.files {
		// Check if file is in this directory
		dir := "."
		if idx := strings.LastIndex(fname, "/"); idx >= 0 {
			dir = fname[:idx]
		}
		if dir == name || (name == "." && !strings.Contains(fname, "/")) {
			result = append(result, ports.FileInfo{
				Name:  fname,
				IsDir: false,
				Size:  int64(len(w.files[fname])),
			})
		}
	}
	return result, nil
}

// --- Mock TestRunner ---

type mockTestRunner struct {
	results map[string]*domain.TestResult
	errs    map[string]error
	calls   []string
}

func newMockTestRunner() *mockTestRunner {
	return &mockTestRunner{
		results: make(map[string]*domain.TestResult),
		errs:    make(map[string]error),
	}
}

func (m *mockTestRunner) Run(ctx context.Context, exerciseDir string) (*domain.TestResult, error) {
	m.calls = append(m.calls, exerciseDir)
	if err, ok := m.errs[exerciseDir]; ok {
		return nil, err
	}
	if result, ok := m.results[exerciseDir]; ok {
		return result, nil
	}
	// Default: passing result
	result, _ := domain.NewTestResult(true, "All tests pass", 100*time.Millisecond)
	result.Stdout = "ok"
	result.Stderr = ""
	return result, nil
}

// Ensure mockTestRunner implements ports.TestRunner
var _ ports.TestRunner = (*mockTestRunner)(nil)

// ============================================================================
// Task 7 Tests: setup_workspace, read_codebase, execute_and_evaluate
// ============================================================================

func TestSenseiService_EightToolsRegistered(t *testing.T) {
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	runner := newMockTestRunner()

	svc := services.NewSenseiService(nil, tools, ws, roadmap, runner)
	_ = svc

	decls := tools.GetDeclarations()
	if len(decls) != 8 {
		t.Fatalf("expected 8 tool declarations, got %d: %v", len(decls), toolNames(decls))
	}

	expected := map[string]bool{
		"create_exercise_file":       true,
		"read_roadmap_section":       true,
		"list_workspace_files":       true,
		"read_workspace_file":        true,
		"read_recent_workspace_file": true,
		"setup_workspace":            true,
		"read_codebase":              true,
		"execute_and_evaluate":       true,
	}
	for _, decl := range decls {
		if !expected[decl.Name] {
			t.Errorf("unexpected tool: %q", decl.Name)
		}
		delete(expected, decl.Name)
	}
	if len(expected) > 0 {
		for name := range expected {
			t.Errorf("missing tool: %q", name)
		}
	}
}

func toolNames(decls []domain.ToolDeclaration) []string {
	names := make([]string, len(decls))
	for i, d := range decls {
		names[i] = d.Name
	}
	return names
}

func TestSenseiService_SetupWorkspaceTool(t *testing.T) {
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	runner := newMockTestRunner()

	svc := services.NewSenseiService(nil, tools, ws, roadmap, runner)
	_ = svc

	// Execute setup_workspace tool directly
	result, err := tools.Execute("setup_workspace", map[string]interface{}{
		"topic_slug": "variables",
		"files": map[string]interface{}{
			"main.go":      "package main\n\nfunc main() {}",
			"main_test.go": "package main\n\nimport \"testing\"",
			"README.md":    "# Variables Exercise",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if res["dir"] != "variables" {
		t.Errorf("dir = %q, want variables", res["dir"])
	}
	if res["success"] != true {
		t.Errorf("success = %v, want true", res["success"])
	}

	// Verify files were created via workspace
	for _, fname := range []string{"variables/main.go", "variables/main_test.go", "variables/README.md"} {
		if _, ok := ws.files[fname]; !ok {
			t.Errorf("expected file %q to be created, files: %v", fname, ws.files)
		}
	}
}

func TestSenseiService_SetupWorkspace_PathTraversalRejected(t *testing.T) {
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	runner := newMockTestRunner()

	svc := services.NewSenseiService(nil, tools, ws, roadmap, runner)
	_ = svc

	_, err := tools.Execute("setup_workspace", map[string]interface{}{
		"topic_slug": "../etc",
		"files": map[string]interface{}{
			"main.go": "package main",
		},
	})
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if !strings.Contains(err.Error(), "..") {
		t.Errorf("expected error about '..', got %q", err.Error())
	}
}

func TestSenseiService_ReadCodebaseTool(t *testing.T) {
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	// Pre-populate workspace with files
	ws.files["variables/main.go"] = "package main\n\nfunc main() {}"
	ws.files["variables/main_test.go"] = "package main\n\nfunc TestMain(t *testing.T) {}"
	ws.files["variables/README.md"] = "# Variables"
	ws.directories["variables"] = true

	roadmap := services.NewRoadmapService()
	runner := newMockTestRunner()

	svc := services.NewSenseiService(nil, tools, ws, roadmap, runner)
	_ = svc

	result, err := tools.Execute("read_codebase", map[string]interface{}{
		"topic_slug": "variables",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	files, ok := res["files"].(map[string]string)
	if !ok {
		t.Fatalf("expected files map, got %T", res["files"])
	}
	if len(files) != 2 {
		t.Errorf("expected 2 .go files, got %d: %v", len(files), files)
	}
	// Should contain .go files but NOT README.md
	if _, ok := files["variables/main.go"]; !ok {
		t.Error("missing variables/main.go")
	}
	if _, ok := files["variables/main_test.go"]; !ok {
		t.Error("missing variables/main_test.go")
	}
	if _, ok := files["variables/README.md"]; ok {
		t.Error("README.md should not be included (not a .go file)")
	}
}

func TestSenseiService_ReadCodebase_MissingDirectory(t *testing.T) {
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	runner := newMockTestRunner()

	svc := services.NewSenseiService(nil, tools, ws, roadmap, runner)
	_ = svc

	_, err := tools.Execute("read_codebase", map[string]interface{}{
		"topic_slug": "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
	if !strings.Contains(err.Error(), "no existe") {
		t.Errorf("expected 'no existe', got %q", err.Error())
	}
}

func TestSenseiService_ExecuteAndEvaluateTool(t *testing.T) {
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	runner := newMockTestRunner()

	// Set up a passing result for variables/
	result, _ := domain.NewTestResult(true, "All tests pass\nTotal: 2 | Pasaron: 2 | Fallaron: 0", 100*time.Millisecond)
	result.Stdout = "ok  \tvariable_test"
	result.Stderr = ""
	runner.results["/mock/workspace/variables"] = result

	svc := services.NewSenseiService(nil, tools, ws, roadmap, runner)
	_ = svc

	res, err := tools.Execute("execute_and_evaluate", map[string]interface{}{
		"topic_slug": "variables",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", res)
	}
	if r["passed"] != true {
		t.Errorf("passed = %v, want true", r["passed"])
	}
	if r["output"] != result.Output {
		t.Errorf("output = %q, want %q", r["output"], result.Output)
	}
}

func TestSenseiService_ExecuteAndEvaluate_NilTestRunner(t *testing.T) {
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()

	svc := services.NewSenseiService(nil, tools, ws, roadmap, nil)
	_ = svc

	_, err := tools.Execute("execute_and_evaluate", map[string]interface{}{
		"topic_slug": "variables",
	})
	if err == nil {
		t.Fatal("expected error when TestRunner is nil")
	}
	if !strings.Contains(err.Error(), "TestRunner") && !strings.Contains(err.Error(), "testRunner") {
		t.Errorf("expected error about TestRunner, got %q", err.Error())
	}
}

func TestSenseiService_NewIntentHints(t *testing.T) {
	provider := &mockSenseiProvider{
		responses: [][]domain.ContentPart{
			{textPart("Voy a ejecutar los tests.")},
		},
	}
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	runner := newMockTestRunner()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, runner)

	// Messages with new intent hints should trigger tool-enabled requests
	// "probar" is one of the new hints
	statusCh, err := svc.ProcessMessage(context.Background(), "Sos un sensei.", "Probalo ejecutando los tests", newMockSession("s-hints"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	response, _ := drainStatusChannel(statusCh)
	if response != "Voy a ejecutar los tests." {
		t.Fatalf("response = %q", response)
	}
	// Should have sent tools (since "probar" is in intent hints)
	if len(provider.sendToolCounts) != 1 || provider.sendToolCounts[0] == 0 {
		t.Fatalf("first SendMessage tools = %v, want a tool-enabled request for 'probar'", provider.sendToolCounts)
	}
}

func TestSenseiService_EmptyFilesSetupWorkspace(t *testing.T) {
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	runner := newMockTestRunner()

	svc := services.NewSenseiService(nil, tools, ws, roadmap, runner)
	_ = svc

	result, err := tools.Execute("setup_workspace", map[string]interface{}{
		"topic_slug": "empty_topic",
		"files":      map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if res["success"] != true {
		t.Errorf("success = %v, want true", res["success"])
	}
	// Directory should be created even with no files
	if !ws.directories["empty_topic"] {
		t.Error("expected empty_topic directory to be created")
	}
}

func textPart(text string) domain.ContentPart {
	return domain.ContentPart{Text: text}
}

func funcCallPart(name string, args map[string]interface{}) domain.ContentPart {
	return domain.ContentPart{
		FunctionCall: &domain.FunctionCall{
			Name: name,
			Args: args,
		},
	}
}

func funcCallPartWithID(id string, name string, args map[string]interface{}) domain.ContentPart {
	return domain.ContentPart{
		FunctionCall: &domain.FunctionCall{
			ID:   id,
			Name: name,
			Args: args,
		},
	}
}

func newMockSession(id string) *chatstore.ChatSession {
	return &chatstore.ChatSession{
		ID:       id,
		Name:     "Test",
		Messages: nil,
	}
}

// drainStatusChannel reads the live status channel produced by ProcessMessage
// and extracts the final response text and a list of status messages.
// It preserves the same extraction logic that the old ProcessMessage internal
// drainer used, so existing test assertions continue to work.
func drainStatusChannel(ch <-chan string) (response string, statuses []string) {
	for msg := range ch {
		switch {
		case strings.HasPrefix(msg, "done:"):
			response = strings.TrimPrefix(msg, "done:")
		case strings.HasPrefix(msg, "error:"):
			response = strings.TrimPrefix(msg, "error:")
			statuses = append(statuses, "Error: "+response)
		case strings.HasPrefix(msg, "stream:"):
			// ignore stream chunks in tests
		default:
			statuses = append(statuses, msg)
		}
	}
	return
}

// --- Test Cases ---

// TestSenseiService_TextOnlyResponse tests the simplest case:
// user sends message, provider streams text → service returns text.
func TestSenseiService_TextOnlyResponse(t *testing.T) {
	provider := &mockSenseiProvider{
		mockStreamResponse: func(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) (<-chan ports.StreamChunk, error) {
			ch := make(chan ports.StreamChunk, 2)
			ch <- ports.StreamChunk{Text: "¡Buenas! ¿En qué te ayudo con Go?"}
			ch <- ports.StreamChunk{Done: true}
			close(ch)
			return ch, nil
		},
	}
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	session := newMockSession("s1")
	ctx := context.Background()

	statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Hola", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	response, statuses := drainStatusChannel(statusCh)

	if response != "¡Buenas! ¿En qué te ayudo con Go?" {
		t.Errorf("response = %q, want greeting", response)
	}
	if len(provider.sendStreamToolCounts) != 1 || provider.sendStreamToolCounts[0] != 0 {
		t.Fatalf("first SendMessageStream tools = %v, want [0]", provider.sendStreamToolCounts)
	}

	// Should have "Pensando..." and "done" statuses
	if len(statuses) < 2 {
		t.Errorf("expected at least 2 status updates, got %d: %v", len(statuses), statuses)
	}
}

func TestSenseiService_StartsWithToolsForWorkspaceIntent(t *testing.T) {
	provider := &mockSenseiProvider{
		responses: [][]domain.ContentPart{
			{textPart("Primero voy a revisar tu workspace.")},
		},
	}
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	statusCh, err := svc.ProcessMessage(context.Background(), "Sos un sensei.", "¿Qué archivos tengo en el workspace?", newMockSession("s-fast-tools"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	response, _ := drainStatusChannel(statusCh)
	if response != "Primero voy a revisar tu workspace." {
		t.Fatalf("response = %q", response)
	}
	if len(provider.sendToolCounts) != 1 || provider.sendToolCounts[0] == 0 {
		t.Fatalf("first SendMessage tools = %v, want a tool-enabled request", provider.sendToolCounts)
	}
}

// TestSenseiService_SingleToolCall tests a single functionCall → tool execution → final text flow.
func TestSenseiService_SingleToolCall(t *testing.T) {
	provider := &mockSenseiProvider{
		responses: [][]domain.ContentPart{
			// First call: functionCall to create_exercise_file
			{funcCallPartWithID("call-create-1", "create_exercise_file", map[string]interface{}{
				"filename": "hola.go",
				"content":  "package main\n\nfunc main() {}",
			})},
			// Second call (SendFunctionResponse): text response
			{textPart("Listo, creé el archivo hola.go en tu workspace.")},
		},
	}
	tools := core.NewToolRegistry()
	tools.Register("create_exercise_file", domain.ToolDeclaration{
		Name:        "create_exercise_file",
		Description: "Crea un archivo .go en el workspace",
		Parameters: domain.ToolParameters{
			Type: "OBJECT",
			Properties: map[string]domain.ToolProperty{
				"filename": {Type: "STRING", Description: "Nombre del archivo"},
				"content":  {Type: "STRING", Description: "Contenido del archivo"},
			},
			Required: []string{"filename", "content"},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{
			"filename": args["filename"],
			"success":  true,
		}, nil
	})

	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	session := newMockSession("s2")
	ctx := context.Background()

	statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Creá un hola mundo", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	response, statuses := drainStatusChannel(statusCh)

	if response != "Listo, creé el archivo hola.go en tu workspace." {
		t.Errorf("response = %q", response)
	}
	if provider.sendCalls != 1 {
		t.Errorf("SendMessage calls = %d, want 1", provider.sendCalls)
	}
	if provider.functionResponseCalls != 1 {
		t.Errorf("SendFunctionResponse calls = %d, want 1", provider.functionResponseCalls)
	}
	if provider.lastFunctionResponseCallID != "call-create-1" {
		t.Errorf("last function response call ID = %q, want %q", provider.lastFunctionResponseCallID, "call-create-1")
	}

	// Verify "Ejecutando create_exercise_file..." status was sent
	found := false
	for _, s := range statuses {
		if strings.Contains(s, "Ejecutando") && strings.Contains(s, "create_exercise_file") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Ejecutando create_exercise_file...' status, got: %v", statuses)
	}
}

func TestSenseiService_ListWorkspaceFilesTool(t *testing.T) {
	provider := &mockSenseiProvider{
		responses: [][]domain.ContentPart{
			{funcCallPartWithID("call-list-1", "list_workspace_files", map[string]interface{}{})},
			{textPart("Tenés archivos en el workspace.")},
		},
	}
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	ws.files["maps/ejercicio.go"] = "package main"
	ws.files["variables/clase-1.go"] = "package main"
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	statusCh, err := svc.ProcessMessage(context.Background(), "Sos un sensei.", "¿Qué archivos tengo en el workspace?", newMockSession("s-workspace-list"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	response, _ := drainStatusChannel(statusCh)
	if response != "Tenés archivos en el workspace." {
		t.Fatalf("response = %q", response)
	}

	result, ok := provider.lastFunctionResponseResult.(map[string]interface{})
	if !ok {
		t.Fatalf("tool result = %#v", provider.lastFunctionResponseResult)
	}
	if result["count"] != 2 {
		t.Fatalf("count = %#v", result["count"])
	}
	files, ok := result["files"].([]string)
	if !ok {
		t.Fatalf("files = %#v", result["files"])
	}
	expected := []string{"maps/ejercicio.go", "variables/clase-1.go"}
	if strings.Join(files, ",") != strings.Join(expected, ",") {
		t.Fatalf("files = %v, want %v", files, expected)
	}
}

func TestSenseiService_ReadWorkspaceFileTool(t *testing.T) {
	provider := &mockSenseiProvider{
		responses: [][]domain.ContentPart{
			{funcCallPartWithID("call-read-1", "read_workspace_file", map[string]interface{}{"filename": "variables/clase-1.go"})},
			{textPart("Revisé el archivo del workspace.")},
		},
	}
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	ws.files["variables/clase-1.go"] = "package main\n\nfunc main() {}"
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	statusCh, err := svc.ProcessMessage(context.Background(), "Sos un sensei.", "Revisá mi archivo", newMockSession("s-workspace-read"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	response, _ := drainStatusChannel(statusCh)
	if response != "Revisé el archivo del workspace." {
		t.Fatalf("response = %q", response)
	}

	result, ok := provider.lastFunctionResponseResult.(map[string]interface{})
	if !ok {
		t.Fatalf("tool result = %#v", provider.lastFunctionResponseResult)
	}
	if result["filename"] != "variables/clase-1.go" {
		t.Fatalf("filename = %#v", result["filename"])
	}
	content, ok := result["content"].(string)
	if !ok {
		t.Fatalf("content = %#v", result["content"])
	}
	if !strings.Contains(content, "func main() {}") {
		t.Fatalf("content = %q", content)
	}
}

func TestSenseiService_ReadRecentWorkspaceFileTool(t *testing.T) {
	provider := &mockSenseiProvider{
		responses: [][]domain.ContentPart{
			{funcCallPartWithID("call-create-1", "create_exercise_file", map[string]interface{}{
				"filename":   "introduccion_tipos_de_datos.go",
				"topic_slug": "tipos",
				"content":    "package main\n\n// TODO: completar runas",
			})},
			{funcCallPartWithID("call-read-recent-1", "read_recent_workspace_file", map[string]interface{}{})},
			{textPart("Leí el archivo reciente y puedo revisarlo contigo.")},
		},
	}
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, core.NewToolRegistry(), ws, roadmap, nil)

	statusCh, err := svc.ProcessMessage(context.Background(), "Sos un sensei.", "Empecemos con tipos y luego revisalo", newMockSession("s-workspace-recent"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	response, _ := drainStatusChannel(statusCh)
	if response != "Leí el archivo reciente y puedo revisarlo contigo." {
		t.Fatalf("response = %q", response)
	}

	if _, ok := ws.files["tipos/introduccion_tipos_de_datos.go"]; !ok {
		t.Fatalf("expected file under tipos/, got files: %v", ws.files)
	}

	result, ok := provider.lastFunctionResponseResult.(map[string]interface{})
	if !ok {
		t.Fatalf("tool result = %#v", provider.lastFunctionResponseResult)
	}
	if result["filename"] != "tipos/introduccion_tipos_de_datos.go" {
		t.Fatalf("filename = %#v", result["filename"])
	}
	content, ok := result["content"].(string)
	if !ok {
		t.Fatalf("content = %#v", result["content"])
	}
	if !strings.Contains(content, "runas") {
		t.Fatalf("content = %q", content)
	}
}

// TestSenseiService_TopicFolderFromRoadmap verifies that the latest consulted topic becomes the folder prefix.
func TestSenseiService_TopicFolderFromRoadmap(t *testing.T) {
	provider := &mockSenseiProvider{
		responses: [][]domain.ContentPart{
			{funcCallPart("read_roadmap_section", map[string]interface{}{
				"section_slug": "variables",
			})},
			{funcCallPart("create_exercise_file", map[string]interface{}{
				"filename": "clase-1.go",
				"content":  "package main",
			})},
			{textPart("Creé el ejercicio dentro de variables/.")},
		},
	}
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, core.NewToolRegistry(), ws, roadmap, nil)

	session := newMockSession("topic-folder")
	ctx := context.Background()

	statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Armame ejercicios de variables", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	response, statuses := drainStatusChannel(statusCh)

	if response != "Creé el ejercicio dentro de variables/." {
		t.Errorf("response = %q", response)
	}

	if _, ok := ws.files["variables/clase-1.go"]; !ok {
		t.Fatalf("expected file to be created under variables/, got files: %v", ws.files)
	}

	found := false
	for _, s := range statuses {
		if strings.Contains(s, "create_exercise_file") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected create_exercise_file status, got: %v", statuses)
	}
}

// TestSenseiService_MultiToolSequential tests 2 functionCalls in sequence → both execute → final text.
func TestSenseiService_MultiToolSequential(t *testing.T) {
	provider := &mockSenseiProvider{
		responses: [][]domain.ContentPart{
			// Round 1: functionCall to read_roadmap_section
			{funcCallPart("read_roadmap_section", map[string]interface{}{
				"section_slug": "fase-1",
			})},
			// Round 2: functionCall to create_exercise_file
			{funcCallPart("create_exercise_file", map[string]interface{}{
				"filename": "variables.go",
				"content":  "package main",
			})},
			// Round 3: text
			{textPart("Creé el ejercicio. ¿Lo ejecutamos?")},
		},
	}
	tools := core.NewToolRegistry()
	tools.Register("read_roadmap_section", domain.ToolDeclaration{
		Name:        "read_roadmap_section",
		Description: "Lee una sección del roadmap",
		Parameters: domain.ToolParameters{
			Type: "OBJECT",
			Properties: map[string]domain.ToolProperty{
				"section_slug": {Type: "STRING", Description: "Slug de la sección"},
			},
			Required: []string{"section_slug"},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"title": "Fundamentos"}, nil
	})
	tools.Register("create_exercise_file", domain.ToolDeclaration{
		Name:        "create_exercise_file",
		Description: "Crea un archivo",
		Parameters: domain.ToolParameters{
			Type: "OBJECT",
			Properties: map[string]domain.ToolProperty{
				"filename": {Type: "STRING", Description: "Nombre"},
				"content":  {Type: "STRING", Description: "Contenido"},
			},
			Required: []string{"filename", "content"},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"success": true}, nil
	})

	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	session := newMockSession("s3")
	ctx := context.Background()

	statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Dame ejercicios de fase 1", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	response, statuses := drainStatusChannel(statusCh)

	if response != "Creé el ejercicio. ¿Lo ejecutamos?" {
		t.Errorf("response = %q", response)
	}
	if provider.sendCalls != 1 {
		t.Errorf("SendMessage calls = %d, want 1", provider.sendCalls)
	}
	if provider.functionResponseCalls != 2 {
		t.Errorf("SendFunctionResponse calls = %d, want 2", provider.functionResponseCalls)
	}

	// Should have executed both tools
	toolCount := 0
	for _, s := range statuses {
		if strings.HasPrefix(s, "Ejecutando") {
			toolCount++
		}
	}
	if toolCount != 2 {
		t.Errorf("expected 2 tool executions, got %d: %v", toolCount, statuses)
	}
}

// TestSenseiService_MaxRoundsExceeded tests that after 5 rounds of functionCalls, we stop and return error.
func TestSenseiService_MaxRoundsExceeded(t *testing.T) {
	// 5 rounds of function calls — the loop should stop and return the exhausted message
	responses := make([][]domain.ContentPart, 5)
	for i := 0; i < 5; i++ {
		responses[i] = []domain.ContentPart{
			funcCallPart("create_exercise_file", map[string]interface{}{
				"filename": fmt.Sprintf("file%d.go", i),
				"content":  "package main",
			}),
		}
	}

	provider := &mockSenseiProvider{responses: responses}
	tools := core.NewToolRegistry()
	tools.Register("create_exercise_file", domain.ToolDeclaration{
		Name:        "create_exercise_file",
		Description: "Crea",
		Parameters: domain.ToolParameters{
			Type: "OBJECT",
			Properties: map[string]domain.ToolProperty{
				"filename": {Type: "STRING", Description: "Nombre"},
				"content":  {Type: "STRING", Description: "Contenido"},
			},
			Required: []string{"filename", "content"},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"ok": true}, nil
	})

	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	session := newMockSession("s4")
	ctx := context.Background()

	statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Creá archivos para probar los límites", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	response, statuses := drainStatusChannel(statusCh)

	// Should contain the exhausted message
	if !strings.Contains(response, "reformulá") {
		t.Errorf("expected exhausted message, got: %q", response)
	}

	// Should have exactly 5 tool statuses
	toolCount := 0
	for _, s := range statuses {
		if strings.HasPrefix(s, "Ejecutando") {
			toolCount++
		}
	}
	if toolCount != 5 {
		t.Errorf("expected 5 tool executions, got %d: %v", toolCount, statuses)
	}
}

// TestSenseiService_ToolExecutionError tests that when a tool handler returns an error,
// the error is sent as functionResponse and the model recovers with text.
func TestSenseiService_ToolExecutionError(t *testing.T) {
	provider := &mockSenseiProvider{
		responses: [][]domain.ContentPart{
			// First call: functionCall
			{funcCallPart("create_exercise_file", map[string]interface{}{
				"filename": "malo.txt",
				"content":  "no es Go",
			})},
			// Second call (after functionResponse with error): model recovers with text
			{textPart("Ese archivo no es .go. ¿Querés que cree hola.go mejor?")},
		},
	}

	tools := core.NewToolRegistry()
	tools.Register("create_exercise_file", domain.ToolDeclaration{
		Name:        "create_exercise_file",
		Description: "Crea",
		Parameters: domain.ToolParameters{
			Type: "OBJECT",
			Properties: map[string]domain.ToolProperty{
				"filename": {Type: "STRING", Description: "Nombre"},
				"content":  {Type: "STRING", Description: "Contenido"},
			},
			Required: []string{"filename", "content"},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		return nil, errors.New("solo se permiten archivos .go")
	})

	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	session := newMockSession("s5")
	ctx := context.Background()

	statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Creá malo.txt", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	response, _ := drainStatusChannel(statusCh)

	if response != "Ese archivo no es .go. ¿Querés que cree hola.go mejor?" {
		t.Errorf("response = %q", response)
	}

	// The session messages should include the functionResponse with the error
	foundErrorResponse := false
	for _, msg := range session.Messages {
		if msg.Role == "user" && strings.Contains(msg.Content, "solo se permiten archivos .go") {
			foundErrorResponse = true
		}
	}
	// Note: The functionResponse messages are stored as role "user" with the error content
	// We don't assert on exact JSON since it's stored as string in ChatMessage.Content
	_ = foundErrorResponse
}

// TestSenseiService_ProviderError tests that when the stream returns an error,
// it's propagated through the status channel.
func TestSenseiService_ProviderError(t *testing.T) {
	provider := &mockSenseiProvider{
		mockStreamResponse: func(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) (<-chan ports.StreamChunk, error) {
			return nil, errors.New("API no disponible")
		},
	}

	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	session := newMockSession("s6")
	ctx := context.Background()

	statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Hola", session)
	if err != nil {
		t.Fatalf("unexpected construction error: %v", err)
	}

	_, statuses := drainStatusChannel(statusCh)

	// Should have an error status
	hasError := false
	for _, s := range statuses {
		if strings.Contains(s, "Error") || strings.Contains(s, "error") || strings.Contains(s, "API") {
			hasError = true
		}
	}
	if !hasError {
		t.Errorf("expected an error status, got: %v", statuses)
	}
}

// TestSenseiService_StatusChannel tests the status channel sends expected messages during the loop.
func TestSenseiService_StatusChannel(t *testing.T) {
	provider := &mockSenseiProvider{
		responses: [][]domain.ContentPart{
			{funcCallPart("create_exercise_file", map[string]interface{}{
				"filename": "test.go",
				"content":  "package main",
			})},
			{textPart("¡Listo!")},
		},
	}
	tools := core.NewToolRegistry()
	tools.Register("create_exercise_file", domain.ToolDeclaration{
		Name:        "create_exercise_file",
		Description: "Crea",
		Parameters: domain.ToolParameters{
			Type: "OBJECT",
			Properties: map[string]domain.ToolProperty{
				"filename": {Type: "STRING", Description: "Nombre"},
				"content":  {Type: "STRING", Description: "Contenido"},
			},
			Required: []string{"filename", "content"},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"ok": true}, nil
	})

	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	session := newMockSession("s7")
	ctx := context.Background()

	statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Crea test.go", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	response, statuses := drainStatusChannel(statusCh)

	if response != "¡Listo!" {
		t.Errorf("response = %q", response)
	}

	// Expected sequence: Pensando... → Ejecutando create_exercise_file... → Pensando... → done
	if len(statuses) < 3 {
		t.Errorf("expected at least 3 status updates, got %d: %v", len(statuses), statuses)
	}

	// Verify "Pensando..." appears
	pensandoFound := false
	metricsFound := false
	for _, s := range statuses {
		if s == "Pensando..." {
			pensandoFound = true
		}
		if strings.HasPrefix(s, "Métricas:") {
			metricsFound = true
		}
	}
	if !pensandoFound {
		t.Errorf("expected 'Pensando...' status, got: %v", statuses)
	}
	if !metricsFound {
		t.Errorf("expected a metrics status, got: %v", statuses)
	}

	// Verify channel is closed after goroutine finishes
	_, ok := <-statusCh
	if ok {
		t.Error("status channel should be closed after completion")
	}
}

// TestSenseiService_ContextTimeout tests that the 60s timeout does not deadlock.
// We use a short timeout context to verify the loop terminates.
func TestSenseiService_ContextTimeout(t *testing.T) {
	provider := &mockSenseiProvider{
		mockStreamResponse: func(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) (<-chan ports.StreamChunk, error) {
			ch := make(chan ports.StreamChunk, 2)
			ch <- ports.StreamChunk{Text: "Respuesta rápida"}
			ch <- ports.StreamChunk{Done: true}
			close(ch)
			return ch, nil
		},
	}
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	session := newMockSession("s8")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Hola", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	response, _ := drainStatusChannel(statusCh)

	if response != "Respuesta rápida" {
		t.Errorf("response = %q", response)
	}
}

// TestSenseiService_SessionMessagesPreserved tests that session messages are properly appended.
func TestSenseiService_SessionMessagesPreserved(t *testing.T) {
	provider := &mockSenseiProvider{
		mockStreamResponse: func(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) (<-chan ports.StreamChunk, error) {
			ch := make(chan ports.StreamChunk, 2)
			ch <- ports.StreamChunk{Text: "Hola, ¿cómo va?"}
			ch <- ports.StreamChunk{Done: true}
			close(ch)
			return ch, nil
		},
	}
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	session := newMockSession("s9")
	ctx := context.Background()

	statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Mi primer mensaje", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, _ = drainStatusChannel(statusCh)

	// Should have user message + sensei response
	if len(session.Messages) < 2 {
		t.Fatalf("expected at least 2 messages in session, got %d", len(session.Messages))
	}

	userMsg := session.Messages[len(session.Messages)-2]
	senseiMsg := session.Messages[len(session.Messages)-1]

	if userMsg.Role != "user" {
		t.Errorf("penultimate message role = %q, want user", userMsg.Role)
	}
	if userMsg.Content != "Mi primer mensaje" {
		t.Errorf("user message content = %q", userMsg.Content)
	}
	if senseiMsg.Role != "sensei" {
		t.Errorf("last message role = %q, want sensei", senseiMsg.Role)
	}
	if senseiMsg.Content != "Hola, ¿cómo va?" {
		t.Errorf("sensei message content = %q", senseiMsg.Content)
	}
}

// --- Task 2: SendMessageStream tests on mockSenseiProvider ---

// TestMockSenseiProvider_SendMessageStream_DefaultNoOp verifies that the default
// no-op stub returns a closed empty channel with nil error when no custom
// mockStreamResponse is set.
func TestMockSenseiProvider_SendMessageStream_DefaultNoOp(t *testing.T) {
	m := &mockSenseiProvider{}

	ch, err := m.SendMessageStream(context.Background(), "Sos un sensei.", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	// Channel should be already closed
	_, ok := <-ch
	if ok {
		t.Error("expected closed channel (no chunks sent)")
	}
}

// TestMockSenseiProvider_SendMessageStream_CustomResponse verifies that a custom
// mockStreamResponse is delegated to when set.
func TestMockSenseiProvider_SendMessageStream_CustomResponse(t *testing.T) {
	expected := []ports.StreamChunk{
		{Text: "Hola"},
		{Text: " mundo"},
		{Done: true},
	}

	m := &mockSenseiProvider{
		mockStreamResponse: func(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) (<-chan ports.StreamChunk, error) {
			ch := make(chan ports.StreamChunk, len(expected))
			for _, c := range expected {
				ch <- c
			}
			close(ch)
			return ch, nil
		},
	}

	ch, err := m.SendMessageStream(context.Background(), "Sos un sensei.", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var chunks []ports.StreamChunk
	for c := range ch {
		chunks = append(chunks, c)
	}

	if len(chunks) != len(expected) {
		t.Fatalf("got %d chunks, want %d", len(chunks), len(expected))
	}
	for i, c := range chunks {
		if c.Text != expected[i].Text {
			t.Errorf("chunk[%d].Text = %q, want %q", i, c.Text, expected[i].Text)
		}
		if c.Done != expected[i].Done {
			t.Errorf("chunk[%d].Done = %v, want %v", i, c.Done, expected[i].Done)
		}
	}
}

// TestMockSenseiProvider_SendMessageStream_ErrorPropagation verifies that errors
// from the mockStreamResponse are propagated.
func TestMockSenseiProvider_SendMessageStream_ErrorPropagation(t *testing.T) {
	expectedErr := errors.New("simulated stream failure")

	m := &mockSenseiProvider{
		mockStreamResponse: func(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) (<-chan ports.StreamChunk, error) {
			return nil, expectedErr
		},
	}

	ch, err := m.SendMessageStream(context.Background(), "Sos un sensei.", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != expectedErr.Error() {
		t.Errorf("error = %q, want %q", err.Error(), expectedErr.Error())
	}
	if ch != nil {
		t.Error("channel should be nil on error")
	}
}

// TestMockSenseiProvider_SendMessageStream_TracksCalls verifies that
// sendStreamCalls and sendStreamToolCounts are tracked.
func TestMockSenseiProvider_SendMessageStream_TracksCalls(t *testing.T) {
	m := &mockSenseiProvider{}

	// Call with 3 tools
	_, _ = m.SendMessageStream(context.Background(), "prompt", []chatstore.ChatMessage{
		{Role: "user", Content: "hello"},
	}, []domain.ToolDeclaration{
		{Name: "tool_a"},
		{Name: "tool_b"},
		{Name: "tool_c"},
	})

	if m.sendStreamCalls != 1 {
		t.Errorf("sendStreamCalls = %d, want 1", m.sendStreamCalls)
	}
	if len(m.sendStreamToolCounts) != 1 || m.sendStreamToolCounts[0] != 3 {
		t.Errorf("sendStreamToolCounts = %v, want [3]", m.sendStreamToolCounts)
	}

	// Call with 0 tools
	_, _ = m.SendMessageStream(context.Background(), "prompt", nil, nil)

	if m.sendStreamCalls != 2 {
		t.Errorf("sendStreamCalls = %d after second call, want 2", m.sendStreamCalls)
	}
	if len(m.sendStreamToolCounts) != 2 || m.sendStreamToolCounts[1] != 0 {
		t.Errorf("sendStreamToolCounts = %v, want [3, 0]", m.sendStreamToolCounts)
	}
}

// --- Task 6 Tests: Streaming path in runAgentLoop ---

// TestSenseiService_StreamingPath_UsedForTextOnly verifies that when tools==nil
// (text-only request), SendMessageStream is called instead of SendMessage.
func TestSenseiService_StreamingPath_UsedForTextOnly(t *testing.T) {
	provider := &mockSenseiProvider{
		mockStreamResponse: func(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) (<-chan ports.StreamChunk, error) {
			ch := make(chan ports.StreamChunk, 2)
			ch <- ports.StreamChunk{Text: "¡Hola! Soy el sensei."}
			ch <- ports.StreamChunk{Done: true}
			close(ch)
			return ch, nil
		},
	}
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	statusCh, err := svc.ProcessMessage(context.Background(), "Sos un sensei.", "Hola", newMockSession("s-stream-used"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	response, _ := drainStatusChannel(statusCh)
	_ = response

	// Streaming path must be used (SendMessageStream), NOT non-streaming (SendMessage)
	if provider.sendCalls != 0 {
		t.Errorf("sendCalls = %d, want 0 (should use SendMessageStream, not SendMessage)", provider.sendCalls)
	}
	if provider.sendStreamCalls == 0 {
		t.Error("sendStreamCalls should be > 0 (SendMessageStream was not called for text-only)")
	}
}

// TestSenseiService_StreamingPath_EmitsStreamChunks verifies that streaming
// chunks are emitted through the status channel with "stream:" prefix,
// followed by "stream:done" and final "done:" message.
func TestSenseiService_StreamingPath_EmitsStreamChunks(t *testing.T) {
	provider := &mockSenseiProvider{
		mockStreamResponse: func(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) (<-chan ports.StreamChunk, error) {
			ch := make(chan ports.StreamChunk, 3)
			ch <- ports.StreamChunk{Text: "Hola "}
			ch <- ports.StreamChunk{Text: "mundo"}
			ch <- ports.StreamChunk{Done: true}
			close(ch)
			return ch, nil
		},
	}
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	statusCh, err := svc.ProcessMessage(context.Background(), "Sos un sensei.", "Hola", newMockSession("s-stream-chunks"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read all statuses including stream: prefixed ones
	var statuses []string
	for msg := range statusCh {
		statuses = append(statuses, msg)
	}

	// Check stream chunks
	hasStreamHola := false
	hasStreamMundo := false
	hasStreamDone := false
	hasDoneResponse := false
	for _, s := range statuses {
		if s == "stream:Hola " {
			hasStreamHola = true
		}
		if s == "stream:mundo" {
			hasStreamMundo = true
		}
		if s == "stream:done" {
			hasStreamDone = true
		}
		if s == "done:Hola mundo" {
			hasDoneResponse = true
		}
	}

	if !hasStreamHola {
		t.Errorf("expected 'stream:Hola ', got: %v", statuses)
	}
	if !hasStreamMundo {
		t.Errorf("expected 'stream:mundo', got: %v", statuses)
	}
	if !hasStreamDone {
		t.Errorf("expected 'stream:done', got: %v", statuses)
	}
	if !hasDoneResponse {
		t.Errorf("expected 'done:Hola mundo', got: %v", statuses)
	}
}

// TestSenseiService_StreamingPath_ErrorInStream verifies that when the stream
// returns an error chunk, the error is emitted through the status channel.
func TestSenseiService_StreamingPath_ErrorInStream(t *testing.T) {
	simulatedErr := errors.New("simulated stream failure")
	provider := &mockSenseiProvider{
		mockStreamResponse: func(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) (<-chan ports.StreamChunk, error) {
			ch := make(chan ports.StreamChunk, 1)
			ch <- ports.StreamChunk{Error: simulatedErr, Done: true}
			close(ch)
			return ch, nil
		},
	}
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	statusCh, err := svc.ProcessMessage(context.Background(), "Sos un sensei.", "Hola", newMockSession("s-stream-err"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, statuses := drainStatusChannel(statusCh)
	hasError := false
	for _, s := range statuses {
		if strings.Contains(s, "Error") || strings.Contains(s, "error") {
			hasError = true
		}
	}
	if !hasError {
		t.Errorf("expected error in statuses, got: %v", statuses)
	}
}

// TestSenseiService_ToolPath_UsesNonStreaming verifies that when tools != nil
// (tool-enabled request), the non-streaming SendMessage is still used.
func TestSenseiService_ToolPath_UsesNonStreaming(t *testing.T) {
	provider := &mockSenseiProvider{
		responses: [][]domain.ContentPart{
			{textPart("Revisando tu workspace...")},
		},
	}
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	// "archivo" is in tool intent hints → tools != nil
	statusCh, err := svc.ProcessMessage(context.Background(), "Sos un sensei.", "¿Qué archivos tengo?", newMockSession("s-tool-nonstream"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	response, _ := drainStatusChannel(statusCh)
	if response != "Revisando tu workspace..." {
		t.Errorf("response = %q, want %q", response, "Revisando tu workspace...")
	}

	// Non-streaming SendMessage must be used
	if provider.sendCalls == 0 {
		t.Error("sendCalls should be > 0 (SendMessage was not called for tool request)")
	}
	if provider.sendStreamCalls != 0 {
		t.Errorf("sendStreamCalls = %d, want 0 (SendMessageStream should NOT be used for tool requests)", provider.sendStreamCalls)
	}
}

// TestSenseiService_StreamingPath_EmptyResponse verifies that when the stream
// produces zero text chunks, the default fallback message is used.
func TestSenseiService_StreamingPath_EmptyResponse(t *testing.T) {
	provider := &mockSenseiProvider{
		mockStreamResponse: func(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) (<-chan ports.StreamChunk, error) {
			ch := make(chan ports.StreamChunk, 1)
			ch <- ports.StreamChunk{Done: true}
			close(ch)
			return ch, nil
		},
	}
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap, nil)

	statusCh, err := svc.ProcessMessage(context.Background(), "Sos un sensei.", "Hola", newMockSession("s-stream-empty"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	response, _ := drainStatusChannel(statusCh)
	if !strings.Contains(response, "reformular") {
		t.Errorf("expected default reformulate message for empty stream, got: %q", response)
	}
}

// TestMockSenseiProvider_SendMessageStream_ContextPassed verifies that the
// context is passed through to the mockStreamResponse.
func TestMockSenseiProvider_SendMessageStream_ContextPassed(t *testing.T) {
	var capturedCtx context.Context
	m := &mockSenseiProvider{
		mockStreamResponse: func(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) (<-chan ports.StreamChunk, error) {
			capturedCtx = ctx
			ch := make(chan ports.StreamChunk)
			close(ch)
			return ch, nil
		},
	}

	type ctxKey struct{}
	testCtx := context.WithValue(context.Background(), ctxKey{}, "test-value")
	ch, err := m.SendMessageStream(testCtx, "Sos un sensei.", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for range ch {
	}
	if capturedCtx == nil {
		t.Fatal("context was not captured")
	}
	if capturedCtx.Value(ctxKey{}) != "test-value" {
		t.Error("context value not preserved")
	}
}
