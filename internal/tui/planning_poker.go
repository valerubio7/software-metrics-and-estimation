package tui

import (
	"context"
	"encoding/json"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

var pokerCards = []string{"0", "1", "2", "3", "5", "8", "13", "20", "40", "100", "?"}
var pokerEstimates = []int{0, 1, 2, 3, 5, 8, 13, 20, 40, 100}

func (m Model) viewPokerParticipants() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Planning Poker · Participantes") + "\nElegí participantes con Space y confirmá con Enter.\n\n")
	if m.section.selector.err != "" {
		b.WriteString(m.errorBlock(m.section.selector.err))
		return b.String()
	}
	if m.section.selector.loading {
		b.WriteString(statusMutedStyle.Render("Cargando…"))
		return b.String()
	}
	if len(m.section.selector.visible()) == 0 {
		b.WriteString("No hay participantes disponibles.")
		return b.String()
	}
	for i, item := range m.section.selector.visible() {
		selected := m.pokerVotes[item.ID] == "selected"
		marker := "[ ] "
		if selected {
			marker = "[x] "
		}
		b.WriteString(m.row(i == m.section.selector.cursor, marker+item.Label) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
func (m Model) updatePlanningPoker(key tea.KeyMsg) (Model, tea.Cmd) {
	if m.pokerPhase == "vote" && len(m.pokerParticipants) == 0 {
		if key.Type == tea.KeyEsc {
			m.Screen = "section"
			m.section.step = sectionStepMembers
		}
		return m, nil
	}
	switch key.Type {
	case tea.KeyEsc:
		m.Screen = "section"
		m.section.step = sectionStepMembers
		return m, nil
	case tea.KeyUp:
		if m.pokerPhase == "estimate" {
			if m.pokerCursor > 0 {
				m.pokerCursor--
			}
		} else if m.pokerCursor > 0 {
			m.pokerCursor--
		}
	case tea.KeyDown:
		if m.pokerPhase == "estimate" {
			if m.pokerCursor < len(pokerEstimates)-1 {
				m.pokerCursor++
			}
		} else if m.pokerCursor < len(m.pokerParticipants)-1 {
			m.pokerCursor++
		}
	case tea.KeyEnter:
		switch m.pokerPhase {
		case "vote":
			if m.pokerCursor < len(m.pokerParticipants) && m.pokerVotes[m.pokerParticipants[m.pokerCursor].ID] != "" {
				if m.pokerCursor == len(m.pokerParticipants)-1 {
					m.pokerPhase = "reveal"
				} else {
					m.pokerCursor++
				}
			}
		case "reveal":
			m.pokerPhase = "revealed"
		case "revealed":
			m.pokerPhase = "estimate"
			m.pokerCursor = 0
		case "estimate":
			m.pokerEstimate = pokerEstimates[m.pokerCursor]
			m.pokerPhase = "confirm"
		case "confirm":
			return m, m.setStoryPointsCmd()
		}
	case tea.KeyRunes:
		switch string(key.Runes) {
		case "n":
			m.pokerVotes = map[string]string{}
			m.pokerCursor = 0
			m.pokerPhase = "vote"
		case "r":
			if m.pokerPhase == "reveal" {
				m.pokerPhase = "revealed"
			}
		case "y":
			if m.pokerPhase == "confirm" {
				return m, m.setStoryPointsCmd()
			}
		default:
			if m.pokerPhase == "vote" && len(key.Runes) == 1 {
				v := string(key.Runes)
				for _, c := range pokerCards {
					if c == v && m.pokerCursor < len(m.pokerParticipants) {
						m.pokerVotes[m.pokerParticipants[m.pokerCursor].ID] = v
					}
				}
			}
		}
	}
	return m, nil
}
func (m Model) viewPlanningPoker() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Planning Poker · "+m.section.selected.Label) + "\n\n")
	switch m.pokerPhase {
	case "vote":
		b.WriteString("Participantes: ")
		for i, p := range m.pokerParticipants {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(p.Label)
		}
		if len(m.pokerParticipants) == 0 {
			b.WriteString("\nNo hay participantes seleccionados.")
			break
		}
		b.WriteString("\nVoto privado de " + m.pokerParticipants[m.pokerCursor].Label)
		if m.pokerVotes[m.pokerParticipants[m.pokerCursor].ID] != "" {
			b.WriteString(" · voto cargado")
		}
		b.WriteString("\nElegí una carta: 0 1 2 3 5 8 13 20 40 100 ?")
	case "reveal":
		b.WriteString("Votos cargados. Presioná Enter o r para revelar la ronda.")
	case "revealed":
		different := false
		first := ""
		for _, p := range m.pokerParticipants {
			v := m.pokerVotes[p.ID]
			b.WriteString(p.Label + ": " + v + "\n")
			if first == "" {
				first = v
			} else if first != v {
				different = true
			}
		}
		if different {
			b.WriteString("Los votos difieren.")
		} else {
			b.WriteString("Los votos coinciden.")
		}
	case "estimate":
		b.WriteString(fmt.Sprintf("Elegí Story Points acordado: %d", pokerEstimates[m.pokerCursor]))
	case "confirm":
		b.WriteString(fmt.Sprintf("Confirmar Story Points %d? (y/Enter)", m.pokerEstimate))
	case "result":
		b.WriteString(m.pokerMessage)
	}
	return b.String()
}
func (m Model) planningPokerFooter() string {
	switch m.pokerPhase {
	case "vote":
		return m.keyHints("0–100/?", "votar", "↑/↓", "cambiar participante", "Enter", "siguiente", "Esc", "volver", "q", "salir")
	case "reveal":
		return m.keyHints("Enter/r", "revelar ronda", "Esc", "volver", "q", "salir")
	case "revealed":
		return m.keyHints("Enter", "elegir estimación", "n", "nueva ronda", "Esc", "volver", "q", "salir")
	case "estimate":
		return m.keyHints("↑/↓", "cambiar estimación", "Enter", "continuar", "Esc", "volver", "q", "salir")
	case "confirm":
		return m.keyHints("y/Enter", "confirmar envío", "Esc", "volver", "q", "salir")
	case "result":
		return m.keyHints("Esc", "volver", "q", "salir")
	default:
		return m.keyHints("Esc", "volver", "q", "salir")
	}
}
func (m Model) setStoryPointsCmd() tea.Cmd {
	client := m.storyPointsClient
	p, s, v := m.activeProjectID, m.section.selected.ID, m.pokerEstimate
	return func() tea.Msg { r, e := client.Set(context.Background(), p, s, v); return pokerSaveResult{r, e} }
}

type pokerSaveResult struct {
	response APIResponse
	err      error
}

func (m Model) applyPokerSave(result pokerSaveResult) (Model, tea.Cmd) {
	if result.err != nil {
		m.pokerPhase = "result"
		m.pokerMessage = "Error de conexión: " + result.err.Error()
		return m, nil
	}
	if result.response.StatusCode < 200 || result.response.StatusCode >= 300 {
		m.pokerPhase = "result"
		m.pokerMessage = describeBacklogFailure(result.response)
		return m, nil
	}
	if json.Unmarshal(result.response.Body, &m.pokerStory) != nil {
		m.pokerPhase = "result"
		m.pokerMessage = "No se pudo interpretar la historia guardada."
		return m, nil
	}
	m.backlogStories = nil
	m.Screen = "backlog"
	m.backlogLoading = true
	return m, m.listStoriesCmd(m.activeProjectID)
}
