package tui

import (
	"strings"
	"testing"
)

func TestViewToolRunning_ShowsSpinner(t *testing.T) {
	m := newModelTest()
	m.state = stateToolRunning

	view := m.View()
	if !strings.Contains(view, "Sensei está trabajando") {
		t.Error("tool running view should show 'Sensei está trabajando'")
	}
}

func TestViewToolRunning_ShowsStatus(t *testing.T) {
	m := newModelTest()
	m.state = stateToolRunning
	m.toolStatus = "Creando archivo de ejercicio..."

	view := m.View()
	if !strings.Contains(view, "Creando archivo de ejercicio") {
		t.Error("tool running view should show current tool status")
	}
}

func TestViewToolRunning_ShowsDefaultText_WhenStatusEmpty(t *testing.T) {
	m := newModelTest()
	m.state = stateToolRunning
	m.toolStatus = ""

	view := m.View()
	if !strings.Contains(view, "Ejecutando") {
		t.Error("tool running view should show 'Ejecutando...' when no status set")
	}
}

func TestViewSenseiChat_Renders(t *testing.T) {
	m := newModelTest()
	m.state = stateSenseiChat

	view := m.View()
	if view == "" {
		t.Error("sensei chat view should not be empty")
	}
	if !strings.Contains(view, "Sensei Chat") {
		t.Error("sensei chat should contain 'Sensei Chat' in title")
	}
}

func TestViewSessionSelector_Renders(t *testing.T) {
	m := newModelTest()
	m.state = stateSessionSelector

	view := m.View()
	if view == "" {
		t.Error("session selector view should not be empty")
	}
	if !strings.Contains(view, "Sesiones") {
		t.Error("session selector should contain 'Sesiones' in title")
	}
}

func TestViewDefault_ShowsLoading(t *testing.T) {
	m := newModelTest()
	m.state = 999 // invalid state

	view := m.View()
	if view != "Cargando..." {
		t.Errorf("default view = %q, want %q", view, "Cargando...")
	}
}
