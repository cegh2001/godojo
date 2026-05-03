package tui

import (
	"context"
	"time"

	"godojo/internal/adapters/chatstore"
	"godojo/internal/core/domain"
	"godojo/internal/core/ports"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// tuiState represents the current view state in the TUI state machine.
type tuiState int

const (
	stateRoadmapView     tuiState = iota // browsing phases/topics
	stateTopicDetail                     // viewing topic info + exercises
	stateExerciseView                    // viewing exercise description
	stateTestRunning                     // spinner while tests run
	stateTestResults                     // showing test output
	stateHintDisplay                     // showing Socratic hint
	stateSenseiChat                      // chat with AI sensei
	stateSessionSelector                 // session list overlay
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
	chatProvider  chatProvider
	chatStore     *chatstore.ChatStore
	chatSessions  []chatstore.ChatSession
	chatMessages  []chatstore.ChatMessage
	chatInput     string
	chatLoading   bool
	chatScroll    int
	chatSessionID string
	chatPrunedMsg string // notification about pruned session
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
