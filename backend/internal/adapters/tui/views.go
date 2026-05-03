package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// --- Styles ---

var (
	// spinnerStyle is used for the spinner during tool execution.
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	cursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	failStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	hintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

// viewToolRunning renders the spinner view during agent loop tool execution.
func (m Model) viewToolRunning() string {
	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Sensei está trabajando...") + "\n\n")
	sb.WriteString(m.spinner.View())
	if m.toolStatus != "" {
		sb.WriteString("  " + helpStyle.Render(m.toolStatus) + "\n")
	} else {
		sb.WriteString("  " + helpStyle.Render("Ejecutando...") + "\n")
	}
	return sb.String()
}
