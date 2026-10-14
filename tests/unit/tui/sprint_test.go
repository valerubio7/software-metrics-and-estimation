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
