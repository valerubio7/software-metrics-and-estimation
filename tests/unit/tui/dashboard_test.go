package tui_test

import (
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
