package tui_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
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
	coverage := `"coverage":{"story_points_planned":{"missing_observations":1},"story_points_completed":{"missing_observations":2},"velocity":{"missing_observations":3},"estimated_hours":{"missing_observations":4},"actual_hours":{"stories_without_records":5,"unattributed_story_level_records":6}}`
	return sectionServer(t, map[string]string{
		"/projects/p1/sprints/open":        `{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Sprint de métricas","is_closed":false}]}`,
		"/projects/p1/metrics":             `{"story_points_planned":0,` + coverage + `}`,
		"/projects/p1/sprints/sp1/metrics": `{"velocity":80,` + coverage + `}`,
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

func TestMetricsFlowLoadsAPIValuesRefreshesAndStaysReadOnly(t *testing.T) {
	requests := 0
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path == "/projects/p1/metrics" {
			requests++
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			_, _ = w.Write([]byte(`{"story_points_planned":0,"story_points_completed":null,"velocity":4.5,"estimated_hours":null,"actual_hours":3,"deviation":1.5,"deviation_percentage":25,"completed_stories_percentage":50,"defects_detected":2,"defects_resolved":1,"coverage":{"story_points_planned":{"missing_observations":1},"story_points_completed":{"missing_observations":2},"velocity":{"missing_observations":3},"estimated_hours":{"missing_observations":4},"actual_hours":{"stories_without_records":5,"unattributed_story_level_records":6}}}`))
			return true
		}
		return false
	})
	m := openSectionWithProject(t, server.URL, "p1", menuMetrics)
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("choosing project scope should issue metrics GET")
	}
	m = run(t, m, cmd)
	view := stripANSI(m.View())
	for _, want := range []string{"0", "sin datos", "4.5", "3", "1.5", "25", "50", "2", "1", "Cobertura", "Faltan puntos planificados: 1", "Faltan puntos completados: 2", "Faltan observaciones velocidad: 3", "Faltan horas estimadas: 4", "Historias sin esfuerzo atribuible: 5", "Registros directos no atribuibles: 6"} {
		if !strings.Contains(view, want) {
			t.Errorf("metrics view lacks %q: %q", want, view)
		}
	}
	m, cmd = apply(m, keyMsg("r"))
	if cmd == nil {
		t.Fatal("r should refresh metrics")
	}
	m = run(t, m, cmd)
	if requests != 2 || m.Screen != "section" {
		t.Fatalf("requests/screen = %d/%q, want 2/section", requests, m.Screen)
	}
}

func TestMetricsErrorKeepsScreenOpenAndRetries(t *testing.T) {
	requests := 0
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path == "/projects/p1/metrics" {
			requests++
			http.Error(w, `{"message":"metrics unavailable"}`, http.StatusServiceUnavailable)
			return true
		}
		return false
	})
	m := openSectionWithProject(t, server.URL, "p1", menuMetrics)
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(t, m, cmd)
	view := stripANSI(m.View())
	if m.Screen != "section" || !strings.Contains(view, "503") || !strings.Contains(view, "metrics unavailable") {
		t.Fatalf("error view = %q", view)
	}
	m, cmd = apply(m, keyMsg("r"))
	if cmd == nil {
		t.Fatal("r should retry")
	}
	m = run(t, m, cmd)
	if requests != 2 || m.Screen != "section" {
		t.Fatalf("requests/screen = %d/%q", requests, m.Screen)
	}
}

func TestMetricsMalformedResponseShowsDecodeReasonAndCanRetry(t *testing.T) {
	requests := 0
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path != "/projects/p1/metrics" {
			return false
		}
		requests++
		if requests == 1 {
			_, _ = w.Write([]byte(`{"story_points_planned":"not-a-number"}`))
		} else {
			_, _ = w.Write([]byte(`{"story_points_planned":4}`))
		}
		return true
	})
	m := openSectionWithProject(t, server.URL, "p1", menuMetrics)
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(t, m, cmd)
	view := stripANSI(m.View())
	if m.Screen != "section" || !strings.Contains(view, "cannot unmarshal") {
		t.Fatalf("decode error should be visible without closing the TUI: %q", view)
	}
	m, cmd = apply(m, keyMsg("r"))
	if cmd == nil {
		t.Fatal("r should retry a malformed metrics response")
	}
	m = run(t, m, cmd)
	view = stripANSI(m.View())
	if requests != 2 || m.Screen != "section" || !strings.Contains(view, "Puntos planificados: 4") {
		t.Fatalf("retry should recover to metrics, requests=%d view=%q", requests, view)
	}
}

func TestMetricsIgnoresLateResponseAfterRefresh(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var releaseOnce sync.Once
	var requestsMu sync.Mutex
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/projects" {
			_, _ = w.Write([]byte(projectsBody(t, sampleProjects())))
			return
		}
		if r.URL.Path != "/projects/p1/metrics" {
			http.NotFound(w, r)
			return
		}
		requestsMu.Lock()
		requests++
		requestNumber := requests
		requestsMu.Unlock()
		if requestNumber == 1 {
			close(firstStarted)
			<-releaseFirst
			_, _ = w.Write([]byte(`{"story_points_planned":1}`))
			return
		}
		_, _ = w.Write([]byte(`{"story_points_planned":80}`))
	}))
	defer server.Close()
	defer releaseOnce.Do(func() { close(releaseFirst) })

	m := openSectionWithProject(t, server.URL, "p1", menuMetrics)
	m, firstCmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	firstResult := make(chan tea.Msg, 1)
	go func() { firstResult <- firstCmd() }()
	select {
	case <-firstStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("first metrics request did not start")
	}
	m, refreshCmd := apply(m, keyMsg("r"))
	if refreshCmd == nil {
		t.Fatal("r should start a second metrics request")
	}
	m = run(t, m, refreshCmd)
	releaseOnce.Do(func() { close(releaseFirst) })
	updated, _ := m.Update(<-firstResult)
	m = updated.(tui.Model)
	view := stripANSI(m.View())
	if !strings.Contains(view, "Puntos planificados: 80") || strings.Contains(view, "Puntos planificados: 1") {
		t.Fatalf("late pre-refresh response replaced the refreshed value: %q", view)
	}
}

func TestMetricsIgnoresLateResponseAfterChangingScope(t *testing.T) {
	projectStarted := make(chan struct{})
	releaseProject := make(chan struct{})
	var releaseOnce sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/projects":
			_, _ = w.Write([]byte(projectsBody(t, sampleProjects())))
		case "/projects/p1/metrics":
			close(projectStarted)
			<-releaseProject
			_, _ = w.Write([]byte(`{"story_points_planned":1}`))
		case "/projects/p1/sprints/open":
			_, _ = w.Write([]byte(`{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Sprint actual","is_closed":false}]}`))
		case "/projects/p1/sprints/sp1/metrics":
			_, _ = w.Write([]byte(`{"story_points_planned":80}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	defer releaseOnce.Do(func() { close(releaseProject) })

	m := openSectionWithProject(t, server.URL, "p1", menuMetrics)
	m, projectCmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if projectCmd == nil {
		t.Fatal("choosing project scope should issue metrics GET")
	}
	projectResult := make(chan tea.Msg, 1)
	go func() { projectResult <- projectCmd() }()
	select {
	case <-projectStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("project metrics request did not start")
	}

	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEsc})
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyDown})
	m, sprintListCmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(t, m, sprintListCmd)
	m, sprintMetricsCmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if sprintMetricsCmd == nil {
		t.Fatal("choosing a Sprint should issue metrics GET")
	}
	m = run(t, m, sprintMetricsCmd)

	releaseOnce.Do(func() { close(releaseProject) })
	updated, _ := m.Update(<-projectResult)
	m = updated.(tui.Model)
	view := stripANSI(m.View())
	if !strings.Contains(view, "80") || strings.Contains(view, "Puntos planificados: 1") {
		t.Fatalf("late project response replaced the selected Sprint metrics: %q", view)
	}
}

func TestMetricsRenderFitsNarrowTerminal(t *testing.T) {
	server := sectionServer(t, map[string]string{"/projects/p1/metrics": `{"story_points_planned":1,"story_points_completed":2,"velocity":3,"estimated_hours":4,"actual_hours":5,"deviation":6,"deviation_percentage":7,"completed_stories_percentage":8,"defects_detected":9,"defects_resolved":10,"coverage":{"story_points_planned":{"missing_observations":1},"story_points_completed":{"missing_observations":2},"velocity":{"missing_observations":3},"estimated_hours":{"missing_observations":4},"actual_hours":{"stories_without_records":5,"unattributed_story_level_records":6}}}`})
	m := openSectionWithProject(t, server.URL, "p1", menuMetrics)
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(t, m, cmd)
	for _, width := range []int{80, 44} {
		updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 44})
		m = updated.(tui.Model)
		for _, line := range strings.Split(m.View(), "\n") {
			if lipgloss.Width(line) > width {
				t.Errorf("width %d line is %d: %q", width, lipgloss.Width(line), line)
			}
		}
	}
}

func TestMetricsScopeSelection(t *testing.T) {
	for _, tc := range []struct {
		name string
		keys []tea.KeyMsg
		want []string
	}{
		{
			name: "project",
			keys: []tea.KeyMsg{{Type: tea.KeyEnter}},
			want: []string{"Proyecto", "Cobertura", "0"},
		},
		{
			name: "sprint",
			keys: []tea.KeyMsg{{Type: tea.KeyDown}, {Type: tea.KeyEnter}},
			want: []string{"Sprint de métricas", "Cobertura", "80"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := metricsServer(t)
			m := openSectionWithProject(t, server.URL, "p1", menuMetrics)
			view := stripANSI(m.View())
			if !strings.Contains(view, "Proyecto") || !strings.Contains(view, "Sprint") {
				t.Fatalf("metrics scope selector should offer project and Sprint: %q", view)
			}
			var cmd tea.Cmd
			for _, key := range tc.keys {
				m, cmd = apply(m, key)
			}
			if tc.name == "sprint" {
				if cmd == nil {
					t.Fatal("Sprint scope should load the existing Sprint selector")
				}
				m = run(t, m, cmd)
				m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
				if cmd == nil {
					t.Fatal("choosing a Sprint should load its metrics")
				}
				m = run(t, m, cmd)
			}
			if tc.name == "project" {
				if cmd == nil {
					t.Fatal("choosing project should load metrics")
				}
				m = run(t, m, cmd)
			}
			view = stripANSI(m.View())
			for _, want := range tc.want {
				if !strings.Contains(view, want) {
					t.Errorf("view lacks %q: %q", want, view)
				}
			}
		})
	}
}

func TestMetricsAndDashboardShowTheirFutureNoticeAfterChoosingASprint(t *testing.T) {
	for _, tc := range []struct {
		name  string
		index int
		issue string
	}{
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
