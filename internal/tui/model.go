package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Model is the BubbleTea model for the TUI base navigation.
type Model struct {
	APIURL  string
	Choices []string
	Cursor  int
	Screen  string
	APIErr  error
}

// NewModel creates the base navigation model.
func NewModel(apiURL string) Model {
	return Model{
		APIURL:  apiURL,
		Choices: []string{"Proyectos", "Backlog", "Sprints", "Salir"},
		Cursor:  0,
		Screen:  "menu",
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. Navigation is read-only: it never emits mutating requests.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyUp:
			if m.Screen == "menu" && m.Cursor > 0 {
				m.Cursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.Screen == "menu" && m.Cursor < len(m.Choices)-1 {
				m.Cursor++
			}
			return m, nil
		case tea.KeyEnter:
			if m.Screen != "menu" {
				return m, nil
			}
			switch m.Cursor {
			case 0:
				m.Screen = "projects"
			case 1:
				m.Screen = "backlog"
			case 2:
				m.Screen = "sprints"
			default:
				return m, tea.Quit
			}
			return m, nil
		case tea.KeyEsc:
			m.Screen = "menu"
			return m, nil
		case tea.KeyRunes:
			if len(msg.Runes) == 1 && (msg.Runes[0] == 'q' || msg.Runes[0] == 'Q') {
				return m, tea.Quit
			}
			return m, nil
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	var b strings.Builder
	if m.APIErr != nil {
		fmt.Fprintf(&b, "API no disponible: %s\n\n", m.APIErr.Error())
	}
	if m.Screen != "menu" {
		fmt.Fprintf(&b, "%s (Disponible en %s)\n\nPulsa esc para volver, q para salir.", m.Choices[m.Cursor], placeholderRelease(m.Cursor))
		return b.String()
	}
	b.WriteString("Software Metrics & Estimation\n\n")
	for i, choice := range m.Choices {
		cursor := " "
		if m.Cursor == i {
			cursor = ">"
		}
		fmt.Fprintf(&b, "%s %s\n", cursor, choice)
	}
	b.WriteString("\n↑/↓ mover · enter entrar · q salir\n")
	return b.String()
}

func placeholderRelease(cursor int) string {
	switch cursor {
	case 0:
		return "TUI-02"
	case 1:
		return "TUI-03"
	default:
		return "TUI-04/TUI-05"
	}
}
