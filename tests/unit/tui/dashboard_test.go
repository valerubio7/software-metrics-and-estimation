package tui_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

const dashboardMetricsBody = `{"project_id":"p1","story_points_planned":21,"story_points_completed":13,"velocity":13,"estimated_hours":34.5,"actual_hours":30.5,"deviation":-4,"completed_stories_percentage":61.9,"defects_detected":3,"defects_resolved":1}`

func dashboardServer(t *testing.T) *httptest.Server {
	t.Helper()
	return dashboardServerWithMetrics(t, dashboardMetricsBody)
}

func dashboardServerWithMetrics(t *testing.T, metricsBody string) *httptest.Server {
	t.Helper()
	return sectionServer(t, map[string]string{
		"/projects/p1":         `{"id":"p1","name":"Alpha","status":"active"}`,
		"/projects/p1/metrics": metricsBody,
	})
}

// startDashboard opens Dashboard from the menu, confirms pid in the project
// picker and applies the dashboard load command.
func startDashboard(t *testing.T, url, pid string) tui.Model {
	t.Helper()
	m, load := startSection(url, menuDashboard)
	if m.Screen != "project-picker" {
		t.Fatalf("Screen = %q without an active project, want project-picker", m.Screen)
	}
	m, cmd := confirmProjectInPicker(t, m, load, pid)
	m = run(t, m, cmd)
	if m.Screen != "dashboard" {
		t.Fatalf("Screen = %q after confirming the project, want dashboard", m.Screen)
	}
	return m
}

func TestDashboardMenuOpensProjectDashboard(t *testing.T) {
	server := dashboardServer(t)
	m := startDashboard(t, server.URL, "p1")
	m, _ = apply(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	view := stripANSI(m.View())
	lower := strings.ToLower(view)

	for _, want := range []string{
		"Dashboard", "Alpha", "en curso",
		"Story Points", "13 / 21",
		"Horas", "30.5 / 34.5 h",
		"Historias completadas", "61.9%",
		"Defectos", "1 / 3",
	} {
		if !strings.Contains(lower, strings.ToLower(want)) {
			t.Errorf("dashboard view lacks %q: %q", want, view)
		}
	}
	if strings.Contains(view, "TUI-11") || strings.Contains(view, "No se guardó ningún dato") {
		t.Errorf("dashboard still shows the TUI-11 notice: %q", view)
	}
}

func TestDashboardAsksForAProjectWhenNoneIsActive(t *testing.T) {
	server := dashboardServer(t)
	m, load := startSection(server.URL, menuDashboard)
	if m.Screen != "project-picker" {
		t.Fatalf("Screen = %q, want project-picker", m.Screen)
	}
	m, cmd := confirmProjectInPicker(t, m, load, "p1")
	if cmd == nil {
		t.Fatal("confirming the project should load the dashboard")
	}
	m = run(t, m, cmd)
	if m.Screen != "dashboard" {
		t.Fatalf("Screen = %q after confirming the project, want dashboard", m.Screen)
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "Alpha") || !strings.Contains(strings.ToLower(view), "en curso") {
		t.Fatalf("dashboard did not load the chosen project: %q", view)
	}
}

func dashboardNullMetricsBody() string {
	return `{"project_id":"p1","story_points_planned":null,"story_points_completed":null,"velocity":null,"estimated_hours":null,"actual_hours":null,"deviation":null,"completed_stories_percentage":null,"defects_detected":null,"defects_resolved":null}`
}

func dashboardPartialMetricsBody() string {
	return `{"project_id":"p1","story_points_planned":20,"story_points_completed":10,"velocity":null,"estimated_hours":null,"actual_hours":null,"deviation":null,"completed_stories_percentage":0,"defects_detected":4,"defects_resolved":2}`
}

func viewLine(view, label string) string {
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, label+":") {
			return line
		}
	}
	return ""
}

func TestDashboardShowsBarsAndNoData(t *testing.T) {
	server := dashboardServerWithMetrics(t, dashboardPartialMetricsBody())
	m := startDashboard(t, server.URL, "p1")
	m, _ = apply(m, tea.WindowSizeMsg{Width: 80, Height: 30})
	view := stripANSI(m.View())

	if !strings.Contains(view, "█") || !strings.Contains(view, "░") {
		t.Errorf("dashboard lacks Unicode bars: %q", view)
	}
	if line := viewLine(view, "Story Points"); !strings.Contains(line, "10 / 20") {
		t.Errorf("Story Points row = %q, want the API pair", line)
	}
	if line := viewLine(view, "Horas"); !strings.Contains(line, "sin datos") || strings.Contains(line, "0") {
		t.Errorf("missing hours row = %q, want sin datos and no zero", line)
	}
	if line := viewLine(view, "Historias completadas"); !strings.Contains(line, "0%") {
		t.Errorf("completed stories row = %q, want the real zero from the API", line)
	}
	if line := viewLine(view, "Defectos"); !strings.Contains(line, "2 / 4") {
		t.Errorf("defects row = %q, want the API pair", line)
	}
	if line := viewLine(view, "Velocidad"); !strings.Contains(line, "sin datos") {
		t.Errorf("velocity row = %q, want sin datos", line)
	}
	if line := viewLine(view, "Desviación"); !strings.Contains(line, "sin datos") {
		t.Errorf("deviation row = %q, want sin datos", line)
	}
}

func TestDashboardAllNullMetricsShowNoBars(t *testing.T) {
	server := dashboardServerWithMetrics(t, dashboardNullMetricsBody())
	m := startDashboard(t, server.URL, "p1")
	m, _ = apply(m, tea.WindowSizeMsg{Width: 80, Height: 30})
	view := stripANSI(m.View())

	if got := strings.Count(view, "sin datos"); got < 6 {
		t.Errorf("dashboard has %d 'sin datos' rows, want at least 6: %q", got, view)
	}
	if strings.Contains(view, "█") || strings.Contains(view, "░") {
		t.Errorf("null metrics should not draw bars: %q", view)
	}
}

func TestDashboardFitsNarrowTerminals(t *testing.T) {
	server := dashboardServerWithMetrics(t, dashboardMetricsBody)
	for _, width := range []int{80, 44} {
		m := startDashboard(t, server.URL, "p1")
		m, _ = apply(m, tea.WindowSizeMsg{Width: width, Height: 40})
		view := stripANSI(m.View())
		for _, line := range strings.Split(view, "\n") {
			if got := utf8.RuneCountInString(line); got > width {
				t.Errorf("width %d: line is %d columns: %q", width, got, line)
			}
		}
		for _, want := range []string{"Dashboard", "en curso", "13 / 21", "█", "░"} {
			if !strings.Contains(strings.ToLower(view), strings.ToLower(want)) {
				t.Errorf("width %d: view lacks %q: %q", width, want, view)
			}
		}
	}
}

func TestDashboardBarsStayInsideThePanel(t *testing.T) {
	server := dashboardServerWithMetrics(t, dashboardMetricsBody)
	m := startDashboard(t, server.URL, "p1")
	m, _ = apply(m, tea.WindowSizeMsg{Width: 44, Height: 40})
	view := stripANSI(m.View())
	barLines := 0
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "█") || strings.Contains(line, "░") {
			barLines++
		}
	}
	if barLines < 4 {
		t.Fatalf("bar lines = %d, want at least the four charted metrics: %q", barLines, view)
	}
}

func TestDashboardShowsMetricErrorsWithoutClosingTheScreen(t *testing.T) {
	rec := &recorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		rec.mu.Lock()
		rec.requests = append(rec.requests, recordedRequest{Method: r.Method, Path: r.URL.EscapedPath(), Body: raw})
		rec.mu.Unlock()
		switch r.URL.Path {
		case "/projects":
			_, _ = w.Write([]byte(projectsBody(t, sampleProjects())))
		case "/projects/p1":
			_, _ = w.Write([]byte(`{"id":"p1","name":"Alpha","status":"active"}`))
		case "/projects/p1/metrics":
			http.Error(w, `{"error":"service_unavailable","message":"metrics down"}`, http.StatusServiceUnavailable)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(server.Close)

	m := startDashboard(t, server.URL, "p1")
	view := stripANSI(m.View())
	if m.Screen != "dashboard" || !strings.Contains(view, "503") || !strings.Contains(view, "metrics down") {
		t.Fatalf("metrics error view = %q (screen %q)", view, m.Screen)
	}
	if !strings.Contains(strings.ToLower(view), "en curso") {
		t.Errorf("project status should stay visible when metrics fail: %q", view)
	}

	m, cmd := apply(m, keyMsg("r"))
	if cmd == nil {
		t.Fatal("r should retry the dashboard reads")
	}
	m = run(t, m, cmd)
	if m.Screen != "dashboard" || !strings.Contains(stripANSI(m.View()), "503") {
		t.Fatalf("retry should keep the dashboard open with the error: screen=%q view=%q", m.Screen, stripANSI(m.View()))
	}
	if got := len(rec.all()); got != 5 {
		t.Fatalf("requests = %d, want the picker plus the initial and retried project/metrics reads", got)
	}
}

func TestDashboardShowsProjectStatusErrorsAndKeepsMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/projects":
			_, _ = w.Write([]byte(projectsBody(t, sampleProjects())))
		case "/projects/p1":
			http.Error(w, `{"error":"internal_error","message":"status down"}`, http.StatusInternalServerError)
		case "/projects/p1/metrics":
			_, _ = w.Write([]byte(dashboardMetricsBody))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(server.Close)

	m := startDashboard(t, server.URL, "p1")
	view := stripANSI(m.View())
	if m.Screen != "dashboard" || !strings.Contains(view, "500") || !strings.Contains(view, "status down") {
		t.Fatalf("status error view = %q (screen %q)", view, m.Screen)
	}
	if !strings.Contains(view, "sin datos") || !strings.Contains(view, "13 / 21") {
		t.Errorf("metrics should stay visible when the status fails: %q", view)
	}
}

func TestDashboardEscapeReturnsToTheMenu(t *testing.T) {
	server := dashboardServer(t)
	m := startDashboard(t, server.URL, "p1")
	m = press(m, tea.KeyEsc)
	if m.Screen != "menu" {
		t.Fatalf("Screen = %q after Esc, want menu", m.Screen)
	}
}

func TestDashboardSwitchesProjectWithP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/projects":
			_, _ = w.Write([]byte(projectsBody(t, sampleProjects())))
		case "/projects/p1":
			_, _ = w.Write([]byte(`{"id":"p1","name":"Alpha","status":"active"}`))
		case "/projects/p1/metrics":
			_, _ = w.Write([]byte(dashboardMetricsBody))
		case "/projects/p2":
			_, _ = w.Write([]byte(`{"id":"p2","name":"Beta","status":"planned"}`))
		case "/projects/p2/metrics":
			_, _ = w.Write([]byte(`{"project_id":"p2","story_points_planned":5,"story_points_completed":2,"velocity":2,"estimated_hours":8,"actual_hours":8,"deviation":0,"completed_stories_percentage":40,"defects_detected":0,"defects_resolved":0}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(server.Close)

	m := startDashboard(t, server.URL, "p1")
	m, cmd := apply(m, keyMsg("p"))
	if m.Screen != "project-picker" {
		t.Fatalf("Screen = %q after p, want project-picker", m.Screen)
	}
	m, cmd = confirmProjectInPicker(t, m, cmd, "p2")
	m = run(t, m, cmd)
	view := stripANSI(m.View())
	if m.Screen != "dashboard" || !strings.Contains(view, "Beta") || !strings.Contains(view, "2 / 5") {
		t.Fatalf("dashboard after switching project = %q (screen %q)", view, m.Screen)
	}
	if !strings.Contains(strings.ToLower(view), "planificado") {
		t.Errorf("dashboard lacks the new project status: %q", view)
	}
}

func TestDashboardOnlyEmitsGETRequests(t *testing.T) {
	rec := &recorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		rec.mu.Lock()
		rec.requests = append(rec.requests, recordedRequest{Method: r.Method, Path: r.URL.EscapedPath(), Body: raw})
		rec.mu.Unlock()
		switch r.URL.Path {
		case "/projects":
			_, _ = w.Write([]byte(projectsBody(t, sampleProjects())))
		case "/projects/p1":
			_, _ = w.Write([]byte(`{"id":"p1","name":"Alpha","status":"active"}`))
		case "/projects/p1/metrics":
			_, _ = w.Write([]byte(dashboardMetricsBody))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(server.Close)

	m := startDashboard(t, server.URL, "p1")
	m, cmd := apply(m, keyMsg("r"))
	m = run(t, m, cmd)
	for _, request := range rec.all() {
		if request.Method != http.MethodGet {
			t.Errorf("dashboard emitted %s %s, want only GET", request.Method, request.Path)
		}
	}
	if len(rec.all()) == 0 {
		t.Fatal("dashboard emitted no requests")
	}
}

func TestDashboardFooterAndHelpOnlyShowValidKeys(t *testing.T) {
	server := dashboardServer(t)
	m := startDashboard(t, server.URL, "p1")
	view := stripANSI(m.View())
	for _, want := range []string{"r  refrescar", "p  proyecto", "Esc  menú", "?  ayuda", "q  salir"} {
		if !strings.Contains(view, want) {
			t.Errorf("dashboard footer lacks %q: %q", want, view)
		}
	}

	m, _ = apply(m, keyMsg("?"))
	help := stripANSI(m.View())
	for _, want := range []string{"Refrescar el estado y las métricas", "Cambiar el proyecto activo", "Volver al menú", "Salir"} {
		if !strings.Contains(help, want) {
			t.Errorf("dashboard help lacks %q: %q", want, help)
		}
	}
	for _, unwanted := range []string{"Crear un elemento nuevo", "Editar el elemento resaltado", "Cerrar el Sprint resaltado", "Ver los Sprints cerrados"} {
		if strings.Contains(help, unwanted) {
			t.Errorf("dashboard help shows unavailable action %q: %q", unwanted, help)
		}
	}
}
