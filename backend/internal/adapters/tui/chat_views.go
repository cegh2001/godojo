package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// --- Chat styles ---

var (
	senseiStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))  // blue
	userStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // yellow
	infoStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")) // gray
	pruneStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // orange
	inputStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252")) // light gray
)

// viewSenseiChat renders the main chat view.
// Clean header with session name, messages area, compact input, and help bar.
func (m Model) viewSenseiChat() string {
	// Reserve bottom 3 lines for input + help bar
	fixedFooter := 3
	messagesHeight := m.height - fixedFooter
	if messagesHeight < 3 {
		messagesHeight = 3
	}

	var sb strings.Builder

	// Header: session name + notification
	sessionName := m.truncateChatSessionName()
	sb.WriteString(titleStyle.Render("🤖 Sensei Chat — " + sessionName) + "\n")

	// Notification bar (pruned session notification)
	if m.chatPrunedMsg != "" {
		sb.WriteString(pruneStyle.Render("🗑️ Sesión más antigua eliminada (máx 10): "+m.chatPrunedMsg) + "\n")
	}

	// Messages area
	messagesLines := m.renderChatMessages(messagesHeight - 2) // -2 for header lines
	sb.WriteString(messagesLines)

	// Fill remaining space
	usedLines := strings.Count(messagesLines, "\n")
	for i := usedLines; i < messagesHeight-2; i++ {
		sb.WriteString("\n")
	}

	// Separator line (thin)
	sb.WriteString(helpStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Input area: just "> {input}"
	inputDisplay := m.chatInput
	if m.chatLoading {
		inputDisplay = "..." // compact while waiting
	}
	sb.WriteString(inputStyle.Render("> " + inputDisplay))

	// Help bar
	sb.WriteString("\n" + helpStyle.Render("Enviar: Enter | Nueva: Ctrl+N | Sesiones: Ctrl+L | Volver: Esc"))

	return sb.String()
}

// renderChatMessages renders the scrollable messages pane.
func (m Model) renderChatMessages(maxLines int) string {
	var sb strings.Builder

	// Welcome message when session is empty
	if len(m.chatMessages) == 0 {
		sb.WriteString(senseiStyle.Render("🤖 Sensei: ") + "¡Hola! Soy tu sensei de Go. ¿En qué puedo ayudarte?" + "\n")
		return sb.String()
	}

	// Calculate how many messages from the end we can show
	var linesToRender []string
	lineCount := 0

	// If loading, show animated "pensando..." message
	if m.chatLoading {
		loadingLine := senseiStyle.Render("🤖 Sensei: ") + m.spinner.View() + " pensando..."
		linesToRender = append([]string{loadingLine}, linesToRender...)
		lineCount++
	}

	// Render messages from newest to oldest until we fill the space
	for i := len(m.chatMessages) - 1; i >= 0 && lineCount < maxLines; i-- {
		msg := m.chatMessages[i]
		prefix := "🤖 Sensei: "
		style := senseiStyle
		if msg.Role == "user" {
			prefix = "Tú:       "
			style = userStyle
		}

		// Truncate long messages for display
		content := msg.Content
		maxContentLen := m.width - 15 // leave room for prefix
		if maxContentLen < 20 {
			maxContentLen = 20
		}
		if len(content) > maxContentLen {
			content = content[:maxContentLen] + "..."
		}

		line := style.Render(prefix) + content
		linesToRender = append([]string{line}, linesToRender...)
		lineCount++
	}

	for _, line := range linesToRender {
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

// truncateChatSessionName returns a short display name for the current session.
func (m Model) truncateChatSessionName() string {
	for _, s := range m.chatSessions {
		if s.ID == m.chatSessionID {
			name := s.Name
			if len([]rune(name)) > 30 {
				return string([]rune(name)[:30]) + "..."
			}
			return name
		}
	}
	return "Nuevo chat"
}

// viewSessionSelector renders the session list overlay.
// Pressing Ctrl+L shows this overlay over the current view.
func (m Model) viewSessionSelector() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("📋 Sesiones de chat") + "\n")
	sb.WriteString(helpStyle.Render("j/k: navegar  enter: seleccionar  esc: volver") + "\n\n")

	if len(m.chatSessions) == 0 {
		sb.WriteString(infoStyle.Render("No hay sesiones guardadas.") + "\n")
		return sb.String()
	}

	for i, session := range m.chatSessions {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("→ ")
		}

		// Show name and timestamp
		timeStr := session.UpdatedAt.Format("02/01 15:04")
		msgCount := len(session.Messages)
		sb.WriteString(fmt.Sprintf("%s%s  %s  (%d mensajes)\n",
			cursor,
			truncateForList(session.Name, 40),
			infoStyle.Render(timeStr),
			msgCount,
		))
	}

	sb.WriteString("\n" + helpStyle.Render("enter: cargar sesión  esc: volver"))
	return sb.String()
}

// truncateForList truncates a string for display in a list.
func truncateForList(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
