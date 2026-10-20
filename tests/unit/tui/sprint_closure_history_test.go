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

	// Close flow: x asks for confirmation, Esc cancels, s closes.
	m := openSprintsWithProject(t, server.URL, "p1")
	view := m.View()
	for _, want := range []string{"Ship", "x cerrar", "h cerrados", "a asignar"} {
		if !strings.Contains(view, want) {
			t.Fatalf("Sprint browser missing %q: %q", want, view)
		}
	}
	m, cmd := apply(m, keyMsg("x"))
	if cmd != nil || m.Screen != "sprint-confirm" {
		t.Fatalf("x = screen %q, cmd %v; want the confirmation screen without a request", m.Screen, cmd != nil)
	}
	if !strings.Contains(m.View(), "¿Cerrar este Sprint?") {
		t.Fatalf("confirmation view = %q", m.View())
	}
	m = press(m, tea.KeyEsc)
	if m.Screen != "sprints" {
		t.Fatalf("cancelling left screen %q, want sprints", m.Screen)
	}
	m, cmd = apply(m, keyMsg("x"))
	if cmd != nil {
		t.Fatal("opening the confirmation issued a request")
	}
	m, cmd = apply(m, keyMsg("s"))
	if cmd == nil {
		t.Fatal("confirming must issue the closure request")
	}
	m = run(t, m, cmd)
	if !strings.Contains(m.View(), "cerrado") {
		t.Fatalf("closure success lacks history guidance: %q", m.View())
	}
	if got := strings.Join(requests, "|"); got != "GET /projects|GET /projects/p1/sprints/open|POST /projects/p1/sprints/sp1/closure" {
		t.Fatalf("closure requests = %v", requests)
	}

	// History flow with a fresh model against the same server.
	requests = nil
	history := openSprintsWithProject(t, server.URL, "p1")
	history, cmd = apply(history, keyMsg("h"))
	if cmd == nil {
		t.Fatal("history selection must issue GET")
	}
	history = run(t, history, cmd)
	if history.Screen != "sprint-history" {
		t.Fatalf("screen = %q, want the history view", history.Screen)
	}
	view = history.View()
	if !strings.Contains(view, "Ship") || !strings.Contains(view, "1/2 historias completadas") {
		t.Fatalf("history view = %q", view)
	}
	if strings.Contains(view, "Done work") {
		t.Fatalf("history rows must stay collapsed until Enter: %q", view)
	}
	history, cmd = apply(history, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expanding a closed Sprint must not issue a request")
	}
	view = history.View()
	for _, want := range []string{"Done work", "Remaining work"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expanded history missing %q: %q", want, view)
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
