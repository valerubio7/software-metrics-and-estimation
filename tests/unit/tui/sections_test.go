package tui_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

const (
	menuPlanning  = 3
	menuEffort    = 4
	menuDefects   = 5
	menuMetrics   = 6
	menuDashboard = 7
)

// startSection opens the section at index from the menu. Without an active
// project the section lands on the project picker and returns its load command.
func startSection(url string, index int) (tui.Model, tea.Cmd) {
	m := tui.NewModel(url)
	for i := 0; i < index; i++ {
		m = press(m, tea.KeyDown)
	}
	return apply(m, tea.KeyMsg{Type: tea.KeyEnter})
}

// openSectionWithProject goes menu -> section -> picker -> pid -> Enter and
// applies the primary collection response.
func openSectionWithProject(t *testing.T, url, pid string, index int) tui.Model {
	t.Helper()
	m, load := startSection(url, index)
	m, cmd := confirmProjectInPicker(t, m, load, pid)
	m = run(t, m, cmd)
	if m.Screen != "section" {
		t.Fatalf("Screen = %q after opening section %d, want section", m.Screen, index)
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

func TestMenuExposesNineOptions(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")
	want := []string{"Proyectos", "Backlog", "Sprints", "Planning Poker", "Esfuerzo", "Defectos", "Métricas", "Dashboard", "Salir"}
	if len(m.Choices) != len(want) {
		t.Fatalf("Choices = %#v, want %d options", m.Choices, len(want))
	}
	for i, option := range want {
		if m.Choices[i] != option {
			t.Errorf("Choices[%d] = %q, want %q", i, m.Choices[i], option)
		}
		if !strings.Contains(m.View(), option) {
			t.Errorf("View() lacks menu option %q", option)
		}
	}
}

func TestMenuRoutesToEachSprint3Section(t *testing.T) {
	for _, tc := range []struct {
		name  string
		index int
		title string
		body  string
		item  string
	}{
		{"planning", menuPlanning, "Planning Poker", storiesBodyForSection(sectionStories()), "Primera historia"},
		{"effort", menuEffort, "Esfuerzo", storiesBodyForSection(sectionStories()), "Primera historia"},
		{"defects", menuDefects, "Defectos", storiesBodyForSection(sectionStories()), "Primera historia"},
		{"metrics", menuMetrics, "Métricas", `{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Sprint base","is_closed":false}]}`, "Sprint base"},
		{"dashboard", menuDashboard, "Dashboard", `{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Sprint base","is_closed":false}]}`, "Sprint base"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := "/projects/p1/stories"
			if tc.index == menuMetrics || tc.index == menuDashboard {
				path = "/projects/p1/sprints/open"
			}
			server := sectionServer(t, map[string]string{path: tc.body})
			m := openSectionWithProject(t, server.URL, "p1", tc.index)
			view := stripANSI(m.View())
			for _, want := range []string{tc.title, tc.item, "Proyecto: Alpha"} {
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

	m := openSectionWithProject(t, server.URL, "p1", menuPlanning)
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
	m := openSectionWithProject(t, server.URL, "p1", menuPlanning)
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
	m := openSectionWithProject(t, server.URL, "p1", menuPlanning)

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
	if m.Screen != "menu" {
		t.Fatalf("second Esc left screen %q, want menu", m.Screen)
	}
}

func TestSectionFooterAndHelpOnlyShowValidKeys(t *testing.T) {
	server := sectionServer(t, map[string]string{"/projects/p1/stories": storiesBodyForSection(sectionStories())})
	m := openSectionWithProject(t, server.URL, "p1", menuPlanning)
	footer := stripANSI(m.View())
	for _, want := range []string{"elegir historia", "p  proyecto", "?  ayuda"} {
		if !strings.Contains(footer, want) {
			t.Errorf("section footer lacks %q: %q", want, footer)
		}
	}

	m, _ = apply(m, keyMsg("?"))
	help := stripANSI(m.View())
	for _, want := range []string{"Mover la selección", "Cambiar el proyecto activo", "Salir"} {
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
