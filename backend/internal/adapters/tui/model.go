package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"godojo/internal/core/domain"
	"godojo/internal/core/ports"
)

// tuiState represents the current view state in the TUI state machine.
type tuiState int

const (
	stateRoadmapView  tuiState = iota // browsing phases/topics
	stateTopicDetail                  // viewing topic info + exercises
	stateExerciseView                 // viewing exercise description
	stateTestRunning                  // spinner while tests run
	stateTestResults                  // showing test output
	stateHintDisplay                  // showing Socratic hint
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
	cursor     int // selected item index
	testResult *domain.TestResult
	hintResult *domain.Hint
	hintError  error
	spinner    spinner.Model
	width      int
	height     int
	err        error

	// Content lists for views
	phases    []*domain.Phase
	topics    []*domain.Topic
	exercises []*domain.ExerciseRef

	// For command execution
	testRunner     ports.TestRunner
	workspacePath  string
	lastTestOutput string
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

// NewModel creates a new TUI Model with the given services, test runner, and workspace.
func NewModel(
	roadmapSvc roadmapService,
	exerciseSvc exerciseService,
	progressSvc progressService,
	hintSvc hintService,
	testRunner ports.TestRunner,
	workspacePath string,
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
		workspacePath: workspacePath,
		spinner:       sp,
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
		m.hintResult = msg.hint
		m.previousView = m.state
		m.state = stateHintDisplay
		return m, nil

	case hintErrorMsg:
		m.hintError = msg.err
		m.previousView = m.state
		m.state = stateHintDisplay
		return m, nil

	case spinner.TickMsg:
		if m.state == stateTestRunning {
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
		m.state = m.previousView
		m.hintError = nil
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
		// Select the exercise at cursor
		if m.cursor >= 0 && m.cursor < len(m.exercises) {
			ref := m.exercises[m.cursor]
			ex, err := m.exerciseSvc.StartExercise(ref.Slug)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.currentExercise = ex
			m.state = stateExerciseView
			m.cursor = 0
			return m, nil
		}
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
		cmd := runTestsCmd(m.testRunner, m.workspacePath)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleCtrlH() (tea.Model, tea.Cmd) {
	if m.state == stateTestResults && m.testResult != nil && !m.testResult.Passed {
		m.state = stateHintDisplay
		m.previousView = stateTestResults
		if m.hintSvc == nil || m.currentExercise == nil {
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
	default:
		return "Cargando..."
	}
}
