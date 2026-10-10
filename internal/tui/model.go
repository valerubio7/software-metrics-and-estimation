package tui

import (
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
	b.WriteString(header())
	b.WriteString("\n\n")
	if m.APIErr != nil {
		b.WriteString(errorBannerStyle.Render("API no disponible: " + m.APIErr.Error()))
		b.WriteString("\n")
		b.WriteString(errorHintStyle.Render("Revisá API_URL o levantá la API; la navegación sigue disponible."))
		b.WriteString("\n\n")
	} else {
		b.WriteString(statusOkStyle.Render("● API conectada · " + m.APIURL))
		b.WriteString("\n\n")
	}
	if m.Screen != "menu" {
		b.WriteString(screenTitleStyle.Render(m.Choices[m.Cursor]))
		b.WriteString("  ")
		b.WriteString(badgeStyle.Render("Disponible en " + placeholderRelease(m.Cursor)))
		b.WriteString("\n\n")
		b.WriteString(hintStyle.Render("Pulsa "))
		b.WriteString(hintKeyStyle.Render("esc"))
		b.WriteString(hintStyle.Render(" para volver, "))
		b.WriteString(hintKeyStyle.Render("q"))
		b.WriteString(hintStyle.Render(" para salir."))
		return b.String()
	}
	for i, choice := range m.Choices {
		if m.Cursor == i {
			b.WriteString(cursorStyle.Render("▸") + " " + selectedStyle.Render(choice))
		} else {
			b.WriteString("  " + itemStyle.Render(choice))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Mover "))
	b.WriteString(hintKeyStyle.Render("↑/↓"))
	b.WriteString(hintStyle.Render(" · entrar "))
	b.WriteString(hintKeyStyle.Render("enter"))
	b.WriteString(hintStyle.Render(" · salir "))
	b.WriteString(hintKeyStyle.Render("q"))
	return b.String()
}

func header() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Software Metrics & Estimation"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Sprint 2 · La Interfaz · TUI base"))
	b.WriteString("\n")
	b.WriteString(ruleStyle.Render("────────────────────────────────"))
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
