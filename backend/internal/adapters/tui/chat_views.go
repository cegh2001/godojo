package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
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
	messagesHeight := m.chatMessageAreaHeight()

	var sb strings.Builder

	// Header: session name + notification
	sessionName := m.truncateChatSessionName()
	sb.WriteString(titleStyle.Render("🤖 Sensei Chat — "+sessionName) + "\n")

	// Notification bar (pruned session notification)
	if m.chatPrunedMsg != "" {
		sb.WriteString(pruneStyle.Render("🗑️ Sesión más antigua eliminada (máx 10): "+m.chatPrunedMsg) + "\n")
	}

	// Messages area
	messagesLines := m.renderChatMessages(messagesHeight)
	sb.WriteString(messagesLines)

	// Fill remaining space
	usedLines := strings.Count(messagesLines, "\n")
	for i := usedLines; i < messagesHeight; i++ {
		sb.WriteString("\n")
	}

	// Separator line (thin)
	sb.WriteString(helpStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Input area with a clear speaker label.
	sb.WriteString(m.renderChatInputLine())
	// Status line: show loading or scroll context.
	sb.WriteString("\n" + m.renderChatStatusLine())

	// Help bar
	sb.WriteString("\n" + helpStyle.Render("Ctrl+N: nuevo chat | Ctrl+L: sesiones | Esc: volver"))

	return sb.String()
}

// renderChatMessages renders the scrollable messages pane.
func (m Model) renderChatMessages(maxLines int) string {
	var sb strings.Builder
	lines := m.buildChatTranscriptLines()

	if len(lines) == 0 {
		return ""
	}

	if maxLines < 1 {
		maxLines = 1
	}

	scroll := m.clampChatScroll(maxLines)
	start := len(lines) - maxLines - scroll
	if start < 0 {
		start = 0
	}
	end := start + maxLines
	if end > len(lines) {
		end = len(lines)
	}

	for _, line := range lines[start:end] {
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

func (m Model) buildChatTranscriptLines() []string {
	if len(m.chatMessages) == 0 {
		return []string{senseiStyle.Render("🤖 Sensei: ") + "¡Hola! Soy tu sensei de Go. ¿En qué puedo ayudarte?"}
	}

	lines := make([]string, 0, len(m.chatMessages)*2)
	for _, msg := range m.chatMessages {
		lines = append(lines, renderWrappedChatMessage(msg.Role, msg.Content, m.width)...)
	}

	return lines
}

func (m Model) chatHeaderLines() int {
	if m.chatPrunedMsg != "" {
		return 2
	}
	return 1
}

func (m Model) chatFooterLines() int {
	return 4
}

func (m Model) chatMessageAreaHeight() int {
	height := m.height - m.chatHeaderLines() - m.chatFooterLines()
	if height < 1 {
		height = 1
	}
	return height
}

func (m Model) chatPageScrollStep() int {
	step := m.chatMessageAreaHeight() - 1
	if step < 1 {
		step = 1
	}
	return step
}

func (m Model) maxChatScroll(maxLines int) int {
	totalLines := len(m.buildChatTranscriptLines())
	if totalLines <= maxLines {
		return 0
	}
	return totalLines - maxLines
}

func (m Model) clampChatScroll(maxLines int) int {
	if m.chatScroll < 0 {
		return 0
	}
	maxScroll := m.maxChatScroll(maxLines)
	if m.chatScroll > maxScroll {
		return maxScroll
	}
	return m.chatScroll
}

func (m Model) shiftChatScroll(delta int) Model {
	if delta == 0 {
		return m
	}

	next := m.chatScroll + delta
	if next < 0 {
		next = 0
	}

	maxScroll := m.maxChatScroll(m.chatMessageAreaHeight())
	if next > maxScroll {
		next = maxScroll
	}

	m.chatScroll = next
	return m
}

func (m Model) renderChatInputLine() string {
	prefix := userStyle.Render("Tú: ")
	if m.chatLoading {
		return prefix + infoStyle.Render("enviando...")
	}

	if m.chatInput == "" {
		return prefix + infoStyle.Render("escribí tu mensaje")
	}

	return prefix + inputStyle.Render(m.chatInput)
}

func (m Model) renderChatStatusLine() string {
	if m.chatLoading {
		return spinnerStyle.Render("🤖 Sensei pensando...")
	}

	scroll := m.clampChatScroll(m.chatMessageAreaHeight())
	maxScroll := m.maxChatScroll(m.chatMessageAreaHeight())
	if maxScroll > 0 && scroll > 0 {
		return infoStyle.Render(fmt.Sprintf("Viendo mensajes anteriores (%d/%d) · End: volver al final", scroll, maxScroll))
	}

	return infoStyle.Render("Enter: enviar | ↑/↓: scroll | PgUp/PgDn: salto")
}

func renderWrappedChatMessage(role, content string, width int) []string {
	prefix := "🤖 Sensei: "
	style := senseiStyle
	if role == "user" {
		prefix = "Tú:       "
		style = userStyle
	}

	prefixWidth := runewidth.StringWidth(prefix)
	contentWidth := width - prefixWidth
	if contentWidth < 1 {
		contentWidth = 1
	}

	wrappedLines := wrapTextByWidth(content, contentWidth)
	if len(wrappedLines) == 0 {
		wrappedLines = []string{""}
	}

	lines := make([]string, 0, len(wrappedLines))
	lines = append(lines, style.Render(prefix)+wrappedLines[0])
	indent := strings.Repeat(" ", prefixWidth)
	for _, line := range wrappedLines[1:] {
		lines = append(lines, indent+line)
	}

	return lines
}

func wrapTextByWidth(text string, width int) []string {
	if width < 1 {
		return []string{text}
	}

	var lines []string
	for _, rawLine := range strings.Split(text, "\n") {
		if rawLine == "" {
			lines = append(lines, "")
			continue
		}

		words := strings.Fields(rawLine)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}

		current := words[0]
		currentWidth := runewidth.StringWidth(current)
		for _, word := range words[1:] {
			for _, piece := range splitWordByWidth(word, width) {
				pieceWidth := runewidth.StringWidth(piece)
				if currentWidth == 0 {
					current = piece
					currentWidth = pieceWidth
					continue
				}

				if currentWidth+1+pieceWidth <= width {
					current += " " + piece
					currentWidth += 1 + pieceWidth
					continue
				}

				lines = append(lines, current)
				current = piece
				currentWidth = pieceWidth
			}
		}

		if current != "" {
			lines = append(lines, current)
		}
	}

	return lines
}

func splitWordByWidth(word string, width int) []string {
	if runewidth.StringWidth(word) <= width {
		return []string{word}
	}

	var pieces []string
	var current strings.Builder
	currentWidth := 0
	for _, r := range word {
		runeWidth := runewidth.RuneWidth(r)
		if currentWidth+runeWidth > width && current.Len() > 0 {
			pieces = append(pieces, current.String())
			current.Reset()
			currentWidth = 0
		}

		current.WriteRune(r)
		currentWidth += runeWidth
	}

	if current.Len() > 0 {
		pieces = append(pieces, current.String())
	}

	return pieces
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
