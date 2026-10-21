package tui_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func planningServer(t *testing.T) *httptest.Server {
	t.Helper()
	return sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		switch r.URL.Path {
		case "/projects/p1/stories":
			_, _ = w.Write([]byte(storiesBodyForSection(sectionStories())))
			return true
		case "/projects/p1/members":
			_, _ = w.Write([]byte(`{"project_id":"p1","members":[{"id":"m1","project_id":"p1","full_name":"Ada Lovelace","email":"ada@example.com"},{"id":"m2","project_id":"p1","full_name":"Grace Hopper","email":null}]}`))
			return true
		}
		return false
	})
}

func effortServer(t *testing.T) *httptest.Server {
	t.Helper()
	return sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		switch r.URL.Path {
		case "/projects/p1/stories":
			_, _ = w.Write([]byte(storiesBodyForSection(sectionStories())))
			return true
		case "/projects/p1/stories/s1/tasks":
			_, _ = w.Write([]byte(`{"project_id":"p1","story_id":"s1","tasks":[{"id":"t1","project_id":"p1","sprint_id":"sp1","story_id":"s1","title":"Diseñar","estimated_hours":4.5},{"id":"t2","project_id":"p1","sprint_id":"sp1","story_id":"s1","title":"Probar"}]}`))
			return true
		}
		return false
	})
}

func defectsServer(t *testing.T) *httptest.Server {
	t.Helper()
	return sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		switch r.URL.Path {
		case "/projects/p1/stories":
			_, _ = w.Write([]byte(storiesBodyForSection(sectionStories())))
			return true
		case "/projects/p1/sprints/open":
			_, _ = w.Write([]byte(`{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Sprint de detección","is_closed":false}]}`))
			return true
		}
		return false
	})
}

func metricsServer(t *testing.T) *httptest.Server {
	t.Helper()
	return sectionServer(t, map[string]string{
		"/projects/p1/sprints/open": `{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Sprint de métricas","is_closed":false}]}`,
	})
}

func TestPlanningPokerFlowChoosesStoryAndMemberThenShowsTheTUI07Notice(t *testing.T) {
	server := planningServer(t)
	m := openSectionWithProject(t, server.URL, "p1", menuPlanning)

	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("choosing a story should load the member selector")
	}
	m = run(t, m, cmd)
	view := stripANSI(m.View())
	for _, want := range []string{"Planning Poker", "Participantes", "Ada Lovelace", "Grace Hopper"} {
		if !strings.Contains(view, want) {
			t.Errorf("member selector lacks %q: %q", want, view)
		}
	}

	m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("choosing a member should not issue a command, got %v", cmd)
	}
	view = stripANSI(m.View())
	for _, want := range []string{"TUI-07", "Ada Lovelace", "No se guardó ningún dato"} {
		if !strings.Contains(view, want) {
			t.Errorf("planning notice lacks %q: %q", want, view)
		}
	}
	m = press(m, tea.KeyEsc)
	if !strings.Contains(stripANSI(m.View()), "Participantes") {
		t.Fatalf("Esc from the notice should return to the member selector: %q", stripANSI(m.View()))
	}
}

func TestEffortFlowChoosesStoryAndTaskThenShowsTheTUI08Notice(t *testing.T) {
	server := effortServer(t)
	m := openSectionWithProject(t, server.URL, "p1", menuEffort)

	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("choosing a story should load its tasks")
	}
	m = run(t, m, cmd)
	view := stripANSI(m.View())
	for _, want := range []string{"Tareas", "Diseñar", "Probar"} {
		if !strings.Contains(view, want) {
			t.Errorf("task selector lacks %q: %q", want, view)
		}
	}

	m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("choosing a task should not issue a command, got %v", cmd)
	}
	view = stripANSI(m.View())
	for _, want := range []string{"TUI-08", "Diseñar", "No se guardó ningún dato"} {
		if !strings.Contains(view, want) {
			t.Errorf("effort notice lacks %q: %q", want, view)
		}
	}
}

func TestDefectsFlowChoosesStoryAndSprintThenShowsTheTUI09Notice(t *testing.T) {
	server := defectsServer(t)
	m := openSectionWithProject(t, server.URL, "p1", menuDefects)

	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("choosing a story should load the Sprint selector")
	}
	m = run(t, m, cmd)
	view := stripANSI(m.View())
	if !strings.Contains(view, "Sprint de detección") {
		t.Fatalf("Sprint selector lacks the Sprint: %q", view)
	}

	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	view = stripANSI(m.View())
	for _, want := range []string{"TUI-09", "Sprint de detección", "No se guardó ningún dato"} {
		if !strings.Contains(view, want) {
			t.Errorf("defect notice lacks %q: %q", want, view)
		}
	}
}

func TestMetricsAndDashboardShowTheirFutureNoticeAfterChoosingASprint(t *testing.T) {
	for _, tc := range []struct {
		name  string
		index int
		issue string
	}{
		{"metrics", menuMetrics, "TUI-10"},
		{"dashboard", menuDashboard, "TUI-11"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := metricsServer(t)
			m := openSectionWithProject(t, server.URL, "p1", tc.index)
			m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
			view := stripANSI(m.View())
			for _, want := range []string{tc.issue, "Sprint de métricas", "No se guardó ningún dato"} {
				if !strings.Contains(view, want) {
					t.Errorf("notice lacks %q: %q", want, view)
				}
			}
		})
	}
}

func TestSectionNavigationOnlyEmitsGetRequests(t *testing.T) {
	rec := &recorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rec.mu.Lock()
		rec.requests = append(rec.requests, recordedRequest{Method: r.Method, Path: r.URL.EscapedPath(), Body: body})
		rec.mu.Unlock()
		switch r.URL.Path {
		case "/projects":
			_, _ = w.Write([]byte(projectsBody(t, sampleProjects())))
		case "/projects/p1/stories":
			_, _ = w.Write([]byte(storiesBodyForSection(sectionStories())))
		case "/projects/p1/members":
			_, _ = w.Write([]byte(`{"project_id":"p1","members":[{"id":"m1","project_id":"p1","full_name":"Ada Lovelace","email":null}]}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(server.Close)

	m := openSectionWithProject(t, server.URL, "p1", menuPlanning)
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(t, m, cmd)
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})

	for _, request := range rec.all() {
		if request.Method != http.MethodGet {
			t.Errorf("navigation emitted %s %s, want only GET", request.Method, request.Path)
		}
	}
	if len(rec.all()) == 0 {
		t.Fatal("navigation emitted no requests")
	}
}
