package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Sprint struct {
	ID         string `json:"id"`
	ProjectID  string `json:"project_id"`
	SprintGoal string `json:"sprint_goal"`
	IsClosed   bool   `json:"is_closed"`
}

type SprintClient struct{ api apiClient }

func NewSprintClient(baseURL string) *SprintClient { return &SprintClient{api: newAPIClient(baseURL)} }

func (c *SprintClient) Create(ctx context.Context, projectID, goal string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodPost, "/projects/"+url.PathEscape(projectID)+"/sprints", struct {
		SprintGoal string `json:"sprint_goal"`
	}{goal})
}

type SprintResultMsg struct {
	Response APIResponse
	Err      error
}

func (m Model) updateSprints(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyUp:
		if m.sprintCursor > 0 {
			m.sprintCursor--
		}
	case tea.KeyDown:
		if m.sprintCursor < 1 {
			m.sprintCursor++
		}
	case tea.KeyEsc:
		m.Screen, m.Cursor, m.sprintCursor = "menu", 0, 0
	case tea.KeyEnter:
		if m.sprintCursor == 0 {
			m.Screen, m.sprintProjectID = "sprint-project", ""
		} else {
			m.Screen, m.Cursor = "menu", 0
		}
	}
	return m, nil
}

func (m Model) updateSprintForm(key tea.KeyMsg) (Model, tea.Cmd) {
	if key.Type == tea.KeyEsc {
		if m.Screen == "sprint-goal" {
			m.Screen = "sprint-project"
		} else {
			m.Screen = "sprints"
		}
		return m, nil
	}
	if value, ok := editText(m.sprintProjectID, key); ok && m.Screen == "sprint-project" {
		m.sprintProjectID = value
		return m, nil
	}
	if value, ok := editText(m.sprintGoal, key); ok && m.Screen == "sprint-goal" {
		m.sprintGoal = value
		return m, nil
	}
	if key.Type == tea.KeyEnter {
		if m.Screen == "sprint-project" && strings.TrimSpace(m.sprintProjectID) != "" {
			m.sprintProjectID = strings.TrimSpace(m.sprintProjectID)
			m.Screen = "sprint-goal"
			return m, nil
		}
		if m.Screen == "sprint-goal" {
			return m, m.createSprintCmd()
		}
	}
	return m, nil
}

func (m Model) createSprintCmd() tea.Cmd {
	client, projectID, goal := m.sprintClient, m.sprintProjectID, m.sprintGoal
	return func() tea.Msg {
		response, err := client.Create(context.Background(), projectID, goal)
		return SprintResultMsg{Response: response, Err: err}
	}
}

func (m Model) applySprintResult(result SprintResultMsg) (Model, tea.Cmd) {
	m.Screen = "sprint-result"
	if result.Err != nil {
		m.sprintMessage = "Connection error: " + result.Err.Error()
		return m, nil
	}
	if result.Response.StatusCode < 200 || result.Response.StatusCode >= 300 {
		m.sprintMessage = describeBacklogFailure(result.Response)
		return m, nil
	}
	var sprint Sprint
	if json.Unmarshal(result.Response.Body, &sprint) == nil && sprint.ID != "" {
		m.sprintMessage = fmt.Sprintf("Sprint %s created", sprint.ID)
	} else {
		m.sprintMessage = strings.TrimSpace(string(result.Response.Body))
	}
	return m, nil
}
