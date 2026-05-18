package tui

import (
	"context"
	"fmt"
	"strings"

	"godojo/internal/adapters/chatstore"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleCtrlG() (tea.Model, tea.Cmd) {
	m.state = stateSenseiChat
	m.chatPrunedMsg = ""

	if m.chatSessionID == "" {
		m.chatSessionID = chatstore.NewSessionID()
		m.chatMessages = nil
	}
	m.chatScroll = 0

	return m, nil
}

func (m Model) handleChatTextInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.chatLoading {
		return m, nil
	}

	if msg.Type == tea.KeyBackspace || (len(msg.Runes) == 1 && msg.Runes[0] == 127) {
		if len(m.chatInput) > 0 {
			runes := []rune(m.chatInput)
			m.chatInput = string(runes[:len(runes)-1])
		}
		return m, nil
	}

	if msg.Paste {
		m.chatInput += normalizePastedChatInput(string(msg.Runes))
		return m, nil
	}

	for _, r := range msg.Runes {
		m.chatInput += string(r)
	}
	return m, nil
}

func (m Model) handleChatSend() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.chatInput)
	if input == "" || m.chatLoading {
		return m, nil
	}

	if m.chatSessionID == "" {
		m.chatSessionID = chatstore.NewSessionID()
	}

	historyBeforeSend := append([]chatstore.ChatMessage(nil), m.chatMessages...)
	now := timeNow()
	m.chatMessages = append(m.chatMessages, chatstore.ChatMessage{
		Role:    "user",
		Content: input,
		Time:    now,
	})

	m.chatInput = ""
	m.chatLoading = true
	m.chatScroll = 0
	m.toolStatus = ""
	m.saveCurrentChatSession()

	if m.senseiSvc == nil {
		m.chatMessages = append(m.chatMessages, chatstore.ChatMessage{
			Role:    "sensei",
			Content: "Sensei no disponible — configura GEMINI_API_KEY en .env",
			Time:    now,
		})
		m.chatLoading = false
		m.saveCurrentChatSession()
		return m, nil
	}

	return m, m.sendSenseiCmd(input, historyBeforeSend)
}

func (m Model) sendSenseiCmd(userMessage string, history []chatstore.ChatMessage) tea.Cmd {
	return func() tea.Msg {
		session := &chatstore.ChatSession{
			ID:       m.chatSessionID,
			Messages: history,
		}

		statusCh, err := m.senseiSvc.ProcessMessage(context.Background(), m.senseiSystemPrompt, userMessage, session)
		if err != nil {
			return senseiResponseMsg{err: err}
		}

		// Return the live channel via streamSubscriptionMsg — the Update function
		// will launch streamReaderCmd to read it progressively.
		return streamSubscriptionMsg{ch: statusCh}
	}
}

func (m Model) handleChatResponse(msg chatResponseMsg) (tea.Model, tea.Cmd) {
	if msg.content == "" && msg.err != nil {
		msg.content = fmt.Sprintf("Error: %v", msg.err)
	}
	if msg.status != "" {
		m.toolStatus = msg.status
	}

	m.chatMessages = append(m.chatMessages, chatstore.ChatMessage{
		Role:    "sensei",
		Content: msg.content,
		Time:    timeNow(),
	})
	m.chatLoading = false
	m.chatScroll = 0
	m.saveCurrentChatSession()

	return m, nil
}

// streamReaderCmd reads one status line from the live channel.
// Returns streamLineMsg if a line is available, or streamCompleteMsg if the channel is closed.
func streamReaderCmd(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		text, ok := <-ch
		if !ok {
			return streamCompleteMsg{}
		}
		return streamLineMsg{text: text}
	}
}

func normalizePastedChatInput(text string) string {
	return strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(text)
}

// handleStreamLine processes a single status line from the live ProcessMessage channel.
// It dispatches based on prefix: stream:, stream:done, done:, error:, Métricas:, or default.
// Returns a streamReaderCmd to schedule the next read when streamCh is available.
func (m Model) handleStreamLine(text string) Model {
	switch {
	case strings.HasPrefix(text, "stream:done"):
		return m
	case strings.HasPrefix(text, "done:"):
		response := strings.TrimPrefix(text, "done:")
		m.chatMessages = append(m.chatMessages, chatstore.ChatMessage{
			Role:    "sensei",
			Content: response,
			Time:    timeNow(),
		})
		m.chatLoading = false
		m.chatScroll = 0
		m.chatStreamingText.Reset()
		m.saveCurrentChatSession()
		return m
	case strings.HasPrefix(text, "error:"):
		msg := strings.TrimPrefix(text, "error:")
		m.chatMessages = append(m.chatMessages, chatstore.ChatMessage{
			Role:    "sensei",
			Content: msg,
			Time:    timeNow(),
		})
		m.chatLoading = false
		m.chatStreamingText.Reset()
		return m
	case strings.HasPrefix(text, "Métricas:"):
		m.toolStatus = text
		return m
	case strings.HasPrefix(text, "stream:"):
		chunk := strings.TrimPrefix(text, "stream:")
		m.chatStreamingText.WriteString(chunk)
		return m
	default:
		m.toolStatus = text
		return m
	}
}

func (m Model) handleChatNew() (tea.Model, tea.Cmd) {
	m.saveCurrentChatSession()
	m.chatSessionID = chatstore.NewSessionID()
	m.chatMessages = nil
	m.chatInput = ""
	m.chatLoading = false
	m.chatScroll = 0
	m.chatPrunedMsg = ""
	return m, nil
}

func (m Model) handleChatList() (tea.Model, tea.Cmd) {
	if m.chatStore == nil {
		return m, nil
	}

	sessions, err := m.chatStore.ListSessions()
	if err != nil {
		m.chatSessions = nil
	} else {
		m.chatSessions = sessions
	}

	m.state = stateSessionSelector
	m.cursor = 0
	return m, nil
}

func (m Model) handleSessionSelectorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyDelete:
		return m.handleSessionDelete()
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		return m.handleEsc()
	case "enter":
		return m.handleSessionSelect()
	case "up", "k":
		return m.handleCursorUp()
	case "down", "j":
		return m.handleCursorDown()
	case "del", "delete", "ctrl+d":
		return m.handleSessionDelete()
	}
	return m, nil
}

func (m Model) handleSessionDelete() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.chatSessions) {
		return m, nil
	}

	if m.chatStore == nil {
		return m, nil
	}

	selected := m.chatSessions[m.cursor]
	if err := m.chatStore.DeleteSession(selected.ID); err != nil {
		return m, nil
	}

	deletedCurrent := selected.ID == m.chatSessionID
	m.chatSessions = append(m.chatSessions[:m.cursor], m.chatSessions[m.cursor+1:]...)
	if m.cursor >= len(m.chatSessions) && m.cursor > 0 {
		m.cursor--
	}
	if len(m.chatSessions) == 0 {
		m.cursor = 0
	}

	if deletedCurrent {
		m.chatSessionID = ""
		m.chatMessages = nil
		m.chatInput = ""
		m.chatLoading = false
		m.chatScroll = 0
		m.chatPrunedMsg = ""
	}

	return m, nil
}

func (m Model) handleSessionSelect() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.chatSessions) {
		return m, nil
	}

	if m.chatStore == nil {
		return m, nil
	}

	selected := m.chatSessions[m.cursor]
	session, err := m.chatStore.LoadSession(selected.ID)
	if err != nil {
		return m, nil
	}

	m.chatSessionID = session.ID
	m.chatMessages = session.Messages
	m.chatInput = ""
	m.chatLoading = false
	m.chatScroll = 0
	m.chatPrunedMsg = ""
	m.state = stateSenseiChat
	m.cursor = 0

	return m, nil
}

func (m Model) handleChatSessionsLoaded(msg chatSessionsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.chatSessions = nil
	} else {
		m.chatSessions = msg.sessions
	}
	return m, nil
}

// handleCursorUp moves the cursor up by one position.
func (m Model) handleCursorUp() (tea.Model, tea.Cmd) {
	if m.cursor > 0 {
		m.cursor--
	}
	return m, nil
}

// handleCursorDown moves the cursor down by one position.
func (m Model) handleCursorDown() (tea.Model, tea.Cmd) {
	maxLen := m.getCursorMax()
	if m.cursor < maxLen-1 {
		m.cursor++
	}
	return m, nil
}
