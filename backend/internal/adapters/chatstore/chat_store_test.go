package chatstore_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"godojo/internal/adapters/chatstore"
)

func newTestStore(t *testing.T) *chatstore.ChatStore {
	t.Helper()
	dir := t.TempDir()
	return chatstore.NewChatStore(dir)
}

func newTestSession(id, name string, messages []chatstore.ChatMessage) *chatstore.ChatSession {
	now := time.Now()
	return &chatstore.ChatSession{
		ID:        id,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  messages,
	}
}

func TestNewChatStore(t *testing.T) {
	dir := t.TempDir()
	store := chatstore.NewChatStore(dir)

	if store == nil {
		t.Fatal("NewChatStore returned nil")
	}

	// Should create directory on first ListSessions
	sessions, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions for new store, got %d", len(sessions))
	}

	// Verify directory was created
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("directory should have been created")
	}
}

func TestSaveAndLoadSession(t *testing.T) {
	store := newTestStore(t)

	session := newTestSession("", "",
		[]chatstore.ChatMessage{
			{Role: "user", Content: "¿Cómo declaro una variable en Go?", Time: time.Now()},
			{Role: "sensei", Content: "Usá `var nombre tipo` o el operador `:=` para inferencia.", Time: time.Now()},
		},
	)

	pruned, err := store.SaveSession(session)
	if err != nil {
		t.Fatalf("SaveSession() error: %v", err)
	}
	if pruned != "" {
		t.Errorf("unexpected pruned session: %q", pruned)
	}
	if session.ID == "" {
		t.Fatal("session ID should be auto-generated")
	}
	if session.Name == "" {
		t.Fatal("session name should be auto-generated")
	}

	// Load and verify
	loaded, err := store.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error: %v", err)
	}
	if loaded.ID != session.ID {
		t.Errorf("loaded ID = %q, want %q", loaded.ID, session.ID)
	}
	if loaded.Name != session.Name {
		t.Errorf("loaded name = %q, want %q", loaded.Name, session.Name)
	}
	if len(loaded.Messages) != 2 {
		t.Errorf("loaded messages count = %d, want 2", len(loaded.Messages))
	}
}

func TestSaveSession_AutoName_FromFirstUserMessage(t *testing.T) {
	store := newTestStore(t)

	session := newTestSession("", "",
		[]chatstore.ChatMessage{
			{Role: "user", Content: "¿Qué es un puntero en Go?", Time: time.Now()},
		},
	)

	_, err := store.SaveSession(session)
	if err != nil {
		t.Fatalf("SaveSession() error: %v", err)
	}

	if session.Name != "¿Qué es un puntero en Go?" {
		t.Errorf("auto-name = %q, want first user message", session.Name)
	}
}

func TestSaveSession_AutoName_TruncatesLongMessage(t *testing.T) {
	store := newTestStore(t)

	longMsg := "Este es un mensaje muy largo que excede los cuarenta caracteres para probar el truncado automático del nombre de sesión"
	session := newTestSession("", "",
		[]chatstore.ChatMessage{
			{Role: "user", Content: longMsg, Time: time.Now()},
		},
	)

	_, err := store.SaveSession(session)
	if err != nil {
		t.Fatalf("SaveSession() error: %v", err)
	}

	if len([]rune(session.Name)) > 43 { // 40 + "..."
		t.Errorf("name too long: %d runes, expected <= 43", len([]rune(session.Name)))
	}
	if !strings.HasSuffix(session.Name, "...") {
		t.Errorf("long name should end with '...': %q", session.Name)
	}
}

func TestSaveSession_AutoName_FallsBackWhenNoUserMessages(t *testing.T) {
	store := newTestStore(t)

	session := newTestSession("", "",
		[]chatstore.ChatMessage{
			{Role: "sensei", Content: "¡Bienvenido al dojo! ¿En qué te puedo ayudar?", Time: time.Now()},
		},
	)

	_, err := store.SaveSession(session)
	if err != nil {
		t.Fatalf("SaveSession() error: %v", err)
	}

	if session.Name != "Nuevo chat" {
		t.Errorf("fallback name = %q, want %q", session.Name, "Nuevo chat")
	}
}

func TestSaveSession_RejectsEmptyMessages(t *testing.T) {
	store := newTestStore(t)

	session := newTestSession("", "", nil)
	_, err := store.SaveSession(session)
	if err == nil {
		t.Fatal("expected error for session with no messages")
	}
}

func TestSaveSession_RejectsNil(t *testing.T) {
	store := newTestStore(t)

	_, err := store.SaveSession(nil)
	if err == nil {
		t.Fatal("expected error for nil session")
	}
}

func TestSaveSession_UpdatesTimestamp(t *testing.T) {
	store := newTestStore(t)

	oldTime := time.Now().Add(-1 * time.Hour)
	session := newTestSession("test-id", "Test",
		[]chatstore.ChatMessage{
			{Role: "user", Content: "Hola", Time: oldTime},
		},
	)
	session.CreatedAt = oldTime
	session.UpdatedAt = oldTime

	_, err := store.SaveSession(session)
	if err != nil {
		t.Fatalf("SaveSession() error: %v", err)
	}

	loaded, _ := store.LoadSession("test-id")
	if loaded.UpdatedAt.Before(oldTime) || loaded.UpdatedAt.Equal(oldTime) {
		t.Error("UpdatedAt should be refreshed on save")
	}
}

func TestSaveSession_PreservesGivenID(t *testing.T) {
	store := newTestStore(t)

	session := newTestSession("my-custom-id", "Custom",
		[]chatstore.ChatMessage{
			{Role: "user", Content: "Hola", Time: time.Now()},
		},
	)

	_, err := store.SaveSession(session)
	if err != nil {
		t.Fatalf("SaveSession() error: %v", err)
	}

	if session.ID != "my-custom-id" {
		t.Errorf("ID changed: %q, want %q", session.ID, "my-custom-id")
	}
}

func TestListSessions_ReturnsSortedByUpdatedAt(t *testing.T) {
	store := newTestStore(t)

	// Create sessions with different timestamps
	s1 := newTestSession("s1", "Oldest",
		[]chatstore.ChatMessage{{Role: "user", Content: "msg1", Time: time.Now()}},
	)
	s1.CreatedAt = time.Now().Add(-3 * time.Hour)
	s1.UpdatedAt = time.Now().Add(-3 * time.Hour)
	store.SaveSession(s1)

	s2 := newTestSession("s2", "Middle",
		[]chatstore.ChatMessage{{Role: "user", Content: "msg2", Time: time.Now()}},
	)
	s2.CreatedAt = time.Now().Add(-2 * time.Hour)
	s2.UpdatedAt = time.Now().Add(-2 * time.Hour)
	store.SaveSession(s2)

	s3 := newTestSession("s3", "Newest",
		[]chatstore.ChatMessage{{Role: "user", Content: "msg3", Time: time.Now()}},
	)
	s3.CreatedAt = time.Now().Add(-1 * time.Hour)
	s3.UpdatedAt = time.Now().Add(-1 * time.Hour)
	store.SaveSession(s3)

	sessions, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}

	if len(sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(sessions))
	}

	// Most recent first
	if sessions[0].Name != "Newest" {
		t.Errorf("first session = %q, want %q", sessions[0].Name, "Newest")
	}
	if sessions[2].Name != "Oldest" {
		t.Errorf("last session = %q, want %q", sessions[2].Name, "Oldest")
	}
}

func TestMaxSessions_PrunesOldest(t *testing.T) {
	store := newTestStore(t)

	// Save 11 sessions
	for i := 0; i < 11; i++ {
		session := newTestSession("", "",
			[]chatstore.ChatMessage{
				{Role: "user", Content: "Mensaje " + string(rune('A'+i)), Time: time.Now()},
			},
		)
		// Stagger creation time to ensure ordering
		session.CreatedAt = time.Now().Add(-time.Duration(11-i) * time.Hour)

		pruned, err := store.SaveSession(session)
		if err != nil {
			t.Fatalf("SaveSession(%d) error: %v", i, err)
		}

		if i < 10 {
			// First 10 saves should not prune
			if pruned != "" {
				t.Errorf("unexpected prune on save %d: %q", i, pruned)
			}
		} else {
			// 11th save should prune the oldest
			if pruned == "" {
				t.Error("expected oldest session to be pruned on 11th save")
			}
		}
	}

	// Verify only 10 sessions remain
	sessions, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}

	if len(sessions) != 10 {
		t.Errorf("expected 10 sessions after max enforcement, got %d", len(sessions))
	}
}

func TestMaxSessions_ReturnsPrunedName(t *testing.T) {
	store := newTestStore(t)

	// Create a session with known name that will be the oldest
	oldestNow := time.Now()
	oldest := newTestSession("oldest-id", "Sesión Vieja",
		[]chatstore.ChatMessage{{Role: "user", Content: "Mensaje viejo", Time: oldestNow}},
	)
	oldest.CreatedAt = oldestNow.Add(-10 * time.Hour)
	store.SaveSession(oldest)

	// Fill remaining slots with newer sessions
	for i := 0; i < 9; i++ {
		session := newTestSession("", "",
			[]chatstore.ChatMessage{
				{Role: "user", Content: "Mensaje " + string(rune('A'+i)), Time: time.Now()},
			},
		)
		store.SaveSession(session)
	}

	// Now save one more — should prune "Sesión Vieja"
	extra := newTestSession("", "",
		[]chatstore.ChatMessage{{Role: "user", Content: "Mensaje Extra", Time: time.Now()}},
	)
	pruned, err := store.SaveSession(extra)
	if err != nil {
		t.Fatalf("SaveSession() error: %v", err)
	}

	if pruned != "Sesión Vieja" {
		t.Errorf("pruned name = %q, want %q", pruned, "Sesión Vieja")
	}
}

func TestDeleteSession(t *testing.T) {
	store := newTestStore(t)

	session := newTestSession("delete-me", "Para borrar",
		[]chatstore.ChatMessage{{Role: "user", Content: "Hola", Time: time.Now()}},
	)
	store.SaveSession(session)

	err := store.DeleteSession("delete-me")
	if err != nil {
		t.Fatalf("DeleteSession() error: %v", err)
	}

	// Verify it's gone
	_, err = store.LoadSession("delete-me")
	if err == nil {
		t.Fatal("expected error loading deleted session")
	}
}

func TestDeleteSession_Idempotent(t *testing.T) {
	store := newTestStore(t)

	// Deleting non-existent session should not error
	err := store.DeleteSession("no-existe")
	if err != nil {
		t.Errorf("DeleteSession on non-existent should be idempotent, got: %v", err)
	}
}

func TestDeleteSession_EmptyID_Errors(t *testing.T) {
	store := newTestStore(t)

	err := store.DeleteSession("")
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestLoadSession_NonExistent(t *testing.T) {
	store := newTestStore(t)

	_, err := store.LoadSession("no-existe")
	if err == nil {
		t.Fatal("expected error for non-existent session")
	}
	if !strings.Contains(err.Error(), "no encontrada") {
		t.Errorf("error should mention 'no encontrada', got: %v", err)
	}
}

func TestLoadSession_CorruptedFile(t *testing.T) {
	// Use a real directory (not TempDir) to simulate corrupted file
	dir := t.TempDir()
	store := chatstore.NewChatStore(dir)

	// Write invalid JSON
	os.WriteFile(filepath.Join(dir, "corrupt.json"), []byte("not valid json"), 0644)

	_, err := store.LoadSession("corrupt")
	if err == nil {
		t.Fatal("expected error loading corrupted session")
	}
}

func TestSessionJSON_Encoding(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	session := chatstore.ChatSession{
		ID:        "test-id",
		Name:      "Mi sesión",
		CreatedAt: now,
		UpdatedAt: now,
		Messages: []chatstore.ChatMessage{
			{Role: "user", Content: "Hola sensei", Time: now},
			{Role: "sensei", Content: "¡Hola! ¿En qué te ayudo?", Time: now},
		},
	}

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var decoded chatstore.ChatSession
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if decoded.ID != "test-id" {
		t.Errorf("decoded ID = %q", decoded.ID)
	}
	if decoded.Name != "Mi sesión" {
		t.Errorf("decoded Name = %q", decoded.Name)
	}
	if len(decoded.Messages) != 2 {
		t.Errorf("decoded messages = %d", len(decoded.Messages))
	}
}

func TestSaveSession_KeepsNameIfAlreadySet(t *testing.T) {
	store := newTestStore(t)

	session := newTestSession("", "Mi Nombre Personalizado",
		[]chatstore.ChatMessage{
			{Role: "user", Content: "¿Cómo uso goroutines?", Time: time.Now()},
		},
	)

	_, err := store.SaveSession(session)
	if err != nil {
		t.Fatalf("SaveSession() error: %v", err)
	}

	// Should keep the custom name, not override with first message
	if session.Name != "Mi Nombre Personalizado" {
		t.Errorf("name was overridden: %q, want %q", session.Name, "Mi Nombre Personalizado")
	}
}

func TestListSessions_SkipsCorruptedFiles(t *testing.T) {
	dir := t.TempDir()
	store := chatstore.NewChatStore(dir)

	// Save a valid session
	valid := newTestSession("valid", "Válido",
		[]chatstore.ChatMessage{{Role: "user", Content: "Hola", Time: time.Now()}},
	)
	store.SaveSession(valid)

	// Write a corrupted file
	os.WriteFile(filepath.Join(dir, "corrupt.json"), []byte("{broken"), 0644)

	sessions, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("expected 1 valid session, got %d", len(sessions))
	}
}

func TestNewChatStore_MaxSessionsDefault(t *testing.T) {
	store := newTestStore(t)

	// Save exactly 10, should succeed
	for i := 0; i < 10; i++ {
		session := newTestSession("", "",
			[]chatstore.ChatMessage{
				{Role: "user", Content: "msg", Time: time.Now()},
			},
		)
		_, err := store.SaveSession(session)
		if err != nil {
			t.Fatalf("SaveSession(%d) error: %v", i, err)
		}
	}

	sessions, _ := store.ListSessions()
	if len(sessions) != 10 {
		t.Errorf("expected 10 sessions, got %d", len(sessions))
	}
}
