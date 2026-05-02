package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"godojo/internal/core/domain"
)

// --- Styles (minimal MVP) ---

var (
	// spinnerStyle is used for the spinner during test execution.
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	cursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	failStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	hintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

// --- Views ---

func (m Model) viewRoadmap() string {
	var sb strings.Builder
	sb.WriteString(titleStyle.Render("GoDojo 🥋") + "\n")
	sb.WriteString(helpStyle.Render("j/k navegar  enter seleccionar  q salir") + "\n\n")

	for i, topic := range m.topics {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("→ ")
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", cursor, topic.Title))
	}

	sb.WriteString("\n" + helpStyle.Render("enter: ver tema  q: salir"))
	return sb.String()
}

func (m Model) viewTopicDetail() string {
	var sb strings.Builder
	topic := m.currentTopic
	if topic == nil {
		return "Cargando tema..."
	}

	sb.WriteString(titleStyle.Render(topic.Title) + "\n")
	sb.WriteString(helpStyle.Render(topic.Description) + "\n\n")

	if m.err != nil {
		sb.WriteString(failStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n")
	}

	sb.WriteString(helpStyle.Render("Ejercicios:") + "\n")

	if len(m.exercises) == 0 {
		sb.WriteString(helpStyle.Render("  (no hay ejercicios disponibles para este tema)\n"))
	}

	for i, ex := range m.exercises {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("→ ")
		}
		diff := difficultyDots(ex.Difficulty)
		sb.WriteString(fmt.Sprintf("%s%s %s\n", cursor, ex.Title, diff))
	}

	sb.WriteString("\n" + helpStyle.Render("enter: ver ejercicio  esc: volver"))
	return sb.String()
}

func (m Model) viewExercise() string {
	var sb strings.Builder
	ex := m.currentExercise
	if ex == nil {
		return "Cargando ejercicio..."
	}

	sb.WriteString(titleStyle.Render(ex.Title) + "\n\n")

	if m.err != nil {
		sb.WriteString(failStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n")
	}

	sb.WriteString(helpStyle.Render(fmt.Sprintf("📁 Archivos en: %s\n\n", m.exercisePath())))
	sb.WriteString(helpStyle.Render("ctrl+t: ejecutar tests  ctrl+h: pedir pista  esc: volver") + "\n\n")
	sb.WriteString(helpStyle.Render("Abrí ejercicio.go en tu editor, completá los // TODO y volvé acá."))

	return sb.String()
}

// exercisePath returns the full path to the current exercise's workspace directory.
func (m Model) exercisePath() string {
	if m.currentExercise == nil {
		return m.workspacePath
	}
	return m.workspacePath + "/" + m.currentExercise.TopicSlug + "/" + m.currentExercise.Slug
}

func (m Model) viewTestRunning() string {
	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Ejecutando tests...") + "\n\n")
	sb.WriteString(m.spinner.View())
	sb.WriteString("  Ejecutando tests...\n")
	return sb.String()
}

func (m Model) viewTestResults() string {
	var sb strings.Builder
	tr := m.testResult
	if tr == nil {
		return "Sin resultados de tests."
	}

	if tr.Passed {
		sb.WriteString(successStyle.Render("✓ Tests pasaron!") + "\n")
	} else {
		sb.WriteString(failStyle.Render("✗ Tests fallaron") + "\n")
	}

	sb.WriteString("\n" + tr.Output + "\n")

	if !tr.Passed {
		sb.WriteString("\n" + hintStyle.Render("ctrl+h: pedir una pista del sensei") + "\n")
	}
	sb.WriteString(helpStyle.Render("esc: volver"))
	return sb.String()
}

func (m Model) viewHintDisplay() string {
	var sb strings.Builder
	sb.WriteString(hintStyle.Render("🤔 Sensei dice:") + "\n\n")

	if m.hintResult != nil {
		sb.WriteString(m.hintResult.Content + "\n")
	} else if m.hintError != nil {
		sb.WriteString(failStyle.Render("El sensei no está disponible ahora. Intentá de nuevo en unos segundos.") + "\n")
	} else {
		sb.WriteString("Consultando al sensei...\n")
	}

	sb.WriteString("\n" + helpStyle.Render("esc: volver"))
	return sb.String()
}

// difficultyDots returns a visual representation of difficulty using dots.
func difficultyDots(d domain.Difficulty) string {
	switch d {
	case domain.DifficultyEasy:
		return successStyle.Render("●○○")
	case domain.DifficultyMedium:
		return hintStyle.Render("●●○")
	case domain.DifficultyHard:
		return failStyle.Render("●●●")
	default:
		return helpStyle.Render("○○○")
	}
}
