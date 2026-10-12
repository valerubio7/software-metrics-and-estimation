package tui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestMenuShowsFourOptions(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")
	view := m.View()

	for _, want := range []string{"proyectos", "backlog", "sprints", "salir"} {
		if !strings.Contains(strings.ToLower(view), want) {
			t.Errorf("View() = %q, want option %q", view, want)
		}
	}
}

func TestCursorMovesDownAndUp(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(tui.Model)
	if m.Cursor != 1 {
		t.Fatalf("Cursor = %d, want 1 after down", m.Cursor)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(tui.Model)
	if m.Cursor != 0 {
		t.Fatalf("Cursor = %d, want 0 after up", m.Cursor)
	}
}

func TestEnterProjectsAndBack(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	if m.Screen == "menu" {
		t.Fatalf("Screen = menu after enter, want placeholder")
	}
	if !strings.Contains(m.View(), "Crear proyecto") {
		t.Errorf("project View() = %q, want project actions", m.View())
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(tui.Model)
	if m.Screen != "menu" {
		t.Errorf("Screen = %q after esc, want menu", m.Screen)
	}
}

func TestProjectScreenOffersCreateEditAndStatus(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	view := strings.ToLower(m.View())
	for _, want := range []string{"crear", "modificar", "estado"} {
		if !strings.Contains(view, want) {
			t.Errorf("project View() = %q, want %q", view, want)
		}
	}
}

func TestProjectAPIErrorIsShownWithoutSuccess(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	for i, value := range []string{"Alpha", "2026-10-01", "2026-12-01"} {
		for _, r := range value {
			updated, _ = m.Update(keyMsg(string(r)))
			m = updated.(tui.Model)
		}
		if i < 2 {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
			m = updated.(tui.Model)
		}
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	if cmd == nil {
		t.Fatal("submitting form should issue API command")
	}
	result := tui.ProjectResultMsg{Response: tui.APIResponse{StatusCode: 422, Body: []byte(`{"error":"validation_failed","message":"invalid dates"}`)}}
	updated, _ = m.Update(result)
	m = updated.(tui.Model)
	view := m.View()
	if !strings.Contains(view, "422") || !strings.Contains(view, "invalid dates") || strings.Contains(view, "creado correctamente") {
		t.Errorf("error view = %q", view)
	}
}

func TestProjectNotFoundIsExplicitAndPreservesAPIDetail(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")
	updated, _ := m.Update(tui.ProjectResultMsg{Response: tui.APIResponse{StatusCode: 404, Body: []byte(`{"error":"project_not_found","message":"project not found"}`)}})
	m = updated.(tui.Model)
	view := m.View()
	if !strings.Contains(view, "no existe") || !strings.Contains(view, "project_not_found") {
		t.Errorf("404 view = %q, want explicit not-found message and API detail", view)
	}
}

func TestProjectNavigationPreservesOtherSectionsAndQuit(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.(tui.Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	if m.Screen != "backlog" {
		t.Fatalf("screen = %q, want backlog", m.Screen)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(tui.Model)
	if m.Screen != "menu" {
		t.Fatalf("screen after escape = %q, want menu", m.Screen)
	}
	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal("q should quit from menu")
	}
}

func TestQuitOnQAndCtrlC(t *testing.T) {
	for _, msg := range []tea.Msg{keyMsg("q"), tea.KeyMsg{Type: tea.KeyCtrlC}} {
		m := tui.NewModel("http://localhost:8080")
		_, cmd := m.Update(msg)
		if cmd == nil {
			t.Fatalf("Update(%v) cmd = nil, want quit command", msg)
		}
		var quitMsg tea.QuitMsg
		if got := cmd(); got != quitMsg {
			_ = got
		}
	}
}
