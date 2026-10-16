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
		case "/projects":
			_, _ = w.Write([]byte(projectsBody(t, sampleProjects())))
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

	// Close flow.
	m := startSprints(server.URL)
	if !strings.Contains(m.View(), "Cerrar Sprint") || !strings.Contains(m.View(), "historial") {
		t.Fatalf("Sprint menu lacks separate close/history actions: %q", m.View())
	}
	for range 4 {
		m = press(m, tea.KeyDown)
	}
	m, load := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m, cmd := confirmProjectInPicker(t, m, load, "p1")
	m = run(t, m, cmd)
	if m.Screen != "sprint-open-list" {
		t.Fatalf("screen = %q, want open Sprint selector", m.Screen)
	}
	m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("selecting an open Sprint must issue closure")
	}
	m = run(t, m, cmd)
	if !strings.Contains(m.View(), "cerrado") {
		t.Fatalf("closure success lacks history guidance: %q", m.View())
	}
	if strings.Join(requests, "|") != "GET /projects|GET /projects/p1/sprints/open|POST /projects/p1/sprints/sp1/closure" {
		t.Fatalf("closure requests = %v", requests)
	}

	// History flow with a fresh model against the same server.
	history := startSprints(server.URL)
	for range 5 {
		history = press(history, tea.KeyDown)
	}
	history, load = apply(history, tea.KeyMsg{Type: tea.KeyEnter})
	history, cmd = confirmProjectInPicker(t, history, load, "p1")
	if cmd == nil {
		t.Fatal("history selection must issue GET")
	}
	history = run(t, history, cmd)
	if history.Screen != "sprint-history" {
		t.Fatalf("screen = %q, want the history view", history.Screen)
	}
	view := history.View()
	for _, want := range []string{"Ship", "Done work", "Remaining work", "completadas"} {
		if !strings.Contains(view, want) {
			t.Fatalf("history view missing %q: %q", want, view)
		}
	}
	if last := requests[len(requests)-1]; last != "GET /projects/p1/sprints" {
		t.Fatalf("history must be read-only: %v", requests)
	}
}

func TestSprintClosureConflictAndEmptyHistory(t *testing.T) {
	m := tui.NewModel("http://localhost")
	updated, _ := m.Update(tui.SprintResultMsg{Op: "close", Response: tui.APIResponse{StatusCode: http.StatusConflict, Body: []byte(`{"error":"sprint_already_closed","message":"Sprint was already closed"}`)}})
	if got := updated.(tui.Model).View(); !strings.Contains(got, "sprint_already_closed") || strings.Contains(got, "cerrado") {
		t.Fatalf("closure conflict view = %q", got)
	}
	updated, _ = m.Update(tui.SprintResultMsg{Op: "history", Response: tui.APIResponse{StatusCode: http.StatusOK, Body: []byte(`{"project_id":"p1","sprints":[]}`)}})
	if got := updated.(tui.Model).View(); !strings.Contains(got, "No hay Sprints cerrados") {
		t.Fatalf("empty history view = %q", got)
	}
}
