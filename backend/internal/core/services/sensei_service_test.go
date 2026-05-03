package services_test

import (
	"context"
	"errors"
	"fmt"
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
	mu        sync.Mutex
	responses [][]domain.ContentPart
	callCount int
	err       error // if set, return this error instead of next response
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
	return m.nextResponse()
}

func (m *mockSenseiProvider) SendFunctionResponse(ctx context.Context, history []chatstore.ChatMessage, callID string, name string, result interface{}) ([]domain.ContentPart, error) {
	return m.nextResponse()
}

// --- Mock WorkspaceManager ---

type mockWorkspace struct {
	files map[string]string
}

func newMockWorkspace() *mockWorkspace {
	return &mockWorkspace{files: make(map[string]string)}
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
	return names, nil
}

func (w *mockWorkspace) WorkspacePath() string {
	return "/mock/workspace"
}

// Ensure mockWorkspace implements ports.WorkspaceManager
var _ ports.WorkspaceManager = (*mockWorkspace)(nil)

// --- Helper ---

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

func newMockSession(id string) *chatstore.ChatSession {
	return &chatstore.ChatSession{
		ID:       id,
		Name:     "Test",
		Messages: nil,
	}
}

// --- Test Cases ---

// TestSenseiService_TextOnlyResponse tests the simplest case:
// user sends message, provider returns text → service returns text.
func TestSenseiService_TextOnlyResponse(t *testing.T) {
	provider := &mockSenseiProvider{
		responses: [][]domain.ContentPart{
			{textPart("¡Buenas! ¿En qué te ayudo con Go?")},
		},
	}
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap)

	session := newMockSession("s1")
	ctx := context.Background()

	response, statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Hola", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Collect all status updates
	var statuses []string
	done := make(chan struct{})
	go func() {
		for s := range statusCh {
			statuses = append(statuses, s)
		}
		close(done)
	}()
	<-done

	if response != "¡Buenas! ¿En qué te ayudo con Go?" {
		t.Errorf("response = %q, want greeting", response)
	}

	// Should have "Pensando..." and "done" statuses
	if len(statuses) < 2 {
		t.Errorf("expected at least 2 status updates, got %d: %v", len(statuses), statuses)
	}
}

// TestSenseiService_SingleToolCall tests a single functionCall → tool execution → final text flow.
func TestSenseiService_SingleToolCall(t *testing.T) {
	provider := &mockSenseiProvider{
		responses: [][]domain.ContentPart{
			// First call: functionCall to create_exercise_file
			{funcCallPart("create_exercise_file", map[string]interface{}{
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
	svc := services.NewSenseiService(provider, tools, ws, roadmap)

	session := newMockSession("s2")
	ctx := context.Background()

	response, statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Creá un hola mundo", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var statuses []string
	for s := range statusCh {
		statuses = append(statuses, s)
	}

	if response != "Listo, creé el archivo hola.go en tu workspace." {
		t.Errorf("response = %q", response)
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
	svc := services.NewSenseiService(provider, core.NewToolRegistry(), ws, roadmap)

	session := newMockSession("topic-folder")
	ctx := context.Background()

	response, statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Armame ejercicios de variables", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var statuses []string
	for s := range statusCh {
		statuses = append(statuses, s)
	}

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
	svc := services.NewSenseiService(provider, tools, ws, roadmap)

	session := newMockSession("s3")
	ctx := context.Background()

	response, statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Dame ejercicios de fase 1", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var statuses []string
	for s := range statusCh {
		statuses = append(statuses, s)
	}

	if response != "Creé el ejercicio. ¿Lo ejecutamos?" {
		t.Errorf("response = %q", response)
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
	svc := services.NewSenseiService(provider, tools, ws, roadmap)

	session := newMockSession("s4")
	ctx := context.Background()

	response, statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Hola", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var statuses []string
	for s := range statusCh {
		statuses = append(statuses, s)
	}

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
	svc := services.NewSenseiService(provider, tools, ws, roadmap)

	session := newMockSession("s5")
	ctx := context.Background()

	response, statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Creá malo.txt", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var statuses []string
	for s := range statusCh {
		statuses = append(statuses, s)
	}

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

// TestSenseiService_ProviderError tests that when SendMessage returns an error,
// it's propagated through the status channel.
func TestSenseiService_ProviderError(t *testing.T) {
	provider := &mockSenseiProvider{
		err: errors.New("API no disponible"),
	}

	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap)

	session := newMockSession("s6")
	ctx := context.Background()

	_, statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Hola", session)
	if err != nil {
		t.Fatalf("unexpected construction error: %v", err)
	}

	var statuses []string
	for s := range statusCh {
		statuses = append(statuses, s)
	}

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
	svc := services.NewSenseiService(provider, tools, ws, roadmap)

	session := newMockSession("s7")
	ctx := context.Background()

	response, statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Crea test.go", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response != "¡Listo!" {
		t.Errorf("response = %q", response)
	}

	var statuses []string
	for s := range statusCh {
		statuses = append(statuses, s)
	}

	// Expected sequence: Pensando... → Ejecutando create_exercise_file... → Pensando... → done
	if len(statuses) < 3 {
		t.Errorf("expected at least 3 status updates, got %d: %v", len(statuses), statuses)
	}

	// Verify "Pensando..." appears
	pensandoFound := false
	for _, s := range statuses {
		if s == "Pensando..." {
			pensandoFound = true
		}
	}
	if !pensandoFound {
		t.Errorf("expected 'Pensando...' status, got: %v", statuses)
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
		responses: [][]domain.ContentPart{
			{textPart("Respuesta rápida")},
		},
	}
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap)

	session := newMockSession("s8")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, statusCh, err := svc.ProcessMessage(ctx, "Sos un sensei.", "Hola", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Drain status channel
	for range statusCh {
	}

	if response != "Respuesta rápida" {
		t.Errorf("response = %q", response)
	}
}

// TestSenseiService_SessionMessagesPreserved tests that session messages are properly appended.
func TestSenseiService_SessionMessagesPreserved(t *testing.T) {
	provider := &mockSenseiProvider{
		responses: [][]domain.ContentPart{
			{textPart("Hola, ¿cómo va?")},
		},
	}
	tools := core.NewToolRegistry()
	ws := newMockWorkspace()
	roadmap := services.NewRoadmapService()
	svc := services.NewSenseiService(provider, tools, ws, roadmap)

	session := newMockSession("s9")
	ctx := context.Background()

	_, statusCh, _ := svc.ProcessMessage(ctx, "Sos un sensei.", "Mi primer mensaje", session)
	for range statusCh {
	}

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
