package tui_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

// startSprints opens Sprints from the top-level menu (Down, Down, Enter). With no
// active project this lands on the project picker and returns its load command.
func startSprints(url string) (tui.Model, tea.Cmd) {
	m := tui.NewModel(url)
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeyDown)
	return apply(m, tea.KeyMsg{Type: tea.KeyEnter})
}

// openSprintsWithProject goes menu -> Sprints -> picker -> pid -> Enter and applies
// the open Sprint list.
func openSprintsWithProject(t *testing.T, url, pid string) tui.Model {
	t.Helper()
	m, load := startSprints(url)
	m, cmd := confirmProjectInPicker(t, m, load, pid)
	m = run(t, m, cmd)
	if m.Screen != "sprints" {
		t.Fatalf("Screen = %q after loading, want sprints", m.Screen)
	}
	return m
}

// sprintServer answers GET /projects plus the Sprint discovery endpoints.
func sprintServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request) bool) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/projects" {
			_, _ = w.Write([]byte(projectsBody(t, sampleProjects())))
			return
		}
		if handler(w, r) {
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestSprintBrowserListsOpenSprintsWithNavigatorKeys(t *testing.T) {
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path == "/projects/p1/sprints/open" {
			_, _ = w.Write([]byte(`{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Ship the release","is_closed":false}]}`))
			return true
		}
		return false
	})

	m := openSprintsWithProject(t, server.URL, "p1")
	view := m.View()
	for _, want := range []string{"Sprints de Alpha", "Ship the release", "sp1", "x cerrar", "h cerrados", "a asignar", "/ busca"} {
		if !strings.Contains(view, want) {
			t.Fatalf("browser view missing %q: %q", want, view)
		}
	}
	if !strings.Contains(view, "Enter") {
		t.Fatalf("browser view lacks the primary action hint: %q", view)
	}
}

func TestSprintBrowserFiltersByGoalWithoutRequests(t *testing.T) {
	var requests []string
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.URL.Path == "/projects/p1/sprints/open" {
			_, _ = w.Write([]byte(`{"sprints":[{"id":"sp1","sprint_goal":"Ship"},{"id":"sp2","sprint_goal":"Polish"}]}`))
			return true
		}
		return false
	})

	m := openSprintsWithProject(t, server.URL, "p1")
	before := len(requests)
	m, _ = apply(m, keyMsg("/"))
	m = typeText(m, "pol")
	view := m.View()
	if !strings.Contains(view, "Polish") || strings.Contains(view, "Ship") {
		t.Fatalf("filtered sprint view = %q, want only Polish", view)
	}
	if len(requests) != before {
		t.Fatalf("filtering issued requests: %v", requests[before:])
	}
	if m.Screen != "sprints" {
		t.Fatalf("Screen = %q, want sprints while filtering", m.Screen)
	}
}

func TestSprintCreateSendsProjectAndGoalAndRendersAPIError(t *testing.T) {
	var method, path string
	var payload map[string]string
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodGet && r.URL.Path == "/projects/p1/sprints/open" {
			_, _ = w.Write([]byte(`{"sprints":[]}`))
			return true
		}
		if r.Method == http.MethodPost && r.URL.Path == "/projects/p1/sprints" {
			method, path = r.Method, r.URL.EscapedPath()
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &payload)
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"conflict","message":"project unavailable"}`))
			return true
		}
		return false
	})

	m := openSprintsWithProject(t, server.URL, "p1")
	m = typeText(m, "n")
	if m.Screen != "sprint-goal" || !strings.Contains(m.View(), "Objetivo") {
		t.Fatalf("creation screen = %q view = %q, want the goal field", m.Screen, m.View())
	}
	m = typeText(m, "Sprint objective")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submitting goal must issue an API request")
	}
	_ = cmd()
	if method != http.MethodPost || path != "/projects/p1/sprints" || payload["sprint_goal"] != "Sprint objective" {
		t.Fatalf("request = %s %s %#v", method, path, payload)
	}
	result, _ := updated.(tui.Model).Update(tui.SprintResultMsg{Op: "create", Response: tui.APIResponse{StatusCode: http.StatusConflict, Body: []byte(`{"error":"conflict","message":"project unavailable"}`)}})
	if !strings.Contains(result.(tui.Model).View(), "project unavailable") {
		t.Fatalf("error response not rendered: %q", result.(tui.Model).View())
	}
}

func TestSprintAssignmentSelectsOpenSprintAndPreservesSelectedStoryIDs(t *testing.T) {
	var requests []recordedRequest
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, recordedRequest{Method: r.Method, Path: r.URL.EscapedPath(), Body: body})
		switch r.URL.Path {
		case "/projects/p1/sprints/open":
			_, _ = w.Write([]byte(`{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Goal","is_closed":false}]}`))
		case "/projects/p1/stories":
			_, _ = w.Write([]byte(`{"project_id":"p1","stories":[{"id":"story-1","title":"One"},{"id":"story-2","title":"Two"}]}`))
		default:
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"story_already_assigned","message":"already assigned"}`))
		}
		return true
	})

	m := openSprintsWithProject(t, server.URL, "p1")
	m, cmd := apply(m, keyMsg("a"))
	if cmd == nil {
		t.Fatal("assigning stories must load the backlog")
	}
	m = run(t, m, cmd)
	if m.Screen != "sprint-story-select" {
		t.Fatalf("screen = %q, want story selection", m.Screen)
	}
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeySpace)
	m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submitting selected stories must issue batch request")
	}
	m, _ = apply(m, cmd())
	if !strings.Contains(m.View(), "story_already_assigned") {
		t.Fatalf("API conflict was not rendered: %q", m.View())
	}
	last := requests[len(requests)-1]
	if last.Method != http.MethodPost || last.Path != "/projects/p1/sprints/sp1/stories" {
		t.Fatalf("assignment request = %+v", last)
	}
	var body struct {
		StoryIDs []string `json:"story_ids"`
	}
	if err := json.Unmarshal(last.Body, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.StoryIDs) != 1 || body.StoryIDs[0] != "story-2" {
		t.Fatalf("story_ids = %v, want exact selected ID story-2", body.StoryIDs)
	}
}

func TestSprintStoryActionsCreateTasksAndComplete(t *testing.T) {
	var paths []string
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		paths = append(paths, r.Method+" "+r.URL.EscapedPath())
		switch r.URL.Path {
		case "/projects/p1/sprints/open":
			_, _ = w.Write([]byte(`{"sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Goal"}]}`))
		case "/projects/p1/sprints/sp1/stories":
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"stories":[{"id":"story1","title":"Story"}]}`))
				return true
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
		return true
	})

	openAssigned := func(t *testing.T) tui.Model {
		t.Helper()
		m := openSprintsWithProject(t, server.URL, "p1")
		m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
		m = run(t, m, cmd)
		if m.Screen != "sprint-assigned-list" {
			t.Fatalf("screen = %q, want assigned story list", m.Screen)
		}
		return m
	}

	t.Run("enter opens the task form", func(t *testing.T) {
		m := openAssigned(t)
		m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
		if cmd != nil || m.Screen != "sprint-task-form" {
			t.Fatalf("task selection = screen %q, cmd %v; want task form without API request", m.Screen, cmd != nil)
		}
	})

	t.Run("c completes the highlighted story", func(t *testing.T) {
		m := openAssigned(t)
		m, cmd := apply(m, keyMsg("c"))
		if cmd == nil {
			t.Fatal("completing a story must issue the completion request")
		}
		result := cmd()
		if got := result.(tui.SprintResultMsg).Op; got != "complete" {
			t.Fatalf("operation = %q, want complete", got)
		}
		if last := paths[len(paths)-1]; last != "POST /projects/p1/sprints/sp1/stories/story1/completion" {
			t.Fatalf("completion request = %q", last)
		}
	})
}

func TestTaskClientOmitsOptionalEstimatedHours(t *testing.T) {
	server, rec := recordingServer(t, http.StatusCreated, `{"tasks":[{"id":"t1","title":"Implement"}]}`)
	hours := 2.5
	response, err := tui.NewTaskClient(server.URL).Create(t.Context(), "p1", "sp1", "story1", []tui.TaskInput{{Title: "Implement"}, {Title: "Test", EstimatedHours: &hours}})
	if err != nil {
		t.Fatal(err)
	}
	request := onlyRequest(t, rec)
	if request.Method != http.MethodPost || request.Path != "/projects/p1/sprints/sp1/stories/story1/tasks" {
		t.Fatalf("request = %+v", request)
	}
	var body map[string][]map[string]json.RawMessage
	if err := json.Unmarshal(request.Body, &body); err != nil {
		t.Fatal(err)
	}
	if len(body["tasks"]) != 2 || string(body["tasks"][0]["title"]) != `"Implement"` || string(body["tasks"][1]["title"]) != `"Test"` {
		t.Fatalf("tasks body = %s", request.Body)
	}
	if _, exists := body["tasks"][0]["estimated_hours"]; exists {
		t.Fatalf("optional estimated_hours was sent: %s", request.Body)
	}
	if got := string(body["tasks"][1]["estimated_hours"]); got != "2.5" {
		t.Fatalf("estimated_hours = %s, want 2.5", got)
	}
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", response.StatusCode)
	}
}

func TestTaskConflictResponseIsShownWithoutSuccess(t *testing.T) {
	m := tui.NewModel("http://localhost")
	updated, _ := m.Update(tui.SprintResultMsg{Op: "tasks", Response: tui.APIResponse{StatusCode: http.StatusConflict, Body: []byte(`{"error":"story_not_in_sprint","message":"story is not assigned to the selected sprint"}`)}})
	view := updated.(tui.Model).View()
	if !strings.Contains(view, "story_not_in_sprint") || strings.Contains(view, "Tareas creadas") {
		t.Fatalf("task conflict view = %q", view)
	}
}

func TestSprintClientCompletesStoryWithinSprintAndPreservesConflict(t *testing.T) {
	server, rec := recordingServer(t, http.StatusConflict, `{"error":"story_already_completed","message":"story already completed in sprint"}`)
	response, err := tui.NewSprintClient(server.URL).Complete(t.Context(), "p1", "sp1", "story1")
	if err != nil {
		t.Fatal(err)
	}
	request := onlyRequest(t, rec)
	if request.Method != http.MethodPost || request.Path != "/projects/p1/sprints/sp1/stories/story1/completion" || len(request.Body) != 0 {
		t.Fatalf("completion request = %+v", request)
	}
	if response.StatusCode != http.StatusConflict || !strings.Contains(string(response.Body), "story_already_completed") {
		t.Fatalf("response = %+v, want preserved repeated-completion conflict", response)
	}
}

func TestCompletionConflictsAreRenderedAsAPIOutcomes(t *testing.T) {
	for _, code := range []string{"sprint_closed", "story_not_in_sprint", "story_already_completed"} {
		t.Run(code, func(t *testing.T) {
			m := tui.NewModel("http://localhost")
			updated, _ := m.Update(tui.SprintResultMsg{Op: "complete", Response: tui.APIResponse{StatusCode: http.StatusConflict, Body: []byte(`{"error":"` + code + `","message":"API rejected completion"}`)}})
			view := updated.(tui.Model).View()
			if !strings.Contains(view, code) || strings.Contains(view, "Historia completada") {
				t.Fatalf("completion conflict view = %q", view)
			}
		})
	}
}
