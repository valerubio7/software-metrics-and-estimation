package tui_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

func TestSprintCloseAndHistoryFlows(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/projects/p1/sprints/open":
			_, _ = w.Write([]byte(`{"sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Ship","is_closed":false}]}`))
		case "/projects/p1/sprints/sp1/closure":
			_, _ = w.Write([]byte(`{"project_id":"p1","sprint_id":"sp1","is_closed":true}`))
		case "/projects/p1/sprints":
			_, _ = w.Write([]byte(`{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Ship","is_closed":true,"stories":[{"id":"st1","title":"Done work","completed":true},{"id":"st2","title":"Remaining work","completed":false}]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	m := tui.NewModel(server.URL)
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeyEnter)
	if !strings.Contains(m.View(), "Cerrar Sprint") || !strings.Contains(m.View(), "historial") {
		t.Fatalf("Sprint menu lacks separate close/history actions: %q", m.View())
	}
	for range 4 {
		m = press(m, tea.KeyDown)
	}
	m = press(m, tea.KeyEnter)
	m = typeText(m, "p1")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("project submission must list open Sprints")
	}
	updated, _ = updated.(tui.Model).Update(cmd())
	m = updated.(tui.Model)
	if m.Screen != "sprint-open-list" {
		t.Fatalf("screen = %q, want open Sprint selector", m.Screen)
	}
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("selecting an open Sprint must issue closure")
	}
	updated, _ = updated.(tui.Model).Update(cmd())
	m = updated.(tui.Model)
	if !strings.Contains(m.View(), "remain available") {
		t.Fatalf("closure success lacks history guidance: %q", m.View())
	}
	if strings.Join(requests, "|") != "GET /projects/p1/sprints/open|POST /projects/p1/sprints/sp1/closure" {
		t.Fatalf("closure requests = %v", requests)
	}
	m = press(m, tea.KeyEnter)
	for range 5 {
		m = press(m, tea.KeyDown)
	} // history
	m = press(m, tea.KeyEnter)
	m = typeText(m, "p1")
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("history submission must issue GET")
	}
	updated, _ = updated.(tui.Model).Update(cmd())
	m = updated.(tui.Model)
	view := m.View()
	for _, want := range []string{"Goal: Ship", "Done work [Completed]", "Remaining work [Not completed]"} {
		if !strings.Contains(view, want) {
			t.Fatalf("history view missing %q: %q", want, view)
		}
	}
	if len(requests) != 3 || requests[2] != "GET /projects/p1/sprints" {
		t.Fatalf("history must be read-only: %v", requests)
	}
}

func TestSprintClosureConflictAndEmptyHistory(t *testing.T) {
	m := tui.NewModel("http://localhost")
	updated, _ := m.Update(tui.SprintResultMsg{Op: "close", Response: tui.APIResponse{StatusCode: http.StatusConflict, Body: []byte(`{"error":"sprint_already_closed","message":"Sprint was already closed"}`)}})
	if got := updated.(tui.Model).View(); !strings.Contains(got, "sprint_already_closed") || strings.Contains(got, "remain available") {
		t.Fatalf("closure conflict view = %q", got)
	}
	updated, _ = m.Update(tui.SprintResultMsg{Op: "history", Response: tui.APIResponse{StatusCode: http.StatusOK, Body: []byte(`{"project_id":"p1","sprints":[]}`)}})
	if got := updated.(tui.Model).View(); !strings.Contains(got, "No closed Sprints") {
		t.Fatalf("empty history view = %q", got)
	}
}
