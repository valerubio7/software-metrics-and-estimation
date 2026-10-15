package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateClosureHistoryMenu(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyUp:
		if m.sprintCursor > 0 {
			m.sprintCursor--
		}
	case tea.KeyDown:
		if m.sprintCursor < 6 {
			m.sprintCursor++
		}
	case tea.KeyEsc:
		m.Screen, m.Cursor, m.sprintCursor = "menu", 0, 0
	case tea.KeyEnter:
		switch m.sprintCursor {
		case 4:
			m.Screen, m.sprintProjectID, m.sprintAction, m.sprintMessage = "sprint-close-project", "", "close", ""
		case 5:
			m.Screen, m.sprintProjectID, m.sprintAction, m.sprintMessage = "sprint-history-project", "", "history", ""
		case 6:
			m.Screen, m.Cursor, m.sprintCursor = "menu", 0, 0
		default:
			return m.updateSprints(key)
		}
	}
	return m, nil
}

func (m Model) updateClosureHistoryProject(key tea.KeyMsg) (Model, tea.Cmd) {
	if key.Type == tea.KeyEsc {
		m.Screen, m.sprintCursor = "sprints", 4
		return m, nil
	}
	if value, ok := editText(m.sprintProjectID, key); ok {
		m.sprintProjectID = value
		return m, nil
	}
	if key.Type == tea.KeyEnter && strings.TrimSpace(m.sprintProjectID) != "" {
		m.sprintProjectID = strings.TrimSpace(m.sprintProjectID)
		if m.sprintAction == "close" {
			return m, m.listOpenSprintsCmd()
		}
		return m, m.historySprintsCmd()
	}
	return m, nil
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
	client, projectID := m.sprintClient, m.sprintProjectID
	return func() tea.Msg {
		response, err := client.History(context.Background(), projectID)
		return SprintResultMsg{Op: "history", Response: response, Err: err}
	}
}

func (m Model) closeSprintCmd() tea.Cmd {
	client, projectID, sprintID := m.sprintClient, m.sprintProjectID, m.selectedSprintID
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
	m.sprintMessage = fmt.Sprintf("Sprint %s closed. Planned and completed stories remain available in the history view.", m.selectedSprintID)
	return m, nil
}

func (m Model) viewSprintHistory() string {
	var b strings.Builder
	b.WriteString("Closed Sprint history\n\n")
	if len(m.sprintHistory.Sprints) == 0 {
		b.WriteString("No closed Sprints in this project.\n")
	}
	for _, sprint := range m.sprintHistory.Sprints {
		b.WriteString(fmt.Sprintf("%s · Goal: %s\n", sprint.ID, sprint.SprintGoal))
		for _, story := range sprint.Stories {
			status := "Not completed"
			if story.Completed {
				status = "Completed"
			}
			b.WriteString(fmt.Sprintf("  %s · %s [%s]", story.ID, story.Title, status))
			if story.Priority != "" {
				b.WriteString(" · Priority: " + story.Priority)
			}
			if story.StoryPoints != nil {
				b.WriteString(fmt.Sprintf(" · Points: %d", *story.StoryPoints))
			}
			b.WriteByte('\n')
		}
	}
	b.WriteString("\nEnter/Esc to return.\n")
	return b.String()
}
