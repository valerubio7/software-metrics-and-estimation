package tui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

// TestFooterWrapsInsteadOfClipping guards the narrow-terminal layout: Bubble Tea
// clips lines wider than the terminal, which used to hide the last shortcuts.
func TestFooterWrapsInsteadOfClipping(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")
	m, _ = apply(m, tea.WindowSizeMsg{Width: 44, Height: 24})
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeyDown)
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter}) // Sprints -> project picker
	view := m.View()

	for _, want := range []string{"Esc", "volver"} {
		if !strings.Contains(view, want) {
			t.Fatalf("narrow footer lost %q: %q", want, view)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if w := lipgloss.Width(stripANSI(line)); w > 44 {
			t.Errorf("line width = %d, want <= 44: %q", w, line)
		}
	}
}
