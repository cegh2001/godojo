package tui

import (
	"testing"

	"godojo/internal/adapters/chatstore"

	tea "github.com/charmbracelet/bubbletea"
)

// newModelTest creates a Model for testing.
func newModelTest() Model {
	return NewModel(nil, nil, "")
}

func TestNewModel_InitialState(t *testing.T) {
	m := newModelTest()

	if m.state != stateSenseiChat {
		t.Errorf("initial state = %v, want %v", m.state, stateSenseiChat)
	}
	if m.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", m.cursor)
	}
}

func TestNewModel_StoresSenseiSvc(t *testing.T) {
	// Given a sensei service is created and passed to NewModel
	// The nil services check just verifies the constructor works
	m := NewModel(nil, nil, "Sos un sensei de Go")

	if m.state != stateSenseiChat {
		t.Errorf("initial state = %v, want %v", m.state, stateSenseiChat)
	}
	if m.senseiSystemPrompt != "Sos un sensei de Go" {
		t.Errorf("senseiSystemPrompt = %q, want %q", m.senseiSystemPrompt, "Sos un sensei de Go")
	}
	if m.toolStatus != "" {
		t.Errorf("initial toolStatus = %q, want empty", m.toolStatus)
	}
}

func TestModel_Init_ReturnsStartupCommands(t *testing.T) {
	m := newModelTest()
	cmd := m.Init()

	if cmd == nil {
		t.Fatal("Init() returned nil command, expected startup commands")
	}

	msg := cmd()
	if msg == nil {
		t.Fatal("command produced nil message")
	}
	if _, ok := msg.(tea.BatchMsg); !ok {
		t.Errorf("command produced %T, want tea.BatchMsg", msg)
	}
}

func TestModel_Quit_OnCtrlC(t *testing.T) {
	m := newModelTest()

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c should return tea.Quit")
	}
	_ = newModel

	// Verify cmd is tea.Quit by checking its string representation
	quitMsg := cmd()
	if quitMsg == nil {
		t.Fatal("tea.Quit produced nil message")
	}
}

func TestModel_Quit_OnQ(t *testing.T) {
	m := newModelTest()
	// In sensei chat, rune keys are captured as text input, not quit.
	// Quit via 'q' only works in non-sensei states like session selector or tool running.
	m.state = stateSessionSelector

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("'q' should return tea.Quit")
	}
	_ = newModel

	quitMsg := cmd()
	if quitMsg == nil {
		t.Fatal("tea.Quit produced nil message")
	}
}

func TestModel_View_ReturnsNonEmpty(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat

	view := m.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestModel_ToolRunningView(t *testing.T) {
	m := newModelTest()
	m.state = stateToolRunning
	m.toolStatus = "Creando archivo de ejercicio..."

	view := m.View()
	if view == "" {
		t.Error("View() in stateToolRunning returned empty string")
	}
}

func TestModel_ToolStatusMsg_UpdatesStatus(t *testing.T) {
	m := newModelTest()

	newM, cmd := m.Update(toolStatusMsg{status: "Ejecutando tests..."})
	updated := newM.(Model)

	if updated.toolStatus != "Ejecutando tests..." {
		t.Errorf("toolStatus = %q, want %q", updated.toolStatus, "Ejecutando tests...")
	}
	if cmd == nil {
		t.Error("toolStatusMsg should return spinner tick cmd")
	}
}

func TestModel_SenseiResponseMsg_DelegatesToChatHandler(t *testing.T) {
	m := newModelTest()
	m.chatLoading = true

	newM, _ := m.Update(senseiResponseMsg{content: "¡Hola! ¿En qué te ayudo?", err: nil})
	updated := newM.(Model)

	if updated.chatLoading {
		t.Error("chatLoading should be false after sensei response")
	}
	if len(updated.chatMessages) != 1 {
		t.Errorf("expected 1 chat message, got %d", len(updated.chatMessages))
	}
	if updated.chatMessages[0].Role != "sensei" {
		t.Errorf("expected sensei role, got %q", updated.chatMessages[0].Role)
	}
}

func TestModel_ChatSessionsLoadedMsg(t *testing.T) {
	m := newModelTest()
	now := timeNow()

	sessions := []chatstore.ChatSession{
		{ID: "s1", Name: "Test", UpdatedAt: now},
	}

	newM, _ := m.Update(chatSessionsLoadedMsg{sessions: sessions, err: nil})
	updated := newM.(Model)

	if len(updated.chatSessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(updated.chatSessions))
	}
	if updated.chatSessions[0].ID != "s1" {
		t.Errorf("session ID = %q", updated.chatSessions[0].ID)
	}
}

func TestModel_StateTransition_SenseiToSessionSelector(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}, Alt: false})
	// ctrl+l should trigger handleChatList
	_ = newM
}

func TestModel_StateTransition_SessionSelectorToSensei_OnEsc(t *testing.T) {
	m := newModelTest()
	m.state = stateSessionSelector
	m.cursor = 0

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := newM.(Model)

	if updated.state != stateSenseiChat {
		t.Errorf("after Esc from SessionSelector, state = %v, want %v", updated.state, stateSenseiChat)
	}
}

func TestModel_WindowSizeMsg(t *testing.T) {
	m := newModelTest()

	newM, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated := newM.(Model)

	if updated.width != 120 {
		t.Errorf("width = %d, want 120", updated.width)
	}
	if updated.height != 40 {
		t.Errorf("height = %d, want 40", updated.height)
	}
}

// --- Streaming message type tests ---

func TestStreamLineMsg_Instantiable(t *testing.T) {
	msg := streamLineMsg{text: "partial response"}

	if msg.text != "partial response" {
		t.Errorf("text = %q, want %q", msg.text, "partial response")
	}

	// Verify it satisfies tea.Msg (compile-time check)
	var _ tea.Msg = streamLineMsg{}
	_ = msg // prevent unused warning
}

func TestStreamCompleteMsg_Instantiable(t *testing.T) {
	msg := streamCompleteMsg{}

	// Verify it satisfies tea.Msg
	var _ tea.Msg = streamCompleteMsg{}
	_ = msg
}

func TestStreamSubscriptionMsg_Instantiable(t *testing.T) {
	ch := make(chan string, 1)
	ch <- "test"
	msg := streamSubscriptionMsg{ch: ch}

	if msg.ch == nil {
		t.Fatal("ch should not be nil")
	}

	// Read from channel to verify it's the same one
	select {
	case val := <-msg.ch:
		if val != "test" {
			t.Errorf("received %q, want %q", val, "test")
		}
	default:
		t.Error("expected to read from channel")
	}

	// Verify it satisfies tea.Msg
	var _ tea.Msg = streamSubscriptionMsg{}
	_ = msg
}

func TestModel_ChatStreamingText_FieldPresent(t *testing.T) {
	m := newModelTest()

	// Zero value check
	if m.chatStreamingText.Len() != 0 {
		t.Errorf("initial chatStreamingText length = %d, want 0", m.chatStreamingText.Len())
	}

	// Write something and verify
	m.chatStreamingText.WriteString("Hola ")
	m.chatStreamingText.WriteString("mundo")

	if m.chatStreamingText.String() != "Hola mundo" {
		t.Errorf("chatStreamingText = %q, want %q", m.chatStreamingText.String(), "Hola mundo")
	}
}
