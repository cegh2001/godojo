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
	if !strings.Contains(view, "¡Hola! Soy tu sensei de Go") {
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

func TestViewSenseiChat_WrapsLongMessages(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.width = 40
	m.height = 24
	m.chatMessages = []chatstore.ChatMessage{
		{
			Role:    "sensei",
			Content: "Este es un mensaje largo del sensei que debe envolver el texto para no cortarse al final.",
			Time:    time.Now(),
		},
	}

	view := m.View()
	normalized := strings.Join(strings.Fields(view), " ")
	if !strings.Contains(normalized, "cortarse al final") {
		t.Error("long sensei messages should wrap instead of truncating their ending")
	}
}

func TestRenderChatMessages_RespectsScrollOffset(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.width = 40
	m.height = 24
	m.chatMessages = []chatstore.ChatMessage{
		{Role: "user", Content: "mensaje uno", Time: time.Now()},
		{Role: "sensei", Content: "respuesta uno", Time: time.Now()},
		{Role: "user", Content: "mensaje dos", Time: time.Now()},
		{Role: "sensei", Content: "respuesta dos", Time: time.Now()},
		{Role: "user", Content: "mensaje tres", Time: time.Now()},
	}

	bottom := m.renderChatMessages(3)
	if !strings.Contains(bottom, "mensaje tres") {
		t.Fatal("bottom of chat should show the latest message")
	}

	m.chatScroll = 2
	scrolled := m.renderChatMessages(3)
	if strings.Contains(scrolled, "mensaje tres") {
		t.Fatal("scrolled chat should move away from the latest message")
	}
	if !strings.Contains(scrolled, "respuesta uno") {
		t.Fatal("scrolled chat should reveal older messages")
	}
}

func TestShiftChatScroll_ClampsAndMoves(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.width = 40
	m.height = 8
	m.chatMessages = []chatstore.ChatMessage{
		{Role: "user", Content: "mensaje uno", Time: time.Now()},
		{Role: "sensei", Content: "respuesta uno", Time: time.Now()},
		{Role: "user", Content: "mensaje dos", Time: time.Now()},
		{Role: "sensei", Content: "respuesta dos", Time: time.Now()},
		{Role: "user", Content: "mensaje tres", Time: time.Now()},
	}

	next := m.shiftChatScroll(1)
	if next.chatScroll != 1 {
		t.Fatalf("chatScroll after first shift = %d, want 1", next.chatScroll)
	}

	next = next.shiftChatScroll(1)
	if next.chatScroll != 2 {
		t.Fatalf("chatScroll after second shift = %d, want 2", next.chatScroll)
	}

	next = next.shiftChatScroll(-1)
	if next.chatScroll != 1 {
		t.Fatalf("chatScroll after reverse shift = %d, want 1", next.chatScroll)
	}
}

func TestViewSenseiChat_ShowsComposerPlaceholder(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "escribí tu mensaje") {
		t.Error("empty chat should show a clear composer placeholder")
	}
}

func TestViewSenseiChat_ShowsTailOfLongInput(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.width = 34
	m.height = 24
	m.chatInput = "PRINCIPIO-DE-TEXTO-MUY-LARGO-QUE-SE-DEJA-DE-VER-Y-FINAL-VISIBLE"

	view := m.View()
	if !strings.Contains(view, "FINAL-VISIBLE") {
		t.Fatal("long input should keep the tail visible")
	}
	if strings.Contains(view, "PRINCIPIO-DE-TEXTO-MUY-LARGO") {
		t.Fatal("long input should clip the hidden prefix")
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
	if !strings.Contains(view, "pensando") {
		t.Error("loading chat should show 'pensando...' indicator")
	}
	if !strings.Contains(view, "Tú:") {
		t.Error("loading chat should clearly show the user composer line")
	}
	if !strings.Contains(view, "enviando") {
		t.Error("loading chat should show the send status in the composer")
	}
}

func TestViewSenseiChat_ShowsScrollStatus(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.width = 80
	m.height = 8
	m.chatScroll = 1
	m.chatMessages = []chatstore.ChatMessage{
		{Role: "user", Content: "mensaje uno", Time: time.Now()},
		{Role: "sensei", Content: "respuesta uno", Time: time.Now()},
		{Role: "user", Content: "mensaje dos", Time: time.Now()},
		{Role: "sensei", Content: "respuesta dos", Time: time.Now()},
		{Role: "user", Content: "mensaje tres", Time: time.Now()},
	}

	view := m.View()
	if !strings.Contains(view, "Viendo mensajes anteriores") {
		t.Error("chat view should explain when the transcript is scrolled up")
	}
}

func TestViewSenseiChat_ShowsHelpBar(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "Ctrl+N") {
		t.Error("chat view should show 'Ctrl+N' help")
	}
	if !strings.Contains(view, "Ctrl+L") {
		t.Error("chat view should show 'Ctrl+L' help")
	}
	if !strings.Contains(view, "Ctrl+C: salir") {
		t.Error("chat view should show 'Ctrl+C: salir' help")
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

func TestViewSessionSelector_ShowsDeleteHint(t *testing.T) {
	m := newModelTest()
	m.state = stateSessionSelector
	m.width = 80
	m.height = 24
	m.chatSessions = []chatstore.ChatSession{
		{ID: "s1", Name: "Sesión 1", UpdatedAt: time.Now(), Messages: []chatstore.ChatMessage{{Role: "user", Content: "x"}}},
	}

	view := m.View()
	if !strings.Contains(view, "del: eliminar") {
		t.Error("session selector should show delete hint")
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
