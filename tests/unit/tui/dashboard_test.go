package tui_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

const dashboardMetricsBody = `{"project_id":"p1","story_points_planned":21,"story_points_completed":13,"velocity":13,"estimated_hours":34.5,"actual_hours":30.5,"deviation":-4,"completed_stories_percentage":61.9,"defects_detected":3,"defects_resolved":1}`

func dashboardServer(t *testing.T) *httptest.Server {
	t.Helper()
	return sectionServer(t, map[string]string{
		"/projects/p1":         `{"id":"p1","name":"Alpha","status":"active"}`,
		"/projects/p1/metrics": dashboardMetricsBody,
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
