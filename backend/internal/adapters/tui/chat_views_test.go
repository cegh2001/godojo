package tui

import (
	"strings"
	"testing"
	"time"

	"godojo/internal/adapters/chatstore"
)

func TestViewSenseiChat_Empty_SenseiWelcome(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "Sensei Chat") {
		t.Error("chat view should show 'Sensei Chat' title")
	}
	if !strings.Contains(view, "Bienvenido al dojo") {
		t.Error("empty chat should show welcome message")
	}
}

func TestViewSenseiChat_ShowsMessages(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.width = 80
	m.height = 24
	m.chatMessages = []chatstore.ChatMessage{
		{Role: "user", Content: "¿Qué es Go?", Time: time.Now()},
		{Role: "sensei", Content: "Go es un lenguaje compilado y concurrente creado por Google.", Time: time.Now()},
	}

	view := m.View()
	if !strings.Contains(view, "¿Qué es Go?") {
		t.Error("chat view should show user message")
	}
	if !strings.Contains(view, "compilado") {
		t.Error("chat view should show sensei message")
	}
}

func TestViewSenseiChat_ShowsHelpBar(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "Enviar: Enter") {
		t.Error("chat view should show 'Enviar: Enter' help")
	}
	if !strings.Contains(view, "Nueva sesión: Ctrl+N") {
		t.Error("chat view should show 'Nueva sesión: Ctrl+N' help")
	}
	if !strings.Contains(view, "Sesiones: Ctrl+L") {
		t.Error("chat view should show 'Sesiones: Ctrl+L' help")
	}
	if !strings.Contains(view, "Volver: Esc") {
		t.Error("chat view should show 'Volver: Esc' help")
	}
}

func TestViewSenseiChat_ShowsLoadingIndicator(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.width = 80
	m.height = 24
	m.chatLoading = true
	m.chatMessages = []chatstore.ChatMessage{
		{Role: "user", Content: "Pregunta", Time: time.Now()},
	}

	view := m.View()
	if !strings.Contains(view, "Pensando") {
		t.Error("loading chat should show 'Pensando...' indicator")
	}
	if !strings.Contains(view, "esperando respuesta") {
		t.Error("loading chat should disable input with 'esperando respuesta'")
	}
}

func TestViewSenseiChat_ShowsPrunedNotification(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.width = 80
	m.height = 24
	m.chatPrunedMsg = "Sesión Vieja"

	view := m.View()
	if !strings.Contains(view, "Sesión más antigua eliminada") {
		t.Error("chat view should show pruned session notification")
	}
	if !strings.Contains(view, "Sesión Vieja") {
		t.Error("chat view should show pruned session name")
	}
}

func TestViewSenseiChat_NoPrunedNotification_WhenEmpty(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.width = 80
	m.height = 24
	m.chatPrunedMsg = ""

	view := m.View()
	if strings.Contains(view, "Sesión más antigua eliminada") {
		t.Error("should not show prune notification when chatPrunedMsg is empty")
	}
}

func TestViewSessionSelector_NoSessions(t *testing.T) {
	m := newModelTest()
	m.state = stateSessionSelector
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "Sesiones de chat") {
		t.Error("session selector should show title")
	}
	if !strings.Contains(view, "No hay sesiones guardadas") {
		t.Error("empty session list should show 'No hay sesiones guardadas'")
	}
}

func TestViewSessionSelector_ShowsSessions(t *testing.T) {
	m := newModelTest()
	m.state = stateSessionSelector
	m.width = 80
	m.height = 24
	m.cursor = 0
	now := time.Now()
	m.chatSessions = []chatstore.ChatSession{
		{
			ID:        "s1",
			Name:      "¿Qué es Go?",
			UpdatedAt: now,
			Messages: []chatstore.ChatMessage{
				{Role: "user", Content: "¿Qué es Go?"},
				{Role: "sensei", Content: "Es un lenguaje..."},
			},
		},
		{
			ID:        "s2",
			Name:      "Goroutines",
			UpdatedAt: now.Add(-1 * time.Hour),
			Messages: []chatstore.ChatMessage{
				{Role: "user", Content: "Goroutines?"},
			},
		},
	}

	view := m.View()
	if !strings.Contains(view, "¿Qué es Go?") {
		t.Error("session selector should show session names")
	}
	if !strings.Contains(view, "Goroutines") {
		t.Error("session selector should show all sessions")
	}
	if !strings.Contains(view, "2 mensajes") {
		t.Error("session selector should show message count")
	}
}

func TestViewSessionSelector_ShowsCursor(t *testing.T) {
	m := newModelTest()
	m.state = stateSessionSelector
	m.width = 80
	m.height = 24
	m.cursor = 1
	m.chatSessions = []chatstore.ChatSession{
		{ID: "s1", Name: "Sesión 1", UpdatedAt: time.Now(), Messages: []chatstore.ChatMessage{{Role: "user", Content: "x"}}},
		{ID: "s2", Name: "Sesión 2", UpdatedAt: time.Now(), Messages: []chatstore.ChatMessage{{Role: "user", Content: "x"}}},
	}

	view := m.View()
	if !strings.Contains(view, "→") {
		t.Error("session selector should show cursor indicator")
	}
}

func TestTruncateForList(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short enough", "Hola", 10, "Hola"},
		{"exact length", "1234567890", 10, "1234567890"},
		{"too long", "Este es un nombre muy largo de sesión", 15, "Este es un nomb..."},
		{"maxLen zero", "texto", 0, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateForList(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateForList(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestViewSenseiChat_ShowsSessionName(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.width = 80
	m.height = 24
	m.chatSessionID = "test-session"
	m.chatSessions = []chatstore.ChatSession{
		{
			ID:        "test-session",
			Name:      "Mi sesión de prueba",
			UpdatedAt: time.Now(),
			Messages:  []chatstore.ChatMessage{{Role: "user", Content: "Hola"}},
		},
	}

	view := m.View()
	if !strings.Contains(view, "Mi sesión de prueba") {
		t.Error("chat view should show current session name")
	}
}
