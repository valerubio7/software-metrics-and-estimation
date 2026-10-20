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

// openSprints enters the Sprints browser: the searchable list of the open Sprints
// of the session project. Without a project it first asks for one.
func (m Model) enterSprints() (Model, tea.Cmd) {
	if m.activeProjectID == "" {
		return m.openProjectPicker("sprints")
	}
	m.Screen = "sprints"
	m.sprintFilter, m.sprintFiltering, m.sprintCursor = "", false, 0
	m.sprintMessage, m.sprintReturn = "", ""
	m.sprintStoriesLoading, m.sprintHistoryLoading = false, false
	m.sprintLoading = true
	return m, m.openSprintsCmd()
}

// openAssignedStories shows the stories assigned to a Sprint, where Enter creates
// tasks and c completes them.
func (m Model) openAssignedStories(sprintID string) (Model, tea.Cmd) {
	m.Screen, m.sprintStoriesLoading, m.sprintStories = "sprint-assigned-list", true, nil
	m.sprintCursor, m.sprintFilter, m.sprintFiltering = 0, "", false
	return m, m.listAssignedSprintStoriesCmd(sprintID)
}

// openSprintHistory shows the closed Sprints of the session project, read-only.
func (m Model) openSprintHistory() (Model, tea.Cmd) {
	m.Screen, m.sprintHistoryLoading = "sprint-history", true
	m.historyCursor, m.historyExpanded = 0, ""
	m.sprintFilter, m.sprintFiltering = "", false
	return m, m.historySprintsCmd()
}

// updateSprints drives the browser of open Sprints with the shared navigator keys.
func (m Model) updateSprints(key tea.KeyMsg) (Model, tea.Cmd) {
	sprints := m.visibleOpenSprints()

	if m.sprintFiltering {
		switch key.Type {
		case tea.KeyEsc:
			m.sprintFiltering, m.sprintFilter, m.sprintCursor = false, "", 0
			return m, nil
		case tea.KeyEnter:
			// Stop searching but keep the filter, so the rows can then be acted on.
			m.sprintFiltering = false
			return m, nil
		case tea.KeyUp:
			if m.sprintCursor > 0 {
				m.sprintCursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.sprintCursor < len(sprints)-1 {
				m.sprintCursor++
			}
			return m, nil
		}
		if value, ok := editText(m.sprintFilter, key); ok {
			m.sprintFilter, m.sprintCursor = value, 0
		}
		return m, nil
	}

	switch key.Type {
	case tea.KeyEsc:
		if m.sprintFilter != "" {
			m.sprintFilter, m.sprintCursor = "", 0
			return m, nil
		}
		m.Screen, m.Cursor = "menu", 0
	case tea.KeyUp:
		if m.sprintCursor > 0 {
			m.sprintCursor--
		}
	case tea.KeyDown:
		if m.sprintCursor < len(sprints)-1 {
			m.sprintCursor++
		}
	case tea.KeyEnter:
		sprint, ok := sprintAt(sprints, m.sprintCursor)
		if !ok {
			return m, nil
		}
		m.selectedSprintID = sprint.ID
		return m.openAssignedStories(sprint.ID)
	case tea.KeyRunes:
		switch string(key.Runes) {
		case "/":
			m.sprintFiltering = true
		case "n":
			m.Screen, m.sprintGoal = "sprint-goal", ""
		case "a":
			sprint, ok := sprintAt(sprints, m.sprintCursor)
			if !ok {
				return m, nil
			}
			m.selectedSprintID = sprint.ID
			m.Screen, m.sprintStoriesLoading = "sprint-story-select", true
			m.sprintStories, m.selectedStories = nil, map[string]bool{}
			m.sprintCursor, m.sprintFilter, m.sprintFiltering = 0, "", false
			return m, m.listAssignedBacklogCmd()
		case "x":
			sprint, ok := sprintAt(sprints, m.sprintCursor)
			if !ok {
				return m, nil
			}
			m.selectedSprintID = sprint.ID
			m.Screen = "sprint-confirm"
		case "h":
			return m.openSprintHistory()
		case "p":
			// p is the global «change project» shortcut; keep it inert here so it
			// does not seed a search.
		default:
			m.sprintFiltering = true
			if value, ok := editText(m.sprintFilter, key); ok {
				m.sprintFilter, m.sprintCursor = value, 0
			}
		}
	}
	return m, nil
}

// updateSprintConfirm asks before the destructive close: s confirms, n or Esc cancel.
func (m Model) updateSprintConfirm(key tea.KeyMsg) (Model, tea.Cmd) {
	if key.Type == tea.KeyEsc {
		m.Screen = "sprints"
		return m, nil
	}
	if key.Type == tea.KeyRunes {
		switch strings.ToLower(string(key.Runes)) {
		case "s":
			return m, m.closeSprintCmd()
		case "n":
			m.Screen = "sprints"
		}
	}
	return m, nil
}

// updateSprintHistory drives the closed Sprints list: Enter expands a Sprint to
// show its stories, which are already loaded with the history response.
func (m Model) updateSprintHistory(key tea.KeyMsg) (Model, tea.Cmd) {
	sprints := m.visibleHistorySprints()

	if m.sprintFiltering {
		switch key.Type {
		case tea.KeyEsc:
			m.sprintFiltering, m.sprintFilter, m.historyCursor = false, "", 0
			return m, nil
		case tea.KeyEnter:
			m.sprintFiltering = false
			return m, nil
		case tea.KeyUp:
			if m.historyCursor > 0 {
				m.historyCursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.historyCursor < len(sprints)-1 {
				m.historyCursor++
			}
			return m, nil
		}
		if value, ok := editText(m.sprintFilter, key); ok {
			m.sprintFilter, m.historyCursor = value, 0
		}
		return m, nil
	}

	switch key.Type {
	case tea.KeyEsc:
		if m.sprintFilter != "" {
			m.sprintFilter, m.historyCursor = "", 0
			return m, nil
		}
		m.Screen = "sprints"
	case tea.KeyUp:
		if m.historyCursor > 0 {
			m.historyCursor--
		}
	case tea.KeyDown:
		if m.historyCursor < len(sprints)-1 {
			m.historyCursor++
		}
	case tea.KeyEnter:
		if m.historyCursor < 0 || m.historyCursor >= len(sprints) {
			return m, nil
		}
		if id := sprints[m.historyCursor].ID; m.historyExpanded == id {
			m.historyExpanded = ""
		} else {
			m.historyExpanded = id
		}
	case tea.KeyRunes:
		switch string(key.Runes) {
		case "/":
			m.sprintFiltering = true
		case "p":
			// p is the global shortcut; keep it inert here.
		default:
			m.sprintFiltering = true
			if value, ok := editText(m.sprintFilter, key); ok {
				m.sprintFilter, m.historyCursor = value, 0
			}
		}
	}
	return m, nil
}

func (m Model) updateSprintForm(key tea.KeyMsg) (Model, tea.Cmd) {
	if key.Type == tea.KeyEsc {
		m.Screen = "sprints"
		return m, nil
	}
	if value, ok := editText(m.sprintGoal, key); ok {
		m.sprintGoal = value
		return m, nil
	}
	if key.Type == tea.KeyEnter {
		return m, m.createSprintCmd()
	}
	return m, nil
}

// updateAssignedStories drives the stories of a Sprint: Enter creates tasks for
// the highlighted story and c completes it.
func (m Model) updateAssignedStories(key tea.KeyMsg) (Model, tea.Cmd) {
	stories := m.visibleSprintStories()

	if m.sprintFiltering {
		switch key.Type {
		case tea.KeyEsc:
			m.sprintFiltering, m.sprintFilter, m.sprintCursor = false, "", 0
			return m, nil
		case tea.KeyEnter:
			m.sprintFiltering = false
			return m, nil
		case tea.KeyUp:
			if m.sprintCursor > 0 {
				m.sprintCursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.sprintCursor < len(stories)-1 {
				m.sprintCursor++
			}
			return m, nil
		}
		if value, ok := editText(m.sprintFilter, key); ok {
			m.sprintFilter, m.sprintCursor = value, 0
		}
		return m, nil
	}

	switch key.Type {
	case tea.KeyEsc:
		if m.sprintFilter != "" {
			m.sprintFilter, m.sprintCursor = "", 0
			return m, nil
		}
		m.Screen = "sprints"
	case tea.KeyUp:
		if m.sprintCursor > 0 {
			m.sprintCursor--
		}
	case tea.KeyDown:
		if m.sprintCursor < len(stories)-1 {
			m.sprintCursor++
		}
	case tea.KeyEnter:
		return m.openTaskForm(stories)
	case tea.KeyRunes:
		switch string(key.Runes) {
		case "/":
			m.sprintFiltering = true
		case "c":
			return m.completeSelectedStory(stories)
		case "p":
			// p is the global shortcut; keep it inert here.
		default:
			m.sprintFiltering = true
			if value, ok := editText(m.sprintFilter, key); ok {
				m.sprintFilter, m.sprintCursor = value, 0
			}
		}
	}
	return m, nil
}

// openTaskForm opens the task form for the highlighted story, or does nothing on
// an empty list.
func (m Model) openTaskForm(stories []Story) (Model, tea.Cmd) {
	if m.sprintCursor < 0 || m.sprintCursor >= len(stories) {
		return m, nil
	}
	m.taskStoryID = stories[m.sprintCursor].ID
	m.taskTitle, m.taskHours, m.taskField, m.Screen = "", "", 0, "sprint-task-form"
	return m, nil
}

// completeSelectedStory sends the completion of the highlighted story; the API
// owns the business rules for the closure state.
func (m Model) completeSelectedStory(stories []Story) (Model, tea.Cmd) {
	if m.sprintCursor < 0 || m.sprintCursor >= len(stories) {
		return m, nil
	}
	storyID := stories[m.sprintCursor].ID
	client, projectID, sprintID := m.sprintClient, m.activeProjectID, m.selectedSprintID
	return m, func() tea.Msg {
		response, err := client.Complete(context.Background(), projectID, sprintID, storyID)
		return SprintResultMsg{Op: "complete", Response: response, Err: err}
	}
}

// updateSprintStorySelection drives the multiselect that assigns backlog stories
// to a Sprint: Space marks, Enter confirms.
func (m Model) updateSprintStorySelection(key tea.KeyMsg) (Model, tea.Cmd) {
	stories := m.visibleSprintStories()

	if m.sprintFiltering {
		switch key.Type {
		case tea.KeyEsc:
			m.sprintFiltering, m.sprintFilter, m.sprintCursor = false, "", 0
			return m, nil
		case tea.KeyEnter:
			m.sprintFiltering = false
			return m, nil
		case tea.KeyUp:
			if m.sprintCursor > 0 {
				m.sprintCursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.sprintCursor < len(stories)-1 {
				m.sprintCursor++
			}
			return m, nil
		}
		if value, ok := editText(m.sprintFilter, key); ok {
			m.sprintFilter, m.sprintCursor = value, 0
		}
		return m, nil
	}

	switch key.Type {
	case tea.KeyEsc:
		if m.sprintFilter != "" {
			m.sprintFilter, m.sprintCursor = "", 0
			return m, nil
		}
		m.Screen = "sprints"
	case tea.KeyUp:
		if m.sprintCursor > 0 {
			m.sprintCursor--
		}
	case tea.KeyDown:
		if m.sprintCursor < len(stories)-1 {
			m.sprintCursor++
		}
	case tea.KeySpace:
		if m.sprintCursor >= 0 && m.sprintCursor < len(stories) {
			id := stories[m.sprintCursor].ID
			m.selectedStories[id] = !m.selectedStories[id]
		}
	case tea.KeyEnter:
		if len(stories) == 0 {
			return m, nil
		}
		return m, m.assignSprintStoriesCmd(m.selectedSprintID)
	case tea.KeyRunes:
		switch string(key.Runes) {
		case "/":
			m.sprintFiltering = true
		case "p":
			// p is the global shortcut; keep it inert here.
		default:
			m.sprintFiltering = true
			if value, ok := editText(m.sprintFilter, key); ok {
				m.sprintFilter, m.sprintCursor = value, 0
			}
		}
	}
	return m, nil
}

func (m Model) openSprintsCmd() tea.Cmd {
	client, projectID := m.sprintClient, m.activeProjectID
	return func() tea.Msg {
		response, err := client.ListOpen(context.Background(), projectID)
		return SprintResultMsg{Op: "open", Response: response, Err: err}
	}
}

func (m Model) createSprintCmd() tea.Cmd {
	client, projectID, goal := m.sprintClient, m.activeProjectID, m.sprintGoal
	return func() tea.Msg {
		response, err := client.Create(context.Background(), projectID, goal)
		return SprintResultMsg{Op: "create", Response: response, Err: err}
	}
}

func (m Model) listAssignedBacklogCmd() tea.Cmd {
	client, projectID := m.backlogClient, m.activeProjectID
	return func() tea.Msg {
		response, err := client.List(context.Background(), projectID)
		return SprintResultMsg{Op: "backlog", Response: response, Err: err}
	}
}

func (m Model) listAssignedSprintStoriesCmd(sprintID string) tea.Cmd {
	client, projectID := m.sprintClient, m.activeProjectID
	return func() tea.Msg {
		response, err := client.ListAssigned(context.Background(), projectID, sprintID)
		return SprintResultMsg{Op: "assigned", Response: response, Err: err}
	}
}

func (m Model) assignSprintStoriesCmd(sprintID string) tea.Cmd {
	client, projectID := m.sprintClient, m.activeProjectID
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

// applySprintResult routes each Sprint response to the screen that owns it: the
// list responses fill their collection and mutations land on the result screen.
func (m Model) applySprintResult(result SprintResultMsg) (Model, tea.Cmd) {
	switch result.Op {
	case "open":
		m.sprintLoading = false
		m.Screen = "sprints"
		if message, failed := sprintFailure(result); failed {
			m.Screen, m.sprintMessage, m.sprintOK = "sprint-result", message, false
			return m, nil
		}
		var listing struct {
			Sprints []Sprint `json:"sprints"`
		}
		if err := json.Unmarshal(result.Response.Body, &listing); err != nil {
			m.Screen, m.sprintMessage, m.sprintOK = "sprint-result", unreadableResponse(err), false
			return m, nil
		}
		m.openSprints, m.sprintCursor = listing.Sprints, 0
		return m, nil
	case "history":
		m.sprintHistoryLoading = false
		m.Screen = "sprint-history"
		if message, failed := sprintFailure(result); failed {
			m.Screen, m.sprintMessage, m.sprintOK = "sprint-result", message, false
			return m, nil
		}
		var history SprintHistory
		if err := json.Unmarshal(result.Response.Body, &history); err != nil {
			m.Screen, m.sprintMessage, m.sprintOK = "sprint-result", unreadableResponse(err), false
			return m, nil
		}
		m.sprintHistory, m.historyCursor, m.historyExpanded = history, 0, ""
		return m, nil
	case "backlog":
		m.sprintStoriesLoading = false
		m.Screen = "sprint-story-select"
		if message, failed := sprintFailure(result); failed {
			m.Screen, m.sprintMessage, m.sprintOK = "sprint-result", message, false
			return m, nil
		}
		listed, ok := storyListing(result)
		if !ok {
			m.Screen, m.sprintMessage, m.sprintOK = "sprint-result", "No se pudo interpretar la respuesta de la API.", false
			return m, nil
		}
		m.sprintStories, m.selectedStories, m.sprintCursor = listed, map[string]bool{}, 0
		return m, nil
	case "assigned":
		m.sprintStoriesLoading = false
		m.Screen = "sprint-assigned-list"
		if message, failed := sprintFailure(result); failed {
			m.Screen, m.sprintMessage, m.sprintOK = "sprint-result", message, false
			return m, nil
		}
		listed, ok := storyListing(result)
		if !ok {
			m.Screen, m.sprintMessage, m.sprintOK = "sprint-result", "No se pudo interpretar la respuesta de la API.", false
			return m, nil
		}
		m.sprintStories, m.sprintCursor = listed, 0
		return m, nil
	}

	m.Screen = "sprint-result"
	if message, failed := sprintFailure(result); failed {
		m.sprintMessage, m.sprintOK = message, false
		m.sprintReturn = "sprints"
		if result.Op == "tasks" || result.Op == "complete" {
			m.sprintReturn = "sprint-assigned-list"
		}
		return m, nil
	}
	m.sprintOK = true
	switch result.Op {
	case "close":
		m.sprintReturn = "sprints"
		m.sprintMessage = fmt.Sprintf("Sprint %s cerrado. Las historias planificadas y completadas siguen disponibles en el historial.", shortID(m.selectedSprintID))
	case "assign":
		m.sprintReturn = "sprints"
		m.sprintMessage = "Historias asignadas al Sprint correctamente."
	case "tasks":
		m.sprintReturn = "sprint-assigned-list"
		m.sprintMessage = fmt.Sprintf("Tareas creadas correctamente (%d).", result.Response.StatusCode)
	case "complete":
		m.sprintReturn = "sprint-assigned-list"
		m.sprintMessage = fmt.Sprintf("Historia completada (%d).", result.Response.StatusCode)
	default:
		m.sprintReturn = "sprints"
		var sprint Sprint
		if json.Unmarshal(result.Response.Body, &sprint) == nil && sprint.ID != "" {
			m.sprintMessage = "Sprint creado: " + sprint.ID
		} else {
			m.sprintMessage = strings.TrimSpace(string(result.Response.Body))
		}
	}
	return m, nil
}

// storyListing decodes a {"stories":[...]} response, reporting whether the body
// was readable.
func storyListing(result SprintResultMsg) ([]Story, bool) {
	var listing struct {
		Stories []Story `json:"stories"`
	}
	if err := json.Unmarshal(result.Response.Body, &listing); err != nil {
		return nil, false
	}
	return listing.Stories, true
}

func unreadableResponse(err error) string {
	return "No se pudo interpretar la respuesta de la API: " + err.Error()
}

func sprintFailure(result SprintResultMsg) (string, bool) {
	if result.Err != nil {
		return "Error de conexión: " + result.Err.Error(), true
	}
	if result.Response.StatusCode < 200 || result.Response.StatusCode >= 300 {
		return describeBacklogFailure(result.Response), true
	}
	return "", false
}

// sprintAt returns the Sprint highlighted by cursor, or false when the list is
// empty or the index is out of range.
func sprintAt(sprints []Sprint, cursor int) (Sprint, bool) {
	if cursor < 0 || cursor >= len(sprints) {
		return Sprint{}, false
	}
	return sprints[cursor], true
}

// filterList narrows items by a case-insensitive substring of a chosen field.
func filterList[T any](items []T, filter string, field func(T) string) []T {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return items
	}
	matches := make([]T, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(field(item)), filter) {
			matches = append(matches, item)
		}
	}
	return matches
}

func (m Model) visibleOpenSprints() []Sprint {
	return filterList(m.openSprints, m.sprintFilter, func(s Sprint) string { return s.SprintGoal })
}

func (m Model) visibleHistorySprints() []ClosedSprint {
	return filterList(m.sprintHistory.Sprints, m.sprintFilter, func(s ClosedSprint) string { return s.SprintGoal })
}

func (m Model) visibleSprintStories() []Story {
	return filterList(m.sprintStories, m.sprintFilter, func(s Story) string { return s.Title })
}
