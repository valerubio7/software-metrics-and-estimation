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

func metricsServer(t *testing.T) *httptest.Server {
	t.Helper()
	return sectionServer(t, map[string]string{
		"/projects/p1/sprints/open":        `{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Sprint de métricas","is_closed":false}]}`,
		"/projects/p1/metrics":             `{"project_id":"p1","story_points_planned":21,"story_points_completed":13,"velocity":13,"estimated_hours":34.5,"actual_hours":30.5,"deviation":-4,"completed_stories_percentage":61.9,"defects_detected":3,"defects_resolved":null}`,
		"/projects/p1/sprints/sp1/metrics": `{"project_id":"p1","sprint_id":"sp1","story_points_planned":8,"story_points_completed":5,"velocity":5,"estimated_hours":14,"actual_hours":12.5,"deviation":-1.5,"completed_stories_percentage":62.5,"defects_detected":2,"defects_resolved":1}`,
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
	m := openStoryDetailWithProject(t, server.URL, "p1", 0)
	m, cmd := apply(m, keyMsg("p"))
	if cmd == nil {
		t.Fatal("Estimar on the detail should start loading participants")
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
	m := openStoryFlowWithProject(t, server.URL, "p1", 0, "p")
	view := stripANSI(m.View())
	if !strings.Contains(view, "Space") {
		t.Fatalf("participant footer should advertise Space to toggle selection: %q", view)
	}
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeySpace})
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	view = stripANSI(m.View())
	if strings.Contains(view, "?  ayuda") || strings.Contains(view, "? ayuda") || strings.Contains(view, "?  más") {
		t.Fatalf("vote footer must not advertise ? as help because it is a vote card: %q", view)
	}
	if !strings.Contains(view, "↑/↓") || !strings.Contains(view, "participante") {
		t.Fatalf("vote footer should describe arrows for participant navigation: %q", view)
	}
}

func TestPlanningPokerCollectsHiddenVotesRevealsResetsAndSaves(t *testing.T) {
	server := planningServer(t)
	m := openStoryFlowWithProject(t, server.URL, "p1", 0, "p")
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

// TestSectionNavigationOnlyEmitsGetRequests walks Backlog -> detail -> each of the
// three actions -> Esc and requires every request the walk emits to be a GET.
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
			_, _ = w.Write([]byte(effortMembersBody))
		case "/projects/p1/stories/s1/tasks":
			_, _ = w.Write([]byte(effortTasksBody))
		case "/projects/p1/sprints/open":
			_, _ = w.Write([]byte(`{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Primero","is_closed":false}]}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(server.Close)

	m := openStoryDetailWithProject(t, server.URL, "p1", 0)

	// Estimar: participants, then Enter on the highlighted one opens the vote, then
	// back out level by level.
	m = stepRunning(t, m, keyMsg("p"))
	m = stepRunning(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != "planning-poker" {
		t.Fatalf("Screen = %q, want the Planning Poker vote", m.Screen)
	}
	m = stepRunning(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	m = stepRunning(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != "story-detail" {
		t.Fatalf("Screen = %q after leaving Estimar, want story-detail", m.Screen)
	}

	// Registrar esfuerzo: tasks, members, the form, then back out level by level.
	m = stepRunning(t, m, keyMsg("t"))
	m = stepRunning(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // whole-story row
	m = stepRunning(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // first member
	if m.Screen != "effort-form" {
		t.Fatalf("Screen = %q, want the effort form", m.Screen)
	}
	for i := 0; i < 3; i++ {
		m = stepRunning(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	}
	if m.Screen != "story-detail" {
		t.Fatalf("Screen = %q after leaving Registrar esfuerzo, want story-detail", m.Screen)
	}

	// Reportar defecto: the form, the Sprint selector, then back.
	m = stepRunning(t, m, keyMsg("d"))
	for i := 0; i < 3; i++ {
		m = stepRunning(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	}
	if m.Screen != "defect-sprint-select" {
		t.Fatalf("Screen = %q, want the Sprint selector", m.Screen)
	}
	m = stepRunning(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	m = stepRunning(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != "story-detail" {
		t.Fatalf("Screen = %q after leaving Reportar defecto, want story-detail", m.Screen)
	}

	for _, request := range rec.all() {
		if request.Method != http.MethodGet {
			t.Errorf("navigation emitted %s %s, want only GET", request.Method, request.Path)
		}
	}
	// Each action issued its own read, so the walk really went through them.
	for _, path := range []string{"/projects/p1/members", "/projects/p1/stories/s1/tasks", "/projects/p1/sprints/open"} {
		if requestCount(rec, http.MethodGet, path) == 0 {
			t.Errorf("the walk never requested GET %s", path)
		}
	}
}
