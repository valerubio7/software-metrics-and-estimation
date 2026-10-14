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

func TestSprintMenuOpensGoalCreation(t *testing.T) {
	m := tui.NewModel("http://localhost")
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeyEnter)
	if !strings.Contains(m.View(), "Crear Sprint") {
		t.Fatalf("Sprint workflow view = %q, want creation action", m.View())
	}
	m = press(m, tea.KeyEnter)
	m = typeText(m, "p1")
	m = press(m, tea.KeyEnter)
	if !strings.Contains(m.View(), "Objetivo") {
		t.Fatalf("creation view = %q, want goal field", m.View())
	}
}

func TestSprintCreateSendsProjectAndGoalAndRendersAPIError(t *testing.T) {
	var method, path string
	var payload map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.EscapedPath()
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &payload)
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"conflict","message":"project unavailable"}`))
	}))
	defer server.Close()

	m := tui.NewModel(server.URL)
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeyEnter)
	m = press(m, tea.KeyEnter)
	m = typeText(m, "p1")
	m = press(m, tea.KeyEnter)
	m = typeText(m, "Sprint objective")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submitting goal must issue an API request")
	}
	_ = updated
	_ = cmd()
	if method != http.MethodPost || path != "/projects/p1/sprints" || payload["sprint_goal"] != "Sprint objective" {
		t.Fatalf("request = %s %s %#v", method, path, payload)
	}
	result, _ := updated.(tui.Model).Update(tui.SprintResultMsg{Response: tui.APIResponse{StatusCode: http.StatusConflict, Body: []byte(`{"error":"conflict","message":"project unavailable"}`)}})
	if !strings.Contains(result.(tui.Model).View(), "project unavailable") {
		t.Fatalf("error response not rendered: %q", result.(tui.Model).View())
	}
}

func TestSprintAssignmentSelectsOpenSprintAndPreservesSelectedStoryIDs(t *testing.T) {
	var requests []recordedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()
	m := tui.NewModel(server.URL)
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeyEnter)
	m = press(m, tea.KeyDown) // batch assignment
	m = press(m, tea.KeyEnter)
	m = typeText(m, "p1")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	if cmd == nil {
		t.Fatal("selecting open Sprint must load the project's open Sprints")
	}
	updated, _ = m.Update(cmd())
	m = updated.(tui.Model)
	if m.Screen != "sprint-open-list" {
		t.Fatalf("screen = %q, want open Sprint selection", m.Screen)
	}
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	if cmd == nil {
		t.Fatal("selecting Sprint must load project backlog")
	}
	updated, _ = m.Update(cmd())
	m = updated.(tui.Model)
	if m.Screen != "sprint-story-select" {
		t.Fatalf("screen = %q, want story selection", m.Screen)
	}
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeySpace)
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submitting selected stories must issue batch request")
	}
	response := cmd()
	updated, _ = updated.(tui.Model).Update(response)
	if !strings.Contains(updated.(tui.Model).View(), "story_already_assigned") {
		t.Fatalf("API conflict was not rendered: %q", updated.(tui.Model).View())
	}
	if len(requests) != 3 {
		t.Fatalf("requests = %d, want discovery, backlog and assignment", len(requests))
	}
	last := requests[2]
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

func TestSprintTaskAndCompletionMenusNavigateThroughAssignedStories(t *testing.T) {
	for _, tt := range []struct {
		name       string
		menuDowns  int
		wantScreen string
		wantOp     string
	}{
		{name: "create tasks", menuDowns: 2, wantScreen: "sprint-task-form", wantOp: "tasks"},
		{name: "complete story", menuDowns: 3, wantScreen: "sprint-result", wantOp: "complete"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var paths []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.Method+" "+r.URL.EscapedPath())
				switch r.URL.Path {
				case "/projects/p1/sprints/open":
					_, _ = w.Write([]byte(`{"sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Goal"}]}`))
				case "/projects/p1/sprints/sp1/stories":
					_, _ = w.Write([]byte(`{"stories":[{"id":"story1","title":"Story"}]}`))
				default:
					_, _ = w.Write([]byte(`{"ok":true}`))
				}
			}))
			defer server.Close()

			m := tui.NewModel(server.URL)
			m = press(m, tea.KeyDown)
			m = press(m, tea.KeyDown)
			m = press(m, tea.KeyEnter)
			for range tt.menuDowns {
				m = press(m, tea.KeyDown)
			}
			m = press(m, tea.KeyEnter)
			m = typeText(m, "p1")
			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			if cmd == nil {
				t.Fatal("entering a project ID must request open Sprints")
			}
			updated, _ = updated.(tui.Model).Update(cmd())
			updated, cmd = updated.(tui.Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
			if cmd == nil {
				t.Fatal("selecting an open Sprint must request assigned stories")
			}
			updated, _ = updated.(tui.Model).Update(cmd())
			m = updated.(tui.Model)
			if len(paths) != 2 || paths[0] != "GET /projects/p1/sprints/open" || paths[1] != "GET /projects/p1/sprints/sp1/stories" {
				t.Fatalf("discovery requests = %v, want open Sprints then assigned stories", paths)
			}
			if m.Screen != "sprint-assigned-list" {
				t.Fatalf("screen = %q, want assigned story selection", m.Screen)
			}

			updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			if tt.wantOp == "tasks" {
				if cmd != nil || updated.(tui.Model).Screen != tt.wantScreen {
					t.Fatalf("task selection = screen %q, cmd %v; want task form without API request", updated.(tui.Model).Screen, cmd != nil)
				}
				return
			}
			if cmd == nil {
				t.Fatal("selecting a story to complete must issue the completion request")
			}
			result := cmd()
			if got := result.(tui.SprintResultMsg).Op; got != tt.wantOp {
				t.Fatalf("operation = %q, want %q", got, tt.wantOp)
			}
			if updated.(tui.Model).Screen != "sprint-assigned-list" || paths[2] != "POST /projects/p1/sprints/sp1/stories/story1/completion" {
				t.Fatalf("completion state/requests = screen %q, paths %v", updated.(tui.Model).Screen, paths)
			}
		})
	}
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
	if !strings.Contains(view, "story_not_in_sprint") || strings.Contains(view, "Tasks created") {
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
			if !strings.Contains(view, code) || strings.Contains(view, "Story completed") {
				t.Fatalf("completion conflict view = %q", view)
			}
		})
	}
}
