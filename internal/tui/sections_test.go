package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestSectionLastStepDoesNotShowNotice covers TUI-13: every section kind has
// its own flow, so choosing a row at a step without a next one stays put
// instead of rendering the former TUI-06 notice.
func TestSectionLastStepDoesNotShowNotice(t *testing.T) {
	m := NewModel("http://127.0.0.1:1")
	m.activeProjectID = "p1"
	m.Screen = "section"
	m.section.kind = sectionDefects
	m.section.step = sectionStepSprints
	m.section.selected = selectorItem{ID: "s1", Label: "Login"}
	m.section.selector.setItems([]selectorItem{{ID: "sp1", Label: "Sprint 1"}})

	next, cmd := m.updateSection(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Errorf("choosing the last step must not issue a command")
	}
	if next.Screen != "section" || next.section.step != sectionStepSprints {
		t.Errorf("Screen/step = %q/%d, want section/%d", next.Screen, next.section.step, sectionStepSprints)
	}
	view := next.View()
	for _, banned := range []string{"Próximamente", "TUI-09", "No se guardó ningún dato"} {
		if strings.Contains(view, banned) {
			t.Errorf("view shows the removed notice text %q", banned)
		}
	}
}
