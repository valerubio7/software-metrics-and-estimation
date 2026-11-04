package tui_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

// Workspace tab digits, in tab bar order.
const (
	tabDashboard  = "1"
	tabBacklog    = "2"
	tabSprints    = "3"
	tabEstimation = "4"
	tabEffort     = "5"
	tabDefects    = "6"
)

// openSectionWithProject goes browser -> pid -> section tab (Estimación or
// Esfuerzo) and applies the primary collection response.
func openSectionWithProject(t *testing.T, url, pid, digit string) tui.Model {
	t.Helper()
	m, cmd := openTab(t, url, pid, digit)
	if cmd == nil {
		t.Fatal("expected a command to load the section")
	}
	m = run(t, m, cmd)
	if m.Screen != "section" {
		t.Fatalf("Screen = %q after opening tab %s, want section", m.Screen, digit)
	}
	return m
}

// openStoryFlowWithProject goes browser -> pid -> Backlog -> detail of the
// storyIndex-th story, presses actionKey (p or t) and applies the command of the
// first step of the flow. The flow starts with the story already chosen, so it
// never passes through a story selector (N1: additive until the old helper goes).
func openStoryFlowWithProject(t *testing.T, url, pid string, storyIndex int, actionKey string) tui.Model {
	t.Helper()
	m := openStoryDetailWithProject(t, url, pid, storyIndex)
	m, cmd := apply(m, keyMsg(actionKey))
	if cmd == nil {
		t.Fatalf("%s on the detail should issue the command of the first step", actionKey)
	}
	m = run(t, m, cmd)
	if m.Screen != "section" {
		t.Fatalf("Screen = %q after %s on the detail, want section", m.Screen, actionKey)
	}
	return m
}

// sectionServer answers GET /projects plus the paths supplied by the caller.
func sectionServer(t *testing.T, routes map[string]string) *httptest.Server {
	t.Helper()
	return sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		body, ok := routes[r.URL.Path]
		if !ok {
			return false
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
		return true
	})
}

func sectionStories() []tui.Story {
	return []tui.Story{
		{ID: "s1", ProjectID: "p1", Title: "Primera historia", Priority: "alta", Status: "pendiente", StoryPoints: intPtr(3)},
		{ID: "s2", ProjectID: "p1", Title: "Segunda historia", Priority: "media", Status: "en progreso"},
	}
}

func storiesBodyForSection(stories []tui.Story) string {
	if stories == nil {
		stories = []tui.Story{}
	}
	encoded := `{"project_id":"p1","stories":[`
	for i, story := range stories {
		if i > 0 {
			encoded += ","
		}
		encoded += `{"id":"` + story.ID + `","project_id":"p1","title":"` + story.Title + `","priority":"` + story.Priority + `","status":"` + story.Status + `"}`
	}
	return encoded + `]}`
}

func TestTabsRouteToEachSprint3Section(t *testing.T) {
	for _, tc := range []struct {
		name  string
		digit string
		title string
		body  string
		item  string
	}{
		{"planning", tabEstimation, "Estimación", storiesBodyForSection(sectionStories()), "Primera historia"},
		{"effort", tabEffort, "Esfuerzo", storiesBodyForSection(sectionStories()), "Primera historia"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := "/projects/p1/stories"
			server := sectionServer(t, map[string]string{path: tc.body})
			m := openSectionWithProject(t, server.URL, "p1", tc.digit)
			view := stripANSI(m.View())
			for _, want := range []string{tc.title, tc.item, "Proyectos › Alpha › " + tc.title} {
				if !strings.Contains(view, want) {
					t.Errorf("section view lacks %q: %q", want, view)
				}
			}
			if strings.Contains(view, "submenú") {
				t.Errorf("section view looks like an operations submenu: %q", view)
			}
		})
	}
}

// TestDefectsTabOpensProjectDefectList covers TUI-09 replacing the former
// placeholder section with the project-scoped defect listing.
func TestDefectsTabOpensProjectDefectList(t *testing.T) {
	server := sectionServer(t, map[string]string{
		"/projects/p1/defects": `{"project_id":"p1","defects":[{"id":"d1","project_id":"p1","story_id":"s1","sprint_id":"sp1","description":"Primer defecto","severity":"alta","status":"detectado","resolution_sprint_id":null}]}`,
	})
	m, cmd := openTab(t, server.URL, "p1", tabDefects)
	if cmd == nil {
		t.Fatal("opening Defectos should load the project defect list")
	}
	m = run(t, m, cmd)
	if m.Screen != "defects" {
		t.Fatalf("Screen = %q after opening Defectos, want defects", m.Screen)
	}
	view := stripANSI(m.View())
	for _, want := range []string{"Defectos", "Primer defecto", "alta", "detectado"} {
		if !strings.Contains(view, want) {
			t.Errorf("defect list lacks %q: %q", want, view)
		}
	}
}

func TestSectionAPIErrorStaysOpenAndRefreshes(t *testing.T) {
	requests := 0
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path == "/projects/p1/stories" {
			requests++
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"internal_error","message":"an unexpected error occurred"}`))
			return true
		}
		return false
	})

	m := openSectionWithProject(t, server.URL, "p1", tabEstimation)
	view := stripANSI(m.View())
	if m.Screen != "section" || !strings.Contains(view, "Error HTTP 500") {
		t.Fatalf("Screen/view = %q / %q, want section with an inline HTTP error", m.Screen, view)
	}
	m, cmd := apply(m, keyMsg("r"))
	if cmd == nil {
		t.Fatal("r should retry the failed section collection")
	}
	m = run(t, m, cmd)
	if requests != 2 || m.Screen != "section" {
		t.Fatalf("requests/screen = %d/%q, want 2/section", requests, m.Screen)
	}
}

func TestSectionEmptyCollectionShowsAnEmptyState(t *testing.T) {
	server := sectionServer(t, map[string]string{"/projects/p1/stories": storiesBodyForSection([]tui.Story{})})
	m := openSectionWithProject(t, server.URL, "p1", tabEstimation)
	view := stripANSI(m.View())
	if !strings.Contains(view, "No hay historias") {
		t.Fatalf("empty section view = %q, want an empty state", view)
	}
}

func TestSectionFilterRefreshAndEscape(t *testing.T) {
	requests := 0
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path == "/projects/p1/stories" {
			requests++
			_, _ = w.Write([]byte(storiesBodyForSection(sectionStories())))
			return true
		}
		return false
	})
	m := openSectionWithProject(t, server.URL, "p1", tabEstimation)

	m, _ = apply(m, keyMsg("/"))
	m = typeText(m, "seg")
	view := stripANSI(m.View())
	if !strings.Contains(view, "Segunda historia") || strings.Contains(view, "Primera historia") {
		t.Fatalf("filtered view = %q, want only the match", view)
	}
	m = press(m, tea.KeyEnter) // stop searching, keep the filter
	m, cmd := apply(m, keyMsg("r"))
	if cmd == nil {
		t.Fatal("r should refresh the section collection")
	}
	m = run(t, m, cmd)
	if requests != 2 {
		t.Fatalf("requests = %d, want 2 (initial + refresh)", requests)
	}
	view = stripANSI(m.View())
	if !strings.Contains(view, "Buscar: seg") || !strings.Contains(view, "Segunda historia") || strings.Contains(view, "Primera historia") {
		t.Fatalf("refresh dropped the filter: %q", view)
	}

	m = press(m, tea.KeyEsc)
	if m.Screen != "section" || strings.Contains(stripANSI(m.View()), "Buscar:") || !strings.Contains(stripANSI(m.View()), "Primera historia") {
		t.Fatalf("first Esc did not clear the filter: screen=%q view=%q", m.Screen, stripANSI(m.View()))
	}
	m = press(m, tea.KeyEsc)
	if m.Screen != "project-picker" {
		t.Fatalf("second Esc left screen %q, want project-picker", m.Screen)
	}
}

func TestSectionFooterAndHelpOnlyShowValidKeys(t *testing.T) {
	server := sectionServer(t, map[string]string{"/projects/p1/stories": storiesBodyForSection(sectionStories())})
	m := openSectionWithProject(t, server.URL, "p1", tabEstimation)
	footer := stripANSI(m.View())
	for _, want := range []string{"elegir historia", "?  más"} {
		if !strings.Contains(footer, want) {
			t.Errorf("section footer lacks %q: %q", want, footer)
		}
	}

	for _, unwanted := range []string{"p  proyecto", "Esc  menú"} {
		if strings.Contains(footer, unwanted) {
			t.Errorf("section footer still shows the removed %q: %q", unwanted, footer)
		}
	}

	m, _ = apply(m, keyMsg("?"))
	help := stripANSI(m.View())
	if strings.Contains(help, "Cambiar el proyecto activo") {
		t.Errorf("section help still offers the removed project shortcut: %q", help)
	}
	for _, want := range []string{"Mover la selección", "Salir"} {
		if !strings.Contains(help, want) {
			t.Errorf("section help lacks %q: %q", want, help)
		}
	}
	for _, unwanted := range []string{"Crear un elemento nuevo", "Editar el elemento resaltado", "Cerrar el Sprint resaltado", "Ver los Sprints cerrados"} {
		if strings.Contains(help, unwanted) {
			t.Errorf("section help shows unavailable action %q: %q", unwanted, help)
		}
	}
}
