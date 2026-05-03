package tui

import (
	"context"
	"fmt"
	"strings"

	"godojo/internal/core/domain"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state == stateSessionSelector {
		return m.handleSessionSelectorKey(msg)
	}

	if m.state == stateSenseiChat {
		switch msg.Type {
		case tea.KeyUp:
			m = m.shiftChatScroll(1)
			return m, nil
		case tea.KeyDown:
			m = m.shiftChatScroll(-1)
			return m, nil
		case tea.KeyHome:
			m.chatScroll = m.maxChatScroll(m.chatMessageAreaHeight())
			return m, nil
		case tea.KeyEnd:
			m.chatScroll = 0
			return m, nil
		case tea.KeyPgUp:
			m = m.shiftChatScroll(m.chatPageScrollStep())
			return m, nil
		case tea.KeyPgDown:
			m = m.shiftChatScroll(-m.chatPageScrollStep())
			return m, nil
		}

		switch msg.String() {
		case "pgup", "pageup":
			m = m.shiftChatScroll(m.chatPageScrollStep())
			return m, nil
		case "pgdown", "pagedown":
			m = m.shiftChatScroll(-m.chatPageScrollStep())
			return m, nil
		}
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
		m.saveCurrentChatSession()
		m.chatPrunedMsg = ""
		m.state = m.previousView
		if m.state == stateSenseiChat {
			m.state = stateRoadmapView
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
		if m.cursor >= 0 && m.cursor < len(m.topics) {
			topic := m.topics[m.cursor]
			m.currentTopic = topic
			m.state = stateTopicDetail
			m.cursor = 0
			m.err = nil

			exercises, err := m.exerciseSvc.GetExercisesByTopic(topic.Slug)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.exercises = exercises
			return m, nil
		}
	case stateTopicDetail:
		if m.cursor >= 0 && m.cursor < len(m.exercises) {
			ref := m.exercises[m.cursor]
			m.err = nil

			ex, err := m.exerciseSvc.StartExercise(ref.Slug)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.currentExercise = ex

			exerciseDir := m.workspacePath + "/" + ex.TopicSlug + "/" + ex.Slug
			if err := m.exerciseRepo.GenerateFiles(context.Background(), ex, exerciseDir); err != nil {
				if !strings.Contains(err.Error(), "ya existen") {
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
