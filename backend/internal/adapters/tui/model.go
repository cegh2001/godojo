package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"godojo/internal/adapters/chatstore"
	"godojo/internal/core/domain"
	"godojo/internal/core/ports"
)

// tuiState represents the current view state in the TUI state machine.
type tuiState int

const (
	stateRoadmapView    tuiState = iota // browsing phases/topics
	stateTopicDetail                    // viewing topic info + exercises
	stateExerciseView                   // viewing exercise description
	stateTestRunning                    // spinner while tests run
	stateTestResults                    // showing test output
	stateHintDisplay                    // showing Socratic hint
	stateSenseiChat                     // chat with AI sensei
	stateSessionSelector                // session list overlay
)

// roadmapService defines the interface for roadmap operations.
type roadmapService interface {
	GetRoadmap() (*domain.Roadmap, error)
	GetTopicBySlug(slug string) (*domain.Topic, error)
}

// exerciseService defines the interface for exercise operations.
type exerciseService interface {
	StartExercise(slug string) (*domain.Exercise, error)
	GetExercisesByTopic(topicSlug string) ([]*domain.ExerciseRef, error)
	ValidateExercise(slug string, testResult *domain.TestResult) (bool, error)
}

// progressService defines the interface for progress operations.
type progressService interface {
	MarkStarted(exerciseSlug string) error
	MarkCompleted(exerciseSlug string) error
	GetCompletionPercent(topicSlug string) (float64, error)
	GetProgress(exerciseSlug string) (*domain.Progress, error)
	GetAllProgress() (map[string]*domain.Progress, error)
}

// hintService defines the interface for hint operations.
type hintService interface {
	RequestHint(exercise *domain.Exercise, testOutput string) (<-chan *domain.Hint, <-chan error)
	IsAvailable() bool
}

// chatProvider defines the interface for chat AI operations.
type chatProvider interface {
	SendMessage(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage) (string, error)
}

// Model is the main Bubbletea model for the GoDojo TUI.
type Model struct {
	// State machine
	state        tuiState
	previousView tuiState // used for back navigation from transient states

	// Services (wired via constructor)
	roadmapSvc  roadmapService
	exerciseSvc exerciseService
	progressSvc progressService
	hintSvc     hintService

	// Current context
	roadmap         *domain.Roadmap
	currentTopic    *domain.Topic
	currentExercise *domain.Exercise

	// UI state
	cursor        int // selected item index
	testResult    *domain.TestResult
	hintResult    *domain.Hint
	hintError     error
	hintRequested bool // true while waiting for hint, false when user navigates away
	spinner       spinner.Model
	width         int
	height        int
	err           error

	// Content lists for views
	phases    []*domain.Phase
	topics    []*domain.Topic
	exercises []*domain.ExerciseRef

	// For command execution
	testRunner     ports.TestRunner
	exerciseRepo   ports.ExerciseRepository
	workspacePath  string
	lastTestOutput string

	// Sensei chat
	chatProvider   chatProvider
	chatStore      *chatstore.ChatStore
	chatSessions   []chatstore.ChatSession
	chatMessages   []chatstore.ChatMessage
	chatInput      string
	chatLoading    bool
	chatSessionID  string
	chatPrunedMsg  string // notification about pruned session
}

// roadmapLoadedMsg is sent when the roadmap is loaded from RoadmapService.
type roadmapLoadedMsg struct {
	roadmap *domain.Roadmap
}

// topicSelectedMsg is sent when the user selects a topic.
type topicSelectedMsg struct {
	topic *domain.Topic
}

// exerciseSelectedMsg is sent when the user selects an exercise.
type exerciseSelectedMsg struct {
	exercise *domain.Exercise
}

// testResultMsg is sent when go test completes.
type testResultMsg struct {
	result *domain.TestResult
	err    error
}

// hintResultMsg is sent when a hint arrives from the provider.
type hintResultMsg struct {
	hint *domain.Hint
}

// hintErrorMsg is sent when hint fetching fails.
type hintErrorMsg struct {
	err error
}

// chatResponseMsg is sent when the sensei responds.
type chatResponseMsg struct {
	content string
	err     error
}

// chatSessionsLoadedMsg is sent when the session list is loaded from the store.
type chatSessionsLoadedMsg struct {
	sessions []chatstore.ChatSession
	err      error
}

// NewModel creates a new TUI Model with the given services, test runner, repo, workspace, and chat dependencies.
func NewModel(
	roadmapSvc roadmapService,
	exerciseSvc exerciseService,
	progressSvc progressService,
	hintSvc hintService,
	testRunner ports.TestRunner,
	exerciseRepo ports.ExerciseRepository,
	workspacePath string,
	chatStore *chatstore.ChatStore,
	chatProv chatProvider,
) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = spinnerStyle

	return Model{
		state:         stateRoadmapView,
		roadmapSvc:    roadmapSvc,
		exerciseSvc:   exerciseSvc,
		progressSvc:   progressSvc,
		hintSvc:       hintSvc,
		testRunner:    testRunner,
		exerciseRepo:  exerciseRepo,
		workspacePath: workspacePath,
		spinner:       sp,
		chatStore:     chatStore,
		chatProvider:  chatProv,
	}
}

// Init returns the initial command: load the roadmap.
func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		rm, err := m.roadmapSvc.GetRoadmap()
		if err != nil {
			return roadmapLoadedMsg{} // handle error gracefully
		}
		return roadmapLoadedMsg{roadmap: rm}
	}
}

// Update handles messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// When in sensei chat or session selector, handle text input first
		if m.state == stateSenseiChat && (msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace) {
			return m.handleChatTextInput(msg)
		}
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case roadmapLoadedMsg:
		return m.handleRoadmapLoaded(msg)

	case topicSelectedMsg:
		return m.handleTopicSelected(msg)

	case exerciseSelectedMsg:
		return m.handleExerciseSelected(msg)

	case testResultMsg:
		return m.handleTestResult(msg)

	case hintResultMsg:
		if !m.hintRequested {
			return m, nil // user navigated away, ignore
		}
		m.hintResult = msg.hint
		m.hintRequested = false
		return m, nil

	case hintErrorMsg:
		if !m.hintRequested {
			return m, nil // user navigated away, ignore
		}
		m.hintError = msg.err
		m.hintRequested = false
		return m, nil

	case chatResponseMsg:
		return m.handleChatResponse(msg)

	case chatSessionsLoadedMsg:
		return m.handleChatSessionsLoaded(msg)

	case spinner.TickMsg:
		if m.state == stateTestRunning || (m.state == stateSenseiChat && m.chatLoading) {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	default:
		return m, nil
	}
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When in session selector, handle differently
	if m.state == stateSessionSelector {
		return m.handleSessionSelectorKey(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "esc":
		return m.handleEsc()

	case "enter":
		return m.handleEnter()

	case "up", "k":
		return m.handleCursorUp()

	case "down", "j":
		return m.handleCursorDown()

	case "ctrl+t":
		return m.handleCtrlT()

	case "ctrl+h":
		return m.handleCtrlH()

	case "ctrl+g":
		return m.handleCtrlG()

	case "ctrl+n":
		if m.state == stateSenseiChat {
			return m.handleChatNew()
		}
		return m, nil

	case "ctrl+l":
		if m.state == stateSenseiChat {
			return m.handleChatList()
		}
		return m, nil

	default:
		return m, nil
	}
}

func (m Model) handleEsc() (tea.Model, tea.Cmd) {
	switch m.state {
	case stateTopicDetail:
		m.state = stateRoadmapView
		m.cursor = 0
		return m, nil
	case stateTestResults:
		m.state = stateTopicDetail
		m.cursor = 0
		return m, nil
	case stateExerciseView:
		m.state = stateTopicDetail
		m.cursor = 0
		return m, nil
	case stateHintDisplay:
		m.hintRequested = false
		m.state = m.previousView
		m.hintError = nil
		return m, nil
	case stateSenseiChat:
		// Save current session before leaving
		m.saveCurrentChatSession()
		m.chatPrunedMsg = "" // clear notification
		m.state = m.previousView
		if m.state == stateSenseiChat {
			m.state = stateRoadmapView // fallback
		}
		return m, nil
	case stateSessionSelector:
		m.state = stateSenseiChat
		m.cursor = 0
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.state {
	case stateRoadmapView:
		// Select the topic at cursor and load its exercises
		if m.cursor >= 0 && m.cursor < len(m.topics) {
			topic := m.topics[m.cursor]
			m.currentTopic = topic
			m.state = stateTopicDetail
			m.cursor = 0
			m.err = nil

			// Load exercises for this topic
			exercises, err := m.exerciseSvc.GetExercisesByTopic(topic.Slug)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.exercises = exercises
			return m, nil
		}
	case stateTopicDetail:
		// Select the exercise at cursor and generate its files
		if m.cursor >= 0 && m.cursor < len(m.exercises) {
			ref := m.exercises[m.cursor]
			m.err = nil

			ex, err := m.exerciseSvc.StartExercise(ref.Slug)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.currentExercise = ex

			// Generate exercise files in workspace
			ctx := context.Background()
			exerciseDir := m.workspacePath + "/" + ex.TopicSlug + "/" + ex.Slug
			if err := m.exerciseRepo.GenerateFiles(ctx, ex, exerciseDir); err != nil {
				// File already exists is OK — user already has them, just show as info
				if strings.Contains(err.Error(), "ya existen") {
					m.err = nil // clear, not a real error
				} else {
					m.err = err
				}
			}

			m.state = stateExerciseView
			m.cursor = 0
			return m, nil
		}
	case stateSenseiChat:
		return m.handleChatSend()
	case stateSessionSelector:
		return m.handleSessionSelect()
	}
	return m, nil
}

func (m Model) handleCursorUp() (tea.Model, tea.Cmd) {
	if m.cursor > 0 {
		m.cursor--
	}
	return m, nil
}

func (m Model) handleCursorDown() (tea.Model, tea.Cmd) {
	maxLen := m.getCursorMax()
	if m.cursor < maxLen-1 {
		m.cursor++
	}
	return m, nil
}

func (m Model) getCursorMax() int {
	switch m.state {
	case stateRoadmapView:
		return len(m.topics)
	case stateTopicDetail:
		return len(m.exercises)
	case stateSessionSelector:
		return len(m.chatSessions)
	default:
		return 0
	}
}

func (m Model) handleCtrlT() (tea.Model, tea.Cmd) {
	if m.state == stateExerciseView && m.currentExercise != nil {
		m.state = stateTestRunning
		if m.testRunner == nil {
			return m, nil
		}
		exerciseDir := m.exercisePath()
		cmd := runTestsCmd(m.testRunner, exerciseDir)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleCtrlH() (tea.Model, tea.Cmd) {
	if m.state == stateTestResults && m.testResult != nil && !m.testResult.Passed {
		m.state = stateHintDisplay
		m.previousView = stateTestResults
		m.hintRequested = true
		if m.hintSvc == nil || m.currentExercise == nil {
			m.hintError = fmt.Errorf("servicio de pistas no disponible")
			return m, nil
		}
		cmd := fetchHintCmd(m.hintSvc, m.currentExercise, m.lastTestOutput)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleRoadmapLoaded(msg roadmapLoadedMsg) (tea.Model, tea.Cmd) {
	m.roadmap = msg.roadmap
	if msg.roadmap == nil {
		return m, nil
	}
	m.phases = msg.roadmap.Phases

	// Build flat topic list
	var topics []*domain.Topic
	for _, phase := range msg.roadmap.Phases {
		topics = append(topics, phase.Topics...)
	}
	m.topics = topics
	m.state = stateRoadmapView
	return m, nil
}

func (m Model) handleTopicSelected(msg topicSelectedMsg) (tea.Model, tea.Cmd) {
	m.currentTopic = msg.topic
	m.state = stateTopicDetail
	m.cursor = 0
	return m, nil
}

func (m Model) handleExerciseSelected(msg exerciseSelectedMsg) (tea.Model, tea.Cmd) {
	m.currentExercise = msg.exercise
	m.state = stateExerciseView
	m.cursor = 0
	return m, nil
}

func (m Model) handleTestResult(msg testResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
	}
	m.testResult = msg.result
	if msg.result != nil {
		m.lastTestOutput = msg.result.Output
	}
	m.state = stateTestResults
	return m, nil
}

// --- Sensei Chat Handlers ---

// handleCtrlG transitions to sensei chat from any main view.
func (m Model) handleCtrlG() (tea.Model, tea.Cmd) {
	// Save previous view for back navigation
	if m.state != stateSenseiChat && m.state != stateSessionSelector {
		m.previousView = m.state
	}

	m.state = stateSenseiChat
	m.chatPrunedMsg = "" // clear old notification

	// If no active session, auto-create one
	if m.chatSessionID == "" {
		m.chatSessionID = chatstore.NewSessionID()
		m.chatMessages = nil
	}

	return m, nil
}

// handleChatTextInput accumulates typed characters into the input buffer.
func (m Model) handleChatTextInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.chatLoading {
		return m, nil // ignore input while loading
	}

	// Handle backspace
	if msg.Type == tea.KeyBackspace || (len(msg.Runes) == 1 && msg.Runes[0] == 127) {
		if len(m.chatInput) > 0 {
			runes := []rune(m.chatInput)
			m.chatInput = string(runes[:len(runes)-1])
		}
		return m, nil
	}

	// Accumulate runes
	for _, r := range msg.Runes {
		m.chatInput += string(r)
	}
	return m, nil
}

// handleChatSend sends the current input as a user message to the sensei.
func (m Model) handleChatSend() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.chatInput)
	if input == "" || m.chatLoading {
		return m, nil
	}

	// Add user message
	now := timeNow()
	m.chatMessages = append(m.chatMessages, chatstore.ChatMessage{
		Role:    "user",
		Content: input,
		Time:    now,
	})

	m.chatInput = ""
	m.chatLoading = true

	// Dispatch async API call
	if m.chatProvider == nil {
		// No provider — show error message
		m.chatMessages = append(m.chatMessages, chatstore.ChatMessage{
			Role:    "sensei",
			Content: "Sensei no disponible — configura GEMINI_API_KEY en .env",
			Time:    now,
		})
		m.chatLoading = false
		m.saveCurrentChatSession()
		return m, nil
	}

	cmd := m.sendChatCmd()
	return m, cmd
}

// sendChatCmd creates a command that calls SendMessage asynchronously.
func (m Model) sendChatCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		response, err := m.chatProvider.SendMessage(ctx, "", m.chatMessages)

		// If err is non-nil, the response string may also contain a user-friendly message
		if err != nil {
			if response != "" {
				return chatResponseMsg{content: response, err: err}
			}
			return chatResponseMsg{content: err.Error(), err: err}
		}

		return chatResponseMsg{content: response, err: nil}
	}
}

// handleChatResponse processes the sensei's response.
func (m Model) handleChatResponse(msg chatResponseMsg) (tea.Model, tea.Cmd) {
	if msg.content == "" && msg.err != nil {
		msg.content = fmt.Sprintf("Error: %v", msg.err)
	}

	m.chatMessages = append(m.chatMessages, chatstore.ChatMessage{
		Role:    "sensei",
		Content: msg.content,
		Time:    timeNow(),
	})
	m.chatLoading = false

	// Auto-save session
	m.saveCurrentChatSession()

	return m, nil
}

// handleChatNew saves the current session and starts a new one.
func (m Model) handleChatNew() (tea.Model, tea.Cmd) {
	m.saveCurrentChatSession()
	m.chatSessionID = chatstore.NewSessionID()
	m.chatMessages = nil
	m.chatInput = ""
	m.chatLoading = false
	m.chatPrunedMsg = ""
	return m, nil
}

// handleChatList loads the session list and transitions to the selector.
func (m Model) handleChatList() (tea.Model, tea.Cmd) {
	if m.chatStore == nil {
		return m, nil
	}

	// Load sessions from the store synchronously for simplicity
	sessions, err := m.chatStore.ListSessions()
	if err != nil {
		// Silently fail — show empty list
		m.chatSessions = nil
	} else {
		m.chatSessions = sessions
	}

	m.state = stateSessionSelector
	m.cursor = 0
	return m, nil
}

// handleSessionSelectorKey handles key presses in the session selector overlay.
func (m Model) handleSessionSelectorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.handleEsc()
	case "enter":
		return m.handleSessionSelect()
	case "up", "k":
		return m.handleCursorUp()
	case "down", "j":
		return m.handleCursorDown()
	}
	return m, nil
}

// handleSessionSelect loads the selected session.
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
	m.chatPrunedMsg = ""
	m.state = stateSenseiChat
	m.cursor = 0

	return m, nil
}

// handleChatSessionsLoaded processes the session list after loading.
func (m Model) handleChatSessionsLoaded(msg chatSessionsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.chatSessions = nil
	} else {
		m.chatSessions = msg.sessions
	}
	return m, nil
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

// timeNow returns the current time. Exists for testability.
var timeNow = func() time.Time { return time.Now() }

// View renders the current TUI view based on state.
func (m Model) View() string {
	switch m.state {
	case stateRoadmapView:
		return m.viewRoadmap()
	case stateTopicDetail:
		return m.viewTopicDetail()
	case stateExerciseView:
		return m.viewExercise()
	case stateTestRunning:
		return m.viewTestRunning()
	case stateTestResults:
		return m.viewTestResults()
	case stateHintDisplay:
		return m.viewHintDisplay()
	case stateSenseiChat:
		return m.viewSenseiChat()
	case stateSessionSelector:
		return m.viewSessionSelector()
	default:
		return "Cargando..."
	}
}
