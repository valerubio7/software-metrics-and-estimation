package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// updateClosureHistoryMenu is the Sprint operations menu. Actions that need a
// project open the shared picker; the rest go straight to their screen.
func (m Model) updateClosureHistoryMenu(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyUp:
		if m.sprintCursor > 0 {
			m.sprintCursor--
		}
	case tea.KeyDown:
		if m.sprintCursor < len(sprintMenuOptions())-1 {
			m.sprintCursor++
		}
	case tea.KeyEsc:
		m.Screen, m.Cursor, m.sprintCursor = "menu", 0, 0
	case tea.KeyEnter:
		switch m.sprintCursor {
		case 4:
			return m.openSprintAction("sprint-close")
		case 5:
			return m.openSprintAction("sprint-history")
		case 6:
			m.Screen, m.Cursor, m.sprintCursor = "menu", 0, 0
		default:
			return m.openSprintAction(sprintPurposeFor(m.sprintCursor))
		}
	}
	return m, nil
}

// sprintPurposeFor maps the first four menu entries to their picker purpose.
func sprintPurposeFor(cursor int) string {
	switch cursor {
	case 0:
		return "sprint-create"
	case 1:
		return "sprint-assign"
	case 2:
		return "sprint-tasks"
	default:
		return "sprint-complete"
	}
}

func (m Model) updateClosureOpenSprints(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyUp:
		if m.sprintCursor > 0 {
			m.sprintCursor--
		}
	case tea.KeyDown:
		if m.sprintCursor < len(m.openSprints)-1 {
			m.sprintCursor++
		}
	case tea.KeyEsc:
		m.Screen, m.sprintCursor = "sprints", 4
	case tea.KeyEnter:
		if len(m.openSprints) == 0 {
			return m, nil
		}
		m.selectedSprintID = m.openSprints[m.sprintCursor].ID
		return m, m.closeSprintCmd()
	}
	return m, nil
}

func (m Model) historySprintsCmd() tea.Cmd {
	client, projectID := m.sprintClient, m.activeProjectID
	return func() tea.Msg {
		response, err := client.History(context.Background(), projectID)
		return SprintResultMsg{Op: "history", Response: response, Err: err}
	}
}

func (m Model) closeSprintCmd() tea.Cmd {
	client, projectID, sprintID := m.sprintClient, m.activeProjectID, m.selectedSprintID
	return func() tea.Msg {
		response, err := client.Close(context.Background(), projectID, sprintID)
		return SprintResultMsg{Op: "close", Response: response, Err: err}
	}
}

func (m Model) applyClosureHistoryResult(result SprintResultMsg) (Model, tea.Cmd) {
	if result.Op == "open" && m.sprintAction == "close" {
		if message, failed := sprintFailure(result); failed {
			m.Screen, m.sprintMessage = "sprint-result", message
			return m, nil
		}
		var listing struct {
			Sprints []Sprint `json:"sprints"`
		}
		if err := json.Unmarshal(result.Response.Body, &listing); err != nil {
			m.Screen, m.sprintMessage = "sprint-result", err.Error()
			return m, nil
		}
		m.openSprints, m.sprintCursor, m.Screen = listing.Sprints, 0, "sprint-open-list"
		return m, nil
	}
	if result.Op == "history" {
		if message, failed := sprintFailure(result); failed {
			m.Screen, m.sprintMessage = "sprint-result", message
			return m, nil
		}
		var history SprintHistory
		if err := json.Unmarshal(result.Response.Body, &history); err != nil {
			m.Screen, m.sprintMessage = "sprint-result", err.Error()
			return m, nil
		}
		m.sprintHistory, m.Screen = history, "sprint-history"
		return m, nil
	}
	m.Screen = "sprint-result"
	if message, failed := sprintFailure(result); failed {
		m.sprintMessage = message
		return m, nil
	}
	m.sprintMessage = fmt.Sprintf("Sprint %s cerrado. Las historias planificadas y completadas siguen disponibles en el historial.", shortID(m.selectedSprintID))
	return m, nil
}

func (m Model) viewSprintHistory() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Historial de Sprints cerrados"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Proyecto: " + m.projectLabel()))
	b.WriteString("\n\n")
	if len(m.sprintHistory.Sprints) == 0 {
		b.WriteString(emptyState("◇", "No hay Sprints cerrados en este proyecto.", "Cerrá un Sprint para verlo acá."))
		return b.String()
	}
	for _, sprint := range m.sprintHistory.Sprints {
		completed, total := 0, len(sprint.Stories)
		for _, story := range sprint.Stories {
			if story.Completed {
				completed++
			}
		}
		goal := sprint.SprintGoal
		if goal == "" {
			goal = "(sin objetivo)"
		}
		b.WriteString(cursorStyle.Render("▸") + " " + sprintHistoryTitle(goal, sprint.ID) + "\n")
		b.WriteString("  " + hintStyle.Render(fmt.Sprintf("Progreso: %d/%d historias completadas", completed, total)) + "\n")
		for _, story := range sprint.Stories {
			b.WriteString("  " + sprintHistoryStoryRow(story) + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func sprintHistoryTitle(goal, id string) string {
	return titleStyle.Render(goal) + hintStyle.Render("   "+shortID(id))
}

func sprintHistoryStoryRow(story SprintStory) string {
	status := storyStatusBadge("pendiente")
	if story.Completed {
		status = storyStatusBadge("completada")
	}
	row := story.Title + "  " + status
	if story.Priority != "" {
		row += "  " + priorityBadge(story.Priority)
	}
	if story.StoryPoints != nil {
		row += hintStyle.Render(fmt.Sprintf("   %d pts", *story.StoryPoints))
	}
	return row
}
