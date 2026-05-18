package tui

import (
	"time"

	"godojo/internal/adapters/chatstore"
	"godojo/internal/core/services"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// timeNow returns the current time. Exists for testability.
var timeNow = func() time.Time { return time.Now() }

// tuiState represents the current view state in the TUI state machine.
type tuiState int

const (
	stateSenseiChat      tuiState = iota // default landing, chat with sensei
	stateToolRunning                     // spinner during agent loop
	stateSessionSelector                 // session list overlay
)

// Model is the main Bubbletea model for the GoDojo TUI.
type Model struct {
	// State machine
	state tuiState

	// Core services
	senseiSvc          *services.SenseiService
	senseiSystemPrompt string
	toolStatus         string // current tool status for ToolRunning view

	// UI state
	cursor  int // selected item index
	spinner spinner.Model
	width   int
	height  int
	err     error

	// Sensei chat
	chatStore     *chatstore.ChatStore
	chatSessions  []chatstore.ChatSession
	chatMessages  []chatstore.ChatMessage
	chatInput     string
	chatLoading   bool
	chatScroll    int
	chatSessionID string
	chatPrunedMsg string // notification about pruned session

	// Streaming state
	chatStreamingText string // accumulates progressive stream text
	streamCh          <-chan string   // live status channel for recursive TUI reads
}

// toolStatusMsg carries a status update during agent loop.
type toolStatusMsg struct {
	status string
}

// senseiResponseMsg carries the final response from the sensei.
type senseiResponseMsg struct {
	content string
	err     error
	status  string
}

// chatResponseMsg is sent when the sensei responds.
type chatResponseMsg struct {
	content string
	err     error
	status  string
}

// chatSessionsLoadedMsg is sent when the session list is loaded from the store.
type chatSessionsLoadedMsg struct {
	sessions []chatstore.ChatSession
	err      error
}

// streamLineMsg carries a single line of streaming text from the sensei.
type streamLineMsg struct {
	text string
}

// streamCompleteMsg signals that the streaming response has finished.
type streamCompleteMsg struct{}

// streamSubscriptionMsg carries the live status channel from ProcessMessage.
type streamSubscriptionMsg struct {
	ch <-chan string
}

// NewModel creates a new TUI Model with SenseiService and chat dependencies.
func NewModel(
	senseiSvc *services.SenseiService,
	chatStore *chatstore.ChatStore,
	senseiSystemPrompt string,
) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = spinnerStyle

	return Model{
		state:              stateSenseiChat,
		senseiSvc:          senseiSvc,
		chatStore:          chatStore,
		senseiSystemPrompt: senseiSystemPrompt,
		spinner:            sp,
	}
}

// Init returns the initial command.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tea.EnableBracketedPaste)
}

// Update handles messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global quit: Ctrl+C always works
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.state == stateSenseiChat && msg.Paste {
			return m.handleChatTextInput(msg)
		}

		// SenseiChat key handling
		if m.state == stateSenseiChat {
			// Esc goes to session list
			if msg.String() == "esc" {
				return m.handleChatList()
			}
			// Text input: route to chat handler
			if msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace {
				return m.handleChatTextInput(msg)
			}
		}

		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case toolStatusMsg:
		m.toolStatus = msg.status
		return m, m.spinner.Tick

	case senseiResponseMsg:
		return m.handleChatResponse(chatResponseMsg{content: msg.content, err: msg.err, status: msg.status})

	case chatResponseMsg:
		return m.handleChatResponse(msg)

	case chatSessionsLoadedMsg:
		return m.handleChatSessionsLoaded(msg)

	case streamSubscriptionMsg:
		m.streamCh = msg.ch
		return m, streamReaderCmd(msg.ch)

	case streamLineMsg:
		return m.handleStreamLine(msg.text), streamReaderCmd(m.streamCh)

	case streamCompleteMsg:
		m.streamCh = nil
		return m, nil

	case spinner.TickMsg:
		if m.state == stateToolRunning || (m.state == stateSenseiChat && m.chatLoading) {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	default:
		return m, nil
	}
}

// View renders the current TUI view based on state.
func (m Model) View() string {
	switch m.state {
	case stateSenseiChat:
		return m.viewSenseiChat()
	case stateToolRunning:
		return m.viewToolRunning()
	case stateSessionSelector:
		return m.viewSessionSelector()
	default:
		return "Cargando..."
	}
}

// saveCurrentChatSession persists the current session to disk.
func (m *Model) saveCurrentChatSession() {
	if m.chatStore == nil || m.chatSessionID == "" || len(m.chatMessages) == 0 {
		return
	}

	session := &chatstore.ChatSession{
		ID:       m.chatSessionID,
		Messages: m.chatMessages,
	}

	pruned, err := m.chatStore.SaveSession(session)
	if err != nil {
		return // silently fail — don't block UI
	}

	if pruned != "" {
		m.chatPrunedMsg = pruned
	}
}
