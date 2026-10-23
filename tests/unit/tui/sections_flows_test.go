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

func TestPlanningPokerParticipantSelectorShowsLoadingAndAPIError(t *testing.T) {
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		switch r.URL.Path {
		case "/projects/p1/stories":
			_, _ = w.Write([]byte(storiesBodyForSection(sectionStories())))
			return true
		case "/projects/p1/members":
			http.Error(w, `{"message":"member service unavailable"}`, http.StatusServiceUnavailable)
			return true
		}
		return false
	})
	m := openSectionWithProject(t, server.URL, "p1", menuPlanning)
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("choosing a story should start loading participants")
	}
	view := stripANSI(m.View())
	if !strings.Contains(view, "Cargando") || strings.Contains(view, "No hay participantes disponibles") {
		t.Fatalf("loading state should not claim the project has no participants: %q", view)
	}
	m = run(t, m, cmd)
	view = stripANSI(m.View())
	if !strings.Contains(view, "503") || strings.Contains(view, "No hay participantes disponibles") {
		t.Fatalf("API failure should be shown instead of an empty-project message: %q", view)
	}
}

func TestPlanningPokerFootersAdvertiseContextualSelectionAndVoteKeys(t *testing.T) {
	server := planningServer(t)
	m := openSectionWithProject(t, server.URL, "p1", menuPlanning)
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(t, m, cmd)
	view := stripANSI(m.View())
	if !strings.Contains(view, "Space") {
		t.Fatalf("participant footer should advertise Space to toggle selection: %q", view)
	}
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeySpace})
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	view = stripANSI(m.View())
	if strings.Contains(view, "?  ayuda") || strings.Contains(view, "? ayuda") {
		t.Fatalf("vote footer must not advertise ? as help because it is a vote card: %q", view)
	}
	if !strings.Contains(view, "↑/↓") || !strings.Contains(view, "participante") {
		t.Fatalf("vote footer should describe arrows for participant navigation: %q", view)
	}
}

func TestPlanningPokerCollectsHiddenVotesRevealsResetsAndSaves(t *testing.T) {
	server := planningServer(t)
	m := openSectionWithProject(t, server.URL, "p1", menuPlanning)
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(t, m, cmd)
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeySpace})
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyDown})
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeySpace})
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := stripANSI(m.View()); !strings.Contains(got, "Ada Lovelace") || !strings.Contains(got, "Grace Hopper") {
		t.Fatalf("vote entry should name both selected participants: %q", got)
	}
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'8'}})
	if got := stripANSI(m.View()); !strings.Contains(got, "voto cargado") || strings.Contains(got, "voto cargado 8") {
		t.Fatalf("loaded vote should show only a generic marker, never its value: %q", got)
	}
	if got := stripANSI(m.View()); strings.Contains(got, "Ada Lovelace: 8") {
		t.Fatalf("hidden vote leaked before reveal: %q", got)
	}
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := stripANSI(m.View()); !strings.Contains(got, "revelar") || strings.Contains(got, "Ada Lovelace: 8") || strings.Contains(got, "Ada Lovelace: ?") {
		t.Fatalf("votes must remain hidden until explicit reveal: %q", got)
	}
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	view := stripANSI(m.View())
	for _, want := range []string{"Ada Lovelace", "8", "Grace Hopper", "?", "difieren"} {
		if !strings.Contains(view, want) {
			t.Errorf("revealed round lacks %q: %q", want, view)
		}
	}
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if got := stripANSI(m.View()); strings.Contains(got, "Ada Lovelace: 8") {
		t.Fatalf("new round leaked prior votes: %q", got)
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
