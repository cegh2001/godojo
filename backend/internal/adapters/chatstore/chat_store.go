package chatstore

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ChatMessage represents a single message in a chat session.
type ChatMessage struct {
	Role    string    `json:"role"` // "user" or "sensei"
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}

// ChatSession represents a persistent chat session.
type ChatSession struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"` // auto-generated from first message
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Messages  []ChatMessage `json:"messages"`
}

// ChatStore manages persistent chat sessions on disk.
type ChatStore struct {
	dir         string // ~/.godojo/sessions/
	maxSessions int    // 10
}

// NewChatStore creates a new ChatStore with the given directory for session files.
func NewChatStore(dir string) *ChatStore {
	return &ChatStore{
		dir:         dir,
		maxSessions: 10,
	}
}

// ListSessions returns all sessions sorted by UpdatedAt descending (most recent first).
func (s *ChatStore) ListSessions() ([]ChatSession, error) {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return nil, fmt.Errorf("error al crear directorio de sesiones: %w", err)
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("error al leer directorio de sesiones: %w", err)
	}

	var sessions []ChatSession
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		session, err := s.LoadSession(id)
		if err != nil {
			// Skip corrupted files
			continue
		}
		sessions = append(sessions, *session)
	}

	// Sort by UpdatedAt descending (most recent first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

// LoadSession loads a single session by ID from its JSON file.
func (s *ChatStore) LoadSession(id string) (*ChatSession, error) {
	path := filepath.Join(s.dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("sesión no encontrada: %s", id)
		}
		return nil, fmt.Errorf("error al leer sesión: %w", err)
	}

	var session ChatSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("error al decodificar sesión: %w", err)
	}

	return &session, nil
}

// SaveSession persists a session to disk. If saving would exceed maxSessions,
// the oldest session (by CreatedAt) is deleted and its name is returned for notification.
// Returns the pruned session name (empty string if none pruned) and any error.
func (s *ChatStore) SaveSession(session *ChatSession) (string, error) {
	if session == nil {
		return "", fmt.Errorf("no se puede guardar una sesión nula")
	}

	if len(session.Messages) == 0 {
		return "", fmt.Errorf("no se guardan sesiones sin mensajes")
	}

	// Auto-name: first 40 chars of first user message
	if session.Name == "" || session.Name == "Nuevo chat" {
		for _, msg := range session.Messages {
			if msg.Role == "user" {
				name := msg.Content
				if len([]rune(name)) > 40 {
					name = string([]rune(name)[:40]) + "..."
				}
				session.Name = name
				break
			}
		}
		if session.Name == "" {
			session.Name = "Nuevo chat"
		}
	}

	// Generate ID if not present
	if session.ID == "" {
		session.ID = NewSessionID()
	}

	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}
	session.UpdatedAt = time.Now()

	// Ensure directory exists
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return "", fmt.Errorf("error al crear directorio de sesiones: %w", err)
	}

	path := filepath.Join(s.dir, session.ID+".json")
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error al codificar sesión: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("error al guardar sesión: %w", err)
	}

	// Enforce max sessions
	prunedName, err := s.enforceMaxSessions()
	if err != nil {
		return prunedName, err
	}

	return prunedName, nil
}

// enforceMaxSessions ensures no more than maxSessions exist.
// Deletes the oldest session (by CreatedAt) if needed.
// Returns the name of the pruned session (empty if none).
func (s *ChatStore) enforceMaxSessions() (string, error) {
	sessions, err := s.ListSessions()
	if err != nil {
		return "", err
	}

	if len(sessions) <= s.maxSessions {
		return "", nil
	}

	// Find oldest by CreatedAt
	var oldest *ChatSession
	for i := range sessions {
		sess := &sessions[i]
		if oldest == nil || sess.CreatedAt.Before(oldest.CreatedAt) {
			oldest = sess
		}
	}

	if oldest == nil {
		return "", nil
	}

	name := oldest.Name
	if err := s.DeleteSession(oldest.ID); err != nil {
		return name, err
	}

	return name, nil
}

// NewSessionID generates a unique session ID using crypto/rand.
func NewSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// DeleteSession removes a session file from disk.
func (s *ChatStore) DeleteSession(id string) error {
	if id == "" {
		return fmt.Errorf("ID de sesión vacío")
	}

	path := filepath.Join(s.dir, id+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil // idempotent: deleting non-existent is OK
		}
		return fmt.Errorf("error al eliminar sesión: %w", err)
	}

	return nil
}
