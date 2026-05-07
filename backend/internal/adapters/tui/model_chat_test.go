package tui

import (
	"testing"

	"godojo/internal/adapters/chatstore"

	"github.com/charmbracelet/bubbles/spinner"
)

func TestModel_ChatSessionsLoadedMsg_Integration(t *testing.T) {
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

func TestModel_ChatResponseMsg_AppendsMessage(t *testing.T) {
	m := newModelTest()
	m.chatLoading = true

	newM, _ := m.Update(chatResponseMsg{content: "Respuesta del sensei", err: nil})
	updated := newM.(Model)

	if updated.chatLoading {
		t.Error("chatLoading should be false after response")
	}
	if len(updated.chatMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(updated.chatMessages))
	}
	if updated.chatMessages[0].Role != "sensei" {
		t.Errorf("expected sensei role, got %q", updated.chatMessages[0].Role)
	}
	if updated.chatMessages[0].Content != "Respuesta del sensei" {
		t.Errorf("content = %q, want %q", updated.chatMessages[0].Content, "Respuesta del sensei")
	}
}

func TestModel_ChatResponseMsg_StoresFinalStatus(t *testing.T) {
	m := newModelTest()
	m.chatLoading = true

	newM, _ := m.Update(chatResponseMsg{content: "Respuesta del sensei", status: "Métricas: 1.2s · 1 ronda · 1 llamada al modelo · 0 herramientas"})
	updated := newM.(Model)

	if updated.toolStatus == "" {
		t.Fatal("toolStatus should keep the final metrics summary")
	}
	if updated.toolStatus != "Métricas: 1.2s · 1 ronda · 1 llamada al modelo · 0 herramientas" {
		t.Errorf("toolStatus = %q", updated.toolStatus)
	}
}

func TestModel_ChatResponseMsg_Error(t *testing.T) {
	m := newModelTest()
	m.chatLoading = true

	newM, _ := m.Update(chatResponseMsg{content: "", err: assertAnError{}})
	updated := newM.(Model)

	if updated.chatLoading {
		t.Error("chatLoading should be false after error response")
	}
	if len(updated.chatMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(updated.chatMessages))
	}
	if updated.chatMessages[0].Content == "" {
		t.Error("error message should have content")
	}
}

type assertAnError struct{}

func (e assertAnError) Error() string { return "test error" }

func TestModel_NewModel_StoresChatStore(t *testing.T) {
	store := &chatstore.ChatStore{}
	m := NewModel(nil, store, "")

	if m.chatStore != store {
		t.Error("chatStore should be stored in model")
	}
	if m.state != stateSenseiChat {
		t.Errorf("initial state = %v, want %v", m.state, stateSenseiChat)
	}
}

func TestModel_SpinnerTick_InToolRunning(t *testing.T) {
	m := newModelTest()
	m.state = stateToolRunning

	spinnerTick := spinner.TickMsg{}
	newM, cmd := m.Update(spinnerTick)
	_ = newM
	_ = cmd
	// Just verify it doesn't panic
}

func TestModel_SpinnerTick_InChatLoading(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.chatLoading = true

	spinnerTick := spinner.TickMsg{}
	newM, cmd := m.Update(spinnerTick)
	_ = newM
	_ = cmd
	// Just verify it doesn't panic
}

func TestModel_SpinnerTick_NotInRelevantState(t *testing.T) {
	m := newModelTest()
	// stateSenseiChat with chatLoading=false, or stateSessionSelector — no spinner update

	spinnerTick := spinner.TickMsg{}
	_, cmd := m.Update(spinnerTick)

	// Spinner tick should be ignored (no command returned)
	if cmd != nil {
		t.Error("spinner tick should return nil when not in relevant state")
	}
}

func TestModel_SenseiResponseMsg_Error(t *testing.T) {
	m := newModelTest()
	m.chatLoading = true
	m.chatSessionID = "test-session"

	newM, _ := m.Update(senseiResponseMsg{content: "", err: assertAnError{}})
	updated := newM.(Model)

	if updated.chatLoading {
		t.Error("chatLoading should be false after error")
	}
	if len(updated.chatMessages) == 0 {
		t.Fatal("expected at least 1 message")
	}
}
