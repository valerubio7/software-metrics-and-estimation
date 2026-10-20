package tui_test

import (
	"net/http"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestSprintBrowserRefreshKeepsTheFilter checks that r re-reads the open Sprint
// list without discarding what the user searched for.
func TestSprintBrowserRefreshKeepsTheFilter(t *testing.T) {
	openRequests := 0
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path == "/projects/p1/sprints/open" {
			openRequests++
			_, _ = w.Write([]byte(`{"sprints":[{"id":"sp1","sprint_goal":"Ship"},{"id":"sp2","sprint_goal":"Polish"}]}`))
			return true
		}
		return false
	})

	m := openSprintsWithProject(t, server.URL, "p1")
	m, _ = apply(m, keyMsg("/"))
	m = typeText(m, "pol")
	m = press(m, tea.KeyEnter) // stop searching, keep the filter
	m, cmd := apply(m, keyMsg("r"))
	if cmd == nil {
		t.Fatal("r must refresh the open Sprint list")
	}
	m = run(t, m, cmd)
	if openRequests != 2 {
		t.Fatalf("open requests = %d, want 2 (initial + refresh)", openRequests)
	}
	view := m.View()
	if !strings.Contains(view, "Polish") || strings.Contains(view, "Ship") {
		t.Fatalf("refresh dropped the filter: %q", view)
	}
}

// TestSprintBrowserEscapeClearsTheFilterFirst mirrors the project browser and the
// backlog: the first Esc clears the search, the second leaves the screen.
func TestSprintBrowserEscapeClearsTheFilterFirst(t *testing.T) {
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path == "/projects/p1/sprints/open" {
			_, _ = w.Write([]byte(`{"sprints":[{"id":"sp1","sprint_goal":"Ship"}]}`))
			return true
		}
		return false
	})

	m := openSprintsWithProject(t, server.URL, "p1")
	m, _ = apply(m, keyMsg("/"))
	m = typeText(m, "zzz")
	m = press(m, tea.KeyEsc)
	if m.Screen != "sprints" {
		t.Fatalf("first Esc left screen %q, want sprints", m.Screen)
	}
	view := m.View()
	if !strings.Contains(view, "Ship") || strings.Contains(view, "Buscar:") {
		t.Fatalf("first Esc did not clear the filter: %q", view)
	}
	m = press(m, tea.KeyEsc)
	if m.Screen != "menu" {
		t.Fatalf("second Esc left screen %q, want menu", m.Screen)
	}
}

// TestSprintScreenChangeProjectReturnsToBrowser checks the global p shortcut from
// a Sprint screen resumes on the Sprint browser of the new project.
func TestSprintScreenChangeProjectReturnsToBrowser(t *testing.T) {
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		if strings.HasSuffix(r.URL.Path, "/sprints/open") {
			_, _ = w.Write([]byte(`{"sprints":[]}`))
			return true
		}
		return false
	})

	m := openSprintsWithProject(t, server.URL, "p1")
	m, load := apply(m, keyMsg("p"))
	if m.Screen != "project-picker" {
		t.Fatalf("p = screen %q, want project-picker", m.Screen)
	}
	m, cmd := confirmProjectInPicker(t, m, load, "p2")
	m = run(t, m, cmd)
	if m.Screen != "sprints" {
		t.Fatalf("after changing project: screen %q, want sprints", m.Screen)
	}
	if !strings.Contains(m.View(), "Sprints de Beta") {
		t.Fatalf("browser kept the old project: %q", m.View())
	}
}
