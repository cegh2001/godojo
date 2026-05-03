package tui

import (
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
	case "esc":
		return m.handleEsc()
	case "enter":
		return m.handleEnter()
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
	case stateSenseiChat:
		// Esc from chat → show sessions (then Esc again goes back to chat)
		return m.handleChatList()
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
	case stateSenseiChat:
		return m.handleChatSend()
	case stateSessionSelector:
		return m.handleSessionSelect()
	}
	return m, nil
}

func (m Model) getCursorMax() int {
	switch m.state {
	case stateSessionSelector:
		return len(m.chatSessions)
	default:
		return 0
	}
}
