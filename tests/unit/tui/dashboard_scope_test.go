package tui_test

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

const dashboardSprintsOpenBody = `{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Sprint de métricas","is_closed":false}]}`

const dashboardSprintMetricsBody = `{"project_id":"p1","sprint_id":"sp1","story_points_planned":8,"story_points_completed":5,"velocity":5,"estimated_hours":14,"actual_hours":12.5,"deviation":-1.5,"completed_stories_percentage":62.5,"defects_detected":2,"defects_resolved":null}`

// dashboardScopeServer answers the project, its metrics, the open Sprints, the
// Sprint metrics and both PDFs, recording every request.
func dashboardScopeServer(t *testing.T, rec *recorder) string {
	t.Helper()
	server := reportServer(t, rec, func(w http.ResponseWriter, r *http.Request) bool {
		switch r.URL.Path {
		case "/projects/p1/sprints/open":
			_, _ = w.Write([]byte(dashboardSprintsOpenBody))
		case "/projects/p1/sprints/sp1/metrics":
			_, _ = w.Write([]byte(dashboardSprintMetricsBody))
		case "/projects/p1/sprints/sp1/report.pdf", "/projects/p1/report.pdf":
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF-1.7\nreport"))
		default:
			return false
		}
		return true
	})
	return server.URL
}

// openDashboardScopes starts the Dashboard, presses s and applies the open
// Sprints load.
func openDashboardScopes(t *testing.T, url string) tui.Model {
	t.Helper()
	m := startDashboard(t, url, "p1")
	m, _ = apply(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m, cmd := apply(m, keyMsg("s"))
	if m.Screen != "dashboard-sprint-select" || cmd == nil {
		t.Fatalf("s should open the scope selector and load open Sprints: screen=%q cmd=%v", m.Screen, cmd != nil)
	}
	return run(t, m, cmd)
}

// chooseDashboardSprint picks the first open Sprint and applies its metrics.
func chooseDashboardSprint(t *testing.T, url string) tui.Model {
	t.Helper()
	m := openDashboardScopes(t, url)
	m = press(m, tea.KeyDown)
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != "dashboard" || cmd == nil {
		t.Fatalf("choosing a Sprint should return to the Dashboard and load its metrics: screen=%q", m.Screen)
	}
	return run(t, m, cmd)
}

func TestDashboardScopeSelectorListsProjectAndOpenSprints(t *testing.T) {
	rec := &recorder{}
	m := openDashboardScopes(t, dashboardScopeServer(t, rec))
	view := stripANSI(m.View())
	project, sprint := strings.Index(view, "Proyecto completo"), strings.Index(view, "Sprint de métricas")
	if project < 0 || sprint < 0 || project > sprint {
		t.Fatalf("selector should list «Proyecto completo» then the open Sprints: %q", view)
	}
	if !strings.Contains(view, "Proyectos › Alpha › Dashboard › Elegir ámbito") {
		t.Errorf("selector breadcrumb missing: %q", view)
	}
	if !strings.Contains(view, "Dashboard") || requestCount(rec, http.MethodGet, "/projects/p1/sprints/open") != 1 {
		t.Errorf("selector should belong to the Dashboard tab and GET the open Sprints: %q", view)
	}
	// Digits type into the filter: the selector is not a tab root.
	m, _ = apply(m, keyMsg("2"))
	if m.Screen != "dashboard-sprint-select" {
		t.Fatalf("digit left the selector: screen=%q", m.Screen)
	}
}

func TestDashboardSprintScopeShowsSprintMetricsExactly(t *testing.T) {
	rec := &recorder{}
	m := chooseDashboardSprint(t, dashboardScopeServer(t, rec))
	view := stripANSI(m.View())
	if requestCount(rec, http.MethodGet, "/projects/p1/sprints/sp1/metrics") != 1 {
		t.Fatalf("choosing the Sprint should GET its metrics: %v", rec.all())
	}
	for _, want := range []string{"Métricas del Sprint", "Sprint de métricas", "5 / 8", "12.5 / 14 h", "62.5%", "Velocidad: 5", "Desviación: -1.5", "█"} {
		if !strings.Contains(view, want) {
			t.Errorf("sprint scope view lacks %q: %q", want, view)
		}
	}
	if line := viewLine(view, "Defectos"); !strings.Contains(line, "sin datos") {
		t.Errorf("null resolved defects should show sin datos: %q", line)
	}
	if strings.Contains(view, "Métricas del proyecto") || strings.Contains(view, "13 / 21") {
		t.Errorf("sprint scope still shows project metrics: %q", view)
	}
	for _, request := range rec.all() {
		if request.Method != http.MethodGet {
			t.Errorf("dashboard emitted %s %s, want only GET", request.Method, request.Path)
		}
	}
}

func TestDashboardWholeProjectScopeReturnsToProjectMetrics(t *testing.T) {
	url := dashboardScopeServer(t, &recorder{})
	m := chooseDashboardSprint(t, url)
	m, cmd := apply(m, keyMsg("s"))
	m = run(t, m, cmd)
	m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(t, m, cmd)
	view := stripANSI(m.View())
	if m.Screen != "dashboard" || !strings.Contains(view, "Métricas del proyecto") || !strings.Contains(view, "13 / 21") {
		t.Fatalf("«Proyecto completo» should show project metrics: %q", view)
	}
	if strings.Contains(view, "Métricas del Sprint") {
		t.Errorf("project scope still titled as Sprint: %q", view)
	}
}

func TestDashboardScopeSelectorEscKeepsScope(t *testing.T) {
	rec := &recorder{}
	m := chooseDashboardSprint(t, dashboardScopeServer(t, rec))
	m, cmd := apply(m, keyMsg("s"))
	m = run(t, m, cmd)
	before := len(rec.all())
	m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		m = run(t, m, cmd)
	}
	view := stripANSI(m.View())
	if m.Screen != "dashboard" || !strings.Contains(view, "Métricas del Sprint") || !strings.Contains(view, "5 / 8") {
		t.Fatalf("Esc should return to the Sprint-scoped Dashboard: screen=%q view=%q", m.Screen, view)
	}
	if got := len(rec.all()); got != before {
		t.Errorf("Esc issued %d requests, want none", got-before)
	}
}

func TestDashboardDownloadsSprintPDFInSprintScope(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	rec := &recorder{}
	m := chooseDashboardSprint(t, dashboardScopeServer(t, rec))
	m, _ = apply(m, tea.WindowSizeMsg{Width: 300, Height: 40})
	m, cmd := apply(m, keyMsg("d"))
	if cmd == nil {
		t.Fatal("d should download the Sprint report")
	}
	m = run(t, m, cmd)
	if got := recordedReports(rec.all(), "/projects/p1/sprints/sp1/report.pdf"); len(got) != 1 || got[0].Method != http.MethodGet {
		t.Fatalf("sprint report requests = %v", got)
	}
	if got := recordedReports(rec.all(), "/projects/p1/report.pdf"); len(got) != 0 {
		t.Errorf("sprint scope downloaded the project report")
	}
	path := filepath.Join(dir, "reporte-sprint-sp1.pdf")
	if view := stripANSI(m.View()); !strings.Contains(view, "Reporte guardado: "+path) {
		t.Errorf("dashboard lacks the saved path %q: %q", path, view)
	}
}

func TestDashboardRefreshReloadsTheSprintScope(t *testing.T) {
	rec := &recorder{}
	m := chooseDashboardSprint(t, dashboardScopeServer(t, rec))
	projectBefore := requestCount(rec, http.MethodGet, "/projects/p1/metrics")
	m, cmd := apply(m, keyMsg("r"))
	if cmd == nil {
		t.Fatal("r should reload the Sprint scope")
	}
	m = run(t, m, cmd)
	if got := requestCount(rec, http.MethodGet, "/projects/p1/sprints/sp1/metrics"); got != 2 {
		t.Errorf("sprint metrics requests = %d, want 2", got)
	}
	if got := requestCount(rec, http.MethodGet, "/projects/p1/metrics"); got != projectBefore {
		t.Errorf("r in Sprint scope requested the project metrics")
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "Métricas del Sprint") || !strings.Contains(view, "5 / 8") {
		t.Errorf("refresh lost the Sprint scope: %q", view)
	}
}

func TestDashboardDiscardsStaleScopeResults(t *testing.T) {
	url := dashboardScopeServer(t, &recorder{})
	m := openDashboardScopes(t, url)
	m = press(m, tea.KeyDown)
	m, stale := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	// Before the Sprint answer arrives, switch back to the whole project.
	m, cmd := apply(m, keyMsg("s"))
	m = run(t, m, cmd)
	m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(t, m, cmd)
	m = run(t, m, stale)
	view := stripANSI(m.View())
	if !strings.Contains(view, "Métricas del proyecto") || !strings.Contains(view, "13 / 21") || strings.Contains(view, "5 / 8") {
		t.Fatalf("late Sprint result replaced the project scope: %q", view)
	}
}

func TestDashboardSprintMetricsErrorStaysOnScreen(t *testing.T) {
	server := reportServer(t, &recorder{}, func(w http.ResponseWriter, r *http.Request) bool {
		switch r.URL.Path {
		case "/projects/p1/sprints/open":
			_, _ = w.Write([]byte(dashboardSprintsOpenBody))
		case "/projects/p1/sprints/sp1/metrics":
			http.Error(w, `{"error":"service_unavailable","message":"metrics down"}`, http.StatusServiceUnavailable)
		default:
			return false
		}
		return true
	})
	m := chooseDashboardSprint(t, server.URL)
	view := stripANSI(m.View())
	if m.Screen != "dashboard" || !strings.Contains(view, "503") || !strings.Contains(view, "metrics down") || !strings.Contains(view, "Métricas del Sprint") {
		t.Fatalf("sprint metrics error view = %q (screen %q)", view, m.Screen)
	}
}

func TestDashboardScopeFooterAndHelp(t *testing.T) {
	url := dashboardScopeServer(t, &recorder{})
	m := startDashboard(t, url, "p1")
	m, _ = apply(m, tea.WindowSizeMsg{Width: 200, Height: 40})
	if view := stripANSI(m.View()); !strings.Contains(view, "s  ámbito") {
		t.Errorf("dashboard footer lacks s: %q", view)
	}
	m, _ = apply(m, keyMsg("?"))
	if help := stripANSI(m.View()); !strings.Contains(help, "Elegir proyecto o Sprint para las métricas") {
		t.Errorf("dashboard help lacks s: %q", help)
	}
	m, _ = apply(m, keyMsg("?"))
	m, cmd := apply(m, keyMsg("s"))
	m = run(t, m, cmd)
	view := stripANSI(m.View())
	for _, want := range []string{"Enter  elegir ámbito", "/  buscar", "r  refrescar", "Esc  volver"} {
		if !strings.Contains(view, want) {
			t.Errorf("selector footer lacks %q: %q", want, view)
		}
	}
	for _, unwanted := range []string{"d  PDF", "s  ámbito"} {
		if strings.Contains(view, unwanted) {
			t.Errorf("selector footer shows %q: %q", unwanted, view)
		}
	}
}

func TestOldMetricsSectionIsUnreachable(t *testing.T) {
	m := startDashboard(t, dashboardServer(t).URL, "p1")
	m, _ = apply(m, tea.WindowSizeMsg{Width: 200, Height: 40})
	header := strings.Split(stripANSI(m.View()), "\n")[:4]
	if strings.Contains(strings.Join(header, "\n"), "Métricas") {
		t.Errorf("workspace header offers a Métricas tab: %q", header)
	}
	for _, key := range []string{"1", "2", "3", "4", "5", "6"} {
		m, _ = apply(m, keyMsg(key))
		if m.Screen == "section" && strings.Contains(stripANSI(m.View()), "Métricas · Ámbito") {
			t.Fatalf("digit %s opened the old Métricas section", key)
		}
		m, _ = apply(m, keyMsg("1"))
	}
}
