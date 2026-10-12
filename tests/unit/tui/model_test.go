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

func TestEnterPlaceholderAndBack(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	if m.Screen == "menu" {
		t.Fatalf("Screen = menu after enter, want placeholder")
	}
	if !strings.Contains(m.View(), "TUI-0") {
		t.Errorf("placeholder View() = %q, want TUI-0x reference", m.View())
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(tui.Model)
	if m.Screen != "menu" {
		t.Errorf("Screen = %q after esc, want menu", m.Screen)
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
