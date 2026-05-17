package tui

import (
	"context"
	"fmt"
	"testing"
	"time"

	"godojo/internal/adapters/chatstore"
	"godojo/internal/core"
	"godojo/internal/core/domain"
	"godojo/internal/core/services"

	tea "github.com/charmbracelet/bubbletea"
)

type stubSenseiProvider struct {
	parts []domain.ContentPart
	err   error
}

func (s stubSenseiProvider) SendMessage(_ context.Context, _ string, _ []chatstore.ChatMessage, _ []domain.ToolDeclaration) ([]domain.ContentPart, error) {
	return s.parts, s.err
}

func (s stubSenseiProvider) SendFunctionResponse(_ context.Context, _ []chatstore.ChatMessage, _ string, _ string, _ interface{}) ([]domain.ContentPart, error) {
	return s.parts, s.err
}

func TestModel_CtrlG_TransitionsToSenseiChat(t *testing.T) {
	// Ctrl+G from sensei chat stays in sensei chat and creates session ID if blank
	m := newModelTest()
	m.state = stateSenseiChat

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	updated := newM.(Model)

	if updated.state != stateSenseiChat {
		t.Errorf("after ctrl+g from sensei chat, state = %v, want %v", updated.state, stateSenseiChat)
	}
	if updated.chatSessionID == "" {
		t.Error("chatSessionID should be auto-created when entering chat")
	}
}

func TestModel_CtrlG_FromSenseiChat_KeepsState(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.chatSessionID = "existing"

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	updated := newM.(Model)

	if updated.state != stateSenseiChat {
		t.Errorf("after ctrl+g from sensei chat, state = %v, want %v", updated.state, stateSenseiChat)
	}
	// Session ID should be preserved
	if updated.chatSessionID != "existing" {
		t.Errorf("chatSessionID changed from 'existing' to %q", updated.chatSessionID)
	}
}

func TestModel_Esc_FromSenseiChat(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.chatSessionID = "test-id"
	m.chatMessages = []chatstore.ChatMessage{
		{Role: "user", Content: "Hola", Time: time.Now()},
	}

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := newM.(Model)

	// Esc from sensei chat stays in sensei chat (no previous view to return to)
	if updated.state != stateSenseiChat {
		t.Errorf("after Esc from chat, state = %v, want %v", updated.state, stateSenseiChat)
	}
}

func TestModel_Esc_FromSessionSelector_GoesBackToChat(t *testing.T) {
	m := newModelTest()
	m.state = stateSessionSelector
	m.chatSessionID = "test-session"
	m.chatMessages = []chatstore.ChatMessage{
		{Role: "user", Content: "Hola", Time: time.Now()},
	}

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := newM.(Model)

	if updated.state != stateSenseiChat {
		t.Errorf("after Esc from session selector, state = %v, want %v", updated.state, stateSenseiChat)
	}
}

func TestModel_SessionSelector_Delete_RemovesSelectedSession(t *testing.T) {
	store := chatstore.NewChatStore(t.TempDir())
	session := &chatstore.ChatSession{
		ID:        "s1",
		Name:      "Sesión para borrar",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages: []chatstore.ChatMessage{
			{Role: "user", Content: "Hola", Time: time.Now()},
		},
	}
	if _, err := store.SaveSession(session); err != nil {
		t.Fatalf("SaveSession() error: %v", err)
	}

	m := newModelTest()
	m.state = stateSessionSelector
	m.chatStore = store
	m.chatSessions = []chatstore.ChatSession{*session}
	m.chatSessionID = session.ID
	m.chatMessages = session.Messages
	m.cursor = 0

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	updated := newM.(Model)

	if len(updated.chatSessions) != 0 {
		t.Fatalf("expected 0 sessions after delete, got %d", len(updated.chatSessions))
	}
	if updated.chatSessionID != "" {
		t.Error("current chat session should be cleared after deleting the active session")
	}
	if len(updated.chatMessages) != 0 {
		t.Error("chat messages should be cleared after deleting the active session")
	}
	if _, err := store.LoadSession(session.ID); err == nil {
		t.Error("deleted session should no longer exist on disk")
	}
}

func TestModel_ChatSend_AddsUserMessage(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.chatSessionID = "test-session"
	m.chatInput = "¿Qué es una goroutine?"
	// senseiSvc is nil → no cmd dispatched, but message still appended

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := newM.(Model)

	if len(updated.chatMessages) != 2 {
		t.Fatalf("expected 2 messages after send (user + fallback), got %d", len(updated.chatMessages))
	}
	if updated.chatMessages[0].Role != "user" {
		t.Errorf("message role = %q, want %q", updated.chatMessages[0].Role, "user")
	}
	if updated.chatMessages[0].Content != "¿Qué es una goroutine?" {
		t.Errorf("message content = %q, want %q", updated.chatMessages[0].Content, "¿Qué es una goroutine?")
	}
	if updated.chatLoading {
		t.Error("chatLoading should be false when senseiSvc is nil (no async dispatch)")
	}
	if updated.chatInput != "" {
		t.Error("chatInput should be cleared after sending")
	}
}

func TestModel_ChatSend_EmptyInput_NoOp(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.chatInput = "   "
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := newM.(Model)

	if len(updated.chatMessages) != 0 {
		t.Error("should not add message for empty/whitespace input")
	}
}

func TestModel_ChatSend_CreatesSessionID_WhenBlank(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.chatInput = "¿Qué es una goroutine?"

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := newM.(Model)

	if updated.chatSessionID == "" {
		t.Error("chatSessionID should be created when sending with an empty session")
	}
}

func TestModel_ChatSend_WithSenseiService_ShowsUserInputImmediately(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.chatSessionID = "test-session"
	m.chatInput = "Necesito ayuda con slices"
	m.senseiSvc = services.NewSenseiService(
		stubSenseiProvider{parts: []domain.ContentPart{{Text: "Vamos con slices."}}},
		core.NewToolRegistry(),
		nil,
		services.NewRoadmapService(),
		nil,
	)

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := newM.(Model)

	if cmd == nil {
		t.Fatal("expected async command when sensei service exists")
	}
	if len(updated.chatMessages) != 1 {
		t.Fatalf("expected 1 local user message before response, got %d", len(updated.chatMessages))
	}
	if updated.chatMessages[0].Role != "user" {
		t.Errorf("first local message role = %q, want user", updated.chatMessages[0].Role)
	}
	if updated.chatMessages[0].Content != "Necesito ayuda con slices" {
		t.Errorf("first local message content = %q", updated.chatMessages[0].Content)
	}
	if !updated.chatLoading {
		t.Error("chatLoading should stay true while waiting for sensei response")
	}
	if updated.chatInput != "" {
		t.Error("chatInput should be cleared after enqueueing the message")
	}
	if updated.chatScroll != 0 {
		t.Errorf("chatScroll = %d, want 0", updated.chatScroll)
	}
	_ = cmd
}

func TestModel_ChatPaste_DoesNotSendOnPastedEnter(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.chatSessionID = "test-session"

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hola\nmundo"), Paste: true})
	updated := newM.(Model)

	if updated.chatInput != "hola mundo" {
		t.Errorf("chatInput = %q, want %q", updated.chatInput, "hola mundo")
	}
	if len(updated.chatMessages) != 0 {
		t.Fatalf("expected no messages after pasted text, got %d", len(updated.chatMessages))
	}

	newM, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter, Paste: true})
	updated = newM.(Model)

	if cmd != nil {
		t.Error("pasted enter should not dispatch a send command")
	}
	if len(updated.chatMessages) != 0 {
		t.Fatalf("expected no sent messages after pasted enter, got %d", len(updated.chatMessages))
	}
	if updated.chatInput != "hola mundo" {
		t.Errorf("chatInput should be preserved after pasted enter, got %q", updated.chatInput)
	}
	if updated.chatLoading {
		t.Error("chatLoading should remain false after pasted enter")
	}
}

func TestModel_ChatSend_WhileLoading_NoOp(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.chatSessionID = "test-id"
	m.chatInput = "Otro mensaje"
	m.chatLoading = true

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := newM.(Model)

	if len(updated.chatMessages) != 0 {
		t.Error("should not send message while chat is loading")
	}
}

func TestModel_ChatResponse_AppendsSenseiMessage(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.chatSessionID = "test-id"
	m.chatMessages = []chatstore.ChatMessage{
		{Role: "user", Content: "Hola", Time: time.Now()},
	}
	m.chatLoading = true

	newM, _ := m.Update(chatResponseMsg{content: "¡Hola! Soy el sensei.", err: nil})
	updated := newM.(Model)

	if len(updated.chatMessages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(updated.chatMessages))
	}
	if updated.chatMessages[1].Role != "sensei" {
		t.Errorf("second message role = %q, want %q", updated.chatMessages[1].Role, "sensei")
	}
	if updated.chatMessages[1].Content != "¡Hola! Soy el sensei." {
		t.Errorf("response content = %q", updated.chatMessages[1].Content)
	}
	if updated.chatLoading {
		t.Error("chatLoading should be false after response")
	}
}

func TestModel_ChatResponse_ErrorShowsInMessage(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.chatSessionID = "test-id"
	m.chatLoading = true

	newM, _ := m.Update(chatResponseMsg{content: "", err: fmt.Errorf("timeout")})
	updated := newM.(Model)

	if len(updated.chatMessages) != 1 {
		t.Fatalf("expected 1 error message, got %d", len(updated.chatMessages))
	}
	if updated.chatLoading {
		t.Error("chatLoading should be false after error")
	}
}

func TestModel_ChatNew_CreatesNewSession(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.chatSessionID = "old-session"
	m.chatMessages = []chatstore.ChatMessage{
		{Role: "user", Content: "Mensaje anterior", Time: time.Now()},
	}
	m.chatInput = "some text"

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	updated := newM.(Model)

	if updated.chatSessionID == "old-session" {
		t.Error("chatSessionID should change on new session")
	}
	if len(updated.chatMessages) != 0 {
		t.Error("messages should be cleared for new session")
	}
	if updated.chatInput != "" {
		t.Error("input should be cleared for new session")
	}
}

func TestModel_ChatList_TransitionsToSelector(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.chatStore = chatstore.NewChatStore(t.TempDir())

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	updated := newM.(Model)

	if updated.state != stateSessionSelector {
		t.Errorf("after ctrl+l, state = %v, want %v", updated.state, stateSessionSelector)
	}
}

func TestModel_ChatTextInput_Accumulates(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	updated := newM.(Model)
	if updated.chatInput != "H" {
		t.Errorf("input after 'H' = %q, want %q", updated.chatInput, "H")
	}

	newM2, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	updated2 := newM2.(Model)
	if updated2.chatInput != "Ho" {
		t.Errorf("input after 'Ho' = %q, want %q", updated2.chatInput, "Ho")
	}

	newM3, _ := updated2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l', 'a'}})
	updated3 := newM3.(Model)
	if updated3.chatInput != "Hola" {
		t.Errorf("input after 'Hola' = %q, want %q", updated3.chatInput, "Hola")
	}
}

func TestModel_ChatTextInput_Backspace(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.chatInput = "Hola"

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	updated := newM.(Model)

	if updated.chatInput != "Hol" {
		t.Errorf("input after backspace = %q, want %q", updated.chatInput, "Hol")
	}
}

func TestModel_ChatTextInput_IgnoredWhileLoading(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat
	m.chatLoading = true
	m.chatInput = ""

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	updated := newM.(Model)

	if updated.chatInput != "" {
		t.Error("text input should be ignored while chat is loading")
	}
}

func TestModel_SessionSelector_Navigation(t *testing.T) {
	m := newModelTest()
	m.state = stateSessionSelector
	m.chatSessions = []chatstore.ChatSession{
		{ID: "s1", Name: "Sesión 1", UpdatedAt: time.Now()},
		{ID: "s2", Name: "Sesión 2", UpdatedAt: time.Now()},
	}
	m.cursor = 0

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := newM.(Model)
	if updated.cursor != 1 {
		t.Errorf("cursor after 'j' in selector = %d, want 1", updated.cursor)
	}

	newM2, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	updated2 := newM2.(Model)
	if updated2.cursor != 0 {
		t.Errorf("cursor after 'k' in selector = %d, want 0", updated2.cursor)
	}
}

func TestModel_SessionSelector_BoundaryDown(t *testing.T) {
	m := newModelTest()
	m.state = stateSessionSelector
	m.chatSessions = []chatstore.ChatSession{
		{ID: "s1", Name: "Sesión 1", UpdatedAt: time.Now()},
	}
	m.cursor = 0

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := newM.(Model)
	if updated.cursor != 0 {
		t.Errorf("cursor should stay at 0 at boundary, got %d", updated.cursor)
	}
}

func TestModel_CtrlL_OnlyInSenseiChat(t *testing.T) {
	m := newModelTest()
	m.state = stateSessionSelector

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	updated := newM.(Model)

	if updated.state != stateSessionSelector {
		t.Error("ctrl+l should not change state outside of sensei chat")
	}
}

func TestModel_CtrlN_OnlyInChat(t *testing.T) {
	m := newModelTest()
	m.state = stateSessionSelector

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	updated := newM.(Model)

	if updated.state != stateSessionSelector {
		t.Error("ctrl+n should not have effect outside of sensei chat")
	}
}
