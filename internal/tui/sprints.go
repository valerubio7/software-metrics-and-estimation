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

func (c *SprintClient) ListOpen(ctx context.Context, projectID string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodGet, "/projects/"+url.PathEscape(projectID)+"/sprints/open", nil)
}

func (c *SprintClient) Assign(ctx context.Context, projectID, sprintID string, storyIDs []string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodPost, "/projects/"+url.PathEscape(projectID)+"/sprints/"+url.PathEscape(sprintID)+"/stories", struct {
		StoryIDs []string `json:"story_ids"`
	}{storyIDs})
}

type SprintResultMsg struct {
	Op       string
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
		if m.sprintCursor < 2 {
			m.sprintCursor++
		}
	case tea.KeyEsc:
		m.Screen, m.Cursor, m.sprintCursor = "menu", 0, 0
	case tea.KeyEnter:
		if m.sprintCursor == 0 {
			m.Screen, m.sprintProjectID = "sprint-project", ""
			m.sprintAction = "create"
		} else if m.sprintCursor == 1 {
			m.Screen, m.sprintProjectID = "sprint-project", ""
			m.sprintAction = "assign"
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
			if m.sprintAction == "assign" {
				return m, m.listOpenSprintsCmd()
			}
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
		return SprintResultMsg{Op: "create", Response: response, Err: err}
	}
}

func (m Model) listOpenSprintsCmd() tea.Cmd {
	client, projectID := m.sprintClient, m.sprintProjectID
	return func() tea.Msg {
		response, err := client.ListOpen(context.Background(), projectID)
		return SprintResultMsg{Op: "open", Response: response, Err: err}
	}
}

func (m Model) listAssignedBacklogCmd(sprintID string) tea.Cmd {
	client, projectID := m.backlogClient, m.sprintProjectID
	return func() tea.Msg {
		response, err := client.List(context.Background(), projectID)
		return SprintResultMsg{Op: "backlog", Response: response, Err: err}
	}
}

func (m Model) assignSprintStoriesCmd(sprintID string) tea.Cmd {
	client, projectID := m.sprintClient, m.sprintProjectID
	ids := make([]string, 0, len(m.selectedStories))
	for _, story := range m.sprintStories {
		if m.selectedStories[story.ID] {
			ids = append(ids, story.ID)
		}
	}
	return func() tea.Msg {
		response, err := client.Assign(context.Background(), projectID, sprintID, ids)
		return SprintResultMsg{Op: "assign", Response: response, Err: err}
	}
}

func (m Model) updateOpenSprints(key tea.KeyMsg) (Model, tea.Cmd) {
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
		m.Screen, m.sprintCursor = "sprints", 1
	case tea.KeyEnter:
		if len(m.openSprints) == 0 {
			return m, nil
		}
		m.selectedSprintID = m.openSprints[m.sprintCursor].ID
		return m, m.listAssignedBacklogCmd(m.selectedSprintID)
	}
	return m, nil
}

func (m Model) updateSprintStorySelection(key tea.KeyMsg) (Model, tea.Cmd) {
	if key.Type == tea.KeyUp && m.sprintCursor > 0 {
		m.sprintCursor--
	}
	if key.Type == tea.KeyDown && m.sprintCursor < len(m.sprintStories)-1 {
		m.sprintCursor++
	}
	if key.Type == tea.KeySpace && len(m.sprintStories) > 0 {
		id := m.sprintStories[m.sprintCursor].ID
		m.selectedStories[id] = !m.selectedStories[id]
	}
	if key.Type == tea.KeyEsc {
		m.Screen, m.sprintCursor = "sprint-open-list", 0
	}
	if key.Type == tea.KeyEnter {
		return m, m.assignSprintStoriesCmd(m.selectedSprintID)
	}
	return m, nil
}

func (m Model) applySprintResult(result SprintResultMsg) (Model, tea.Cmd) {
	if result.Op == "open" {
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
	if result.Op == "backlog" {
		if message, failed := sprintFailure(result); failed {
			m.Screen, m.sprintMessage = "sprint-result", message
			return m, nil
		}
		var listing struct {
			Stories []Story `json:"stories"`
		}
		if err := json.Unmarshal(result.Response.Body, &listing); err != nil {
			m.Screen, m.sprintMessage = "sprint-result", err.Error()
			return m, nil
		}
		m.sprintStories, m.selectedStories, m.sprintCursor, m.Screen = listing.Stories, map[string]bool{}, 0, "sprint-story-select"
		return m, nil
	}
	m.Screen = "sprint-result"
	if message, failed := sprintFailure(result); failed {
		m.sprintMessage = message
		return m, nil
	}
	if result.Op == "assign" {
		m.sprintMessage = "Stories assigned to Sprint"
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

func sprintFailure(result SprintResultMsg) (string, bool) {
	if result.Err != nil {
		return "Connection error: " + result.Err.Error(), true
	}
	if result.Response.StatusCode < 200 || result.Response.StatusCode >= 300 {
		return describeBacklogFailure(result.Response), true
	}
	return "", false
}
