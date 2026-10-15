package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

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

// SprintHistory is the response from GET /projects/{project_id}/sprints.
type SprintHistory struct {
	ProjectID string         `json:"project_id"`
	Sprints   []ClosedSprint `json:"sprints"`
}

// ClosedSprint is a closed Sprint and its planned stories.
type ClosedSprint struct {
	ID         string        `json:"id"`
	ProjectID  string        `json:"project_id"`
	SprintGoal string        `json:"sprint_goal"`
	IsClosed   bool          `json:"is_closed"`
	Stories    []SprintStory `json:"stories"`
}

// SprintStory retains completion and planning details from Sprint history.
type SprintStory struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Priority    string     `json:"priority"`
	StoryPoints *int       `json:"story_points"`
	CompletedAt *time.Time `json:"completed_at"`
	Completed   bool       `json:"completed"`
}

// SprintClosure is the success response from the Sprint closure endpoint.
type SprintClosure struct {
	ProjectID string `json:"project_id"`
	SprintID  string `json:"sprint_id"`
	IsClosed  bool   `json:"is_closed"`
}

// History fetches closed Sprint history without issuing a mutation.
func (c *SprintClient) History(ctx context.Context, projectID string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodGet, sprintsPath(projectID), nil)
}

// Close closes a Sprint using the existing closure endpoint.
func (c *SprintClient) Close(ctx context.Context, projectID, sprintID string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodPost, sprintsPath(projectID)+"/"+url.PathEscape(sprintID)+"/closure", nil)
}

func sprintsPath(projectID string) string {
	return "/projects/" + url.PathEscape(projectID) + "/sprints"
}

func (c *SprintClient) Create(ctx context.Context, projectID, goal string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodPost, "/projects/"+url.PathEscape(projectID)+"/sprints", struct {
		SprintGoal string `json:"sprint_goal"`
	}{goal})
}

func (c *SprintClient) ListOpen(ctx context.Context, projectID string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodGet, "/projects/"+url.PathEscape(projectID)+"/sprints/open", nil)
}

func (c *SprintClient) ListAssigned(ctx context.Context, projectID, sprintID string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodGet, "/projects/"+url.PathEscape(projectID)+"/sprints/"+url.PathEscape(sprintID)+"/stories", nil)
}

func (c *SprintClient) Assign(ctx context.Context, projectID, sprintID string, storyIDs []string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodPost, "/projects/"+url.PathEscape(projectID)+"/sprints/"+url.PathEscape(sprintID)+"/stories", struct {
		StoryIDs []string `json:"story_ids"`
	}{storyIDs})
}

func (c *SprintClient) Complete(ctx context.Context, projectID, sprintID, storyID string) (APIResponse, error) {
	path := "/projects/" + url.PathEscape(projectID) + "/sprints/" + url.PathEscape(sprintID) + "/stories/" + url.PathEscape(storyID) + "/completion"
	return c.api.do(ctx, http.MethodPost, path, nil)
}

type TaskInput struct {
	Title          string   `json:"title"`
	EstimatedHours *float64 `json:"estimated_hours,omitempty"`
}

type TaskClient struct{ api apiClient }

func NewTaskClient(baseURL string) *TaskClient { return &TaskClient{api: newAPIClient(baseURL)} }

func (c *TaskClient) Create(ctx context.Context, projectID, sprintID, storyID string, tasks []TaskInput) (APIResponse, error) {
	path := "/projects/" + url.PathEscape(projectID) + "/sprints/" + url.PathEscape(sprintID) + "/stories/" + url.PathEscape(storyID) + "/tasks"
	return c.api.do(ctx, http.MethodPost, path, struct {
		Tasks []TaskInput `json:"tasks"`
	}{tasks})
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
		if m.sprintCursor < 4 {
			m.sprintCursor++
		}
	case tea.KeyEsc:
		m.Screen, m.Cursor, m.sprintCursor = "menu", 0, 0
	case tea.KeyEnter:
		if m.sprintCursor == 0 {
			m.Screen, m.sprintProjectID = "sprint-project", ""
			m.sprintAction = "create"
		} else if m.sprintCursor >= 1 && m.sprintCursor <= 3 {
			m.Screen, m.sprintProjectID = "sprint-project", ""
			m.sprintAction = "assign"
			if m.sprintCursor == 2 {
				m.sprintAction = "tasks"
			}
			if m.sprintCursor == 3 {
				m.sprintAction = "complete"
			}
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
			if m.sprintAction == "assign" || m.sprintAction == "tasks" || m.sprintAction == "complete" {
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

func (m Model) listAssignedSprintStoriesCmd(sprintID string) tea.Cmd {
	client, projectID := m.sprintClient, m.sprintProjectID
	return func() tea.Msg {
		response, err := client.ListAssigned(context.Background(), projectID, sprintID)
		return SprintResultMsg{Op: "assigned", Response: response, Err: err}
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
		if m.sprintAction == "tasks" || m.sprintAction == "complete" {
			return m, m.listAssignedSprintStoriesCmd(m.selectedSprintID)
		}
		return m, m.listAssignedBacklogCmd(m.selectedSprintID)
	}
	return m, nil
}

func (m Model) updateAssignedStories(key tea.KeyMsg) (Model, tea.Cmd) {
	if key.Type == tea.KeyUp && m.sprintCursor > 0 {
		m.sprintCursor--
	}
	if key.Type == tea.KeyDown && m.sprintCursor < len(m.sprintStories)-1 {
		m.sprintCursor++
	}
	if key.Type == tea.KeyEsc {
		m.Screen, m.sprintCursor = "sprint-open-list", 0
	}
	if key.Type == tea.KeyEnter && len(m.sprintStories) > 0 {
		m.taskStoryID = m.sprintStories[m.sprintCursor].ID
		if m.sprintAction == "complete" {
			client, projectID, sprintID, storyID := m.sprintClient, m.sprintProjectID, m.selectedSprintID, m.taskStoryID
			return m, func() tea.Msg {
				response, err := client.Complete(context.Background(), projectID, sprintID, storyID)
				return SprintResultMsg{Op: "complete", Response: response, Err: err}
			}
		}
		m.taskTitle, m.taskHours, m.taskField, m.Screen = "", "", 0, "sprint-task-form"
	}
	return m, nil
}

func (m Model) updateSprintTaskForm(key tea.KeyMsg) (Model, tea.Cmd) {
	value := &m.taskTitle
	if m.taskField == 1 {
		value = &m.taskHours
	}
	if key.Type == tea.KeyEsc {
		m.Screen = "sprint-assigned-list"
		return m, nil
	}
	if key.Type == tea.KeyTab || key.Type == tea.KeyDown || key.Type == tea.KeyUp {
		m.taskField = (m.taskField + 1) % 2
		return m, nil
	}
	if text, ok := editText(*value, key); ok {
		*value = text
		m.sprintMessage = ""
		return m, nil
	}
	if key.Type == tea.KeyEnter {
		if m.taskField == 0 {
			m.taskField = 1
			return m, nil
		}
		var hours *float64
		if strings.TrimSpace(m.taskHours) != "" {
			parsed, err := parseEstimatedHours(m.taskHours)
			if err != nil {
				m.sprintMessage = err.Error()
				return m, nil
			}
			hours = parsed
		}
		client, projectID, sprintID, storyID, title := m.taskClient, m.sprintProjectID, m.selectedSprintID, m.taskStoryID, m.taskTitle
		return m, func() tea.Msg {
			response, err := client.Create(context.Background(), projectID, sprintID, storyID, []TaskInput{{Title: title, EstimatedHours: hours}})
			return SprintResultMsg{Op: "tasks", Response: response, Err: err}
		}
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
	if result.Op == "assigned" {
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
		m.sprintStories, m.sprintCursor, m.Screen = listing.Stories, 0, "sprint-assigned-list"
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
	if result.Op == "tasks" {
		m.sprintMessage = fmt.Sprintf("Tasks created (%d)", result.Response.StatusCode)
		return m, nil
	}
	if result.Op == "complete" {
		m.sprintMessage = fmt.Sprintf("Story completion response (%d)", result.Response.StatusCode)
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
