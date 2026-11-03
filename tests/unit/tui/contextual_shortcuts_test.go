package tui_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

// shortcutServer serves the projects, the backlog of p1 (stories, or an empty
// one), its members and the tasks of every story, recording each request.
func shortcutServer(t *testing.T, stories []tui.Story) (*httptest.Server, *recorder) {
	t.Helper()
	rec := &recorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.mu.Lock()
		rec.requests = append(rec.requests, recordedRequest{Method: r.Method, Path: r.URL.EscapedPath()})
		rec.mu.Unlock()
		path := r.URL.Path
		switch {
		case path == "/projects":
			_, _ = w.Write([]byte(projectsBody(t, sampleProjects())))
		case path == "/projects/p1/stories":
			_, _ = w.Write([]byte(storiesBodyForSection(stories)))
		case path == "/projects/p1/members":
			_, _ = w.Write([]byte(`{"project_id":"p1","members":[{"id":"m1","project_id":"p1","full_name":"Ada Lovelace","email":null}]}`))
		case strings.HasSuffix(path, "/tasks"):
			_, _ = w.Write([]byte(`{"tasks":[{"id":"t1","title":"Diseñar pantalla"}]}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(server.Close)
	return server, rec
}

func requestedPath(rec *recorder, want string) bool {
	for _, request := range rec.all() {
		if request.Method+" "+request.Path == want {
			return true
		}
	}
	return false
}

// backlogOnSecondStory opens the Backlog tab of p1 and highlights its second story.
func backlogOnSecondStory(t *testing.T, url string) tui.Model {
	t.Helper()
	m := openBacklogWithProject(t, url, "p1")
	return press(m, tea.KeyDown)
}

// shortcut presses key on the backlog and applies the load it issues.
func shortcut(t *testing.T, m tui.Model, key string) tui.Model {
	t.Helper()
	m, cmd := apply(m, keyMsg(key))
	return run(t, m, cmd)
}

func TestBacklogPOpensEstimationParticipantsForHighlightedStory(t *testing.T) {
	server, rec := shortcutServer(t, sectionStories())
	m := shortcut(t, backlogOnSecondStory(t, server.URL), "P")

	if m.Screen != "section" {
		t.Fatalf("Screen = %q after P, want section", m.Screen)
	}
	if !requestedPath(rec, "GET /projects/p1/members") {
		t.Fatalf("P did not request the members: %+v", rec.all())
	}
	view := stripANSI(m.View())
	for _, want := range []string{"Estimación", "Participantes", "Segunda historia", "Ada Lovelace"} {
		if !strings.Contains(view, want) {
			t.Errorf("view after P lacks %q: %q", want, view)
		}
	}
}

func TestBacklogEOpensEffortTasksForHighlightedStory(t *testing.T) {
	server, rec := shortcutServer(t, sectionStories())
	m := shortcut(t, backlogOnSecondStory(t, server.URL), "E")

	if m.Screen != "section" {
		t.Fatalf("Screen = %q after E, want section", m.Screen)
	}
	if !requestedPath(rec, "GET /projects/p1/stories/s2/tasks") {
		t.Fatalf("E did not request the tasks of s2: %+v", rec.all())
	}
	view := stripANSI(m.View())
	for _, want := range []string{"Esfuerzo · Tareas", "Segunda historia", "Diseñar pantalla"} {
		if !strings.Contains(view, want) {
			t.Errorf("view after E lacks %q: %q", want, view)
		}
	}
}

func TestEscFromPreselectedSelectorReturnsToBacklog(t *testing.T) {
	for _, key := range []string{"P", "E"} {
		t.Run(key, func(t *testing.T) {
			server, rec := shortcutServer(t, sectionStories())
			m := shortcut(t, backlogOnSecondStory(t, server.URL), key)
			before := len(rec.all())

			m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEsc})
			if m.Screen != "backlog" {
				t.Fatalf("Screen = %q after Esc, want backlog", m.Screen)
			}
			m = run(t, m, cmd)
			requests := rec.all()[before:]
			if len(requests) != 1 || requests[0].Method+" "+requests[0].Path != "GET /projects/p1/stories" {
				t.Fatalf("Esc requests = %+v, want one backlog reload", requests)
			}
			if view := stripANSI(m.View()); !strings.Contains(view, "Segunda historia") {
				t.Errorf("backlog after Esc lacks its stories: %q", view)
			}
		})
	}
}

func TestDeeperEscStillGoesBackOneStep(t *testing.T) {
	server, _ := shortcutServer(t, sectionStories())
	m := shortcut(t, backlogOnSecondStory(t, server.URL), "E")

	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter}) // whole story → members
	m = run(t, m, cmd)
	if view := stripANSI(m.View()); !strings.Contains(view, "Esfuerzo · Integrante") {
		t.Fatalf("Enter on the task did not show the members: %q", view)
	}
	m = press(m, tea.KeyEsc)
	if m.Screen != "section" || !strings.Contains(stripANSI(m.View()), "Esfuerzo · Tareas") {
		t.Fatalf("Esc from members left %q: %q", m.Screen, stripANSI(m.View()))
	}
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != "backlog" {
		t.Fatalf("Esc from the preselected tasks left %q, want backlog", m.Screen)
	}
}

func TestShortcutsDoNothingOnEmptyBacklog(t *testing.T) {
	server, rec := shortcutServer(t, nil)
	m := openBacklogWithProject(t, server.URL, "p1")
	before := len(rec.all())
	for _, key := range []string{"P", "E"} {
		next, cmd := apply(m, keyMsg(key))
		if next.Screen != "backlog" || cmd != nil {
			t.Errorf("%s on an empty backlog: Screen = %q, cmd = %v", key, next.Screen, cmd != nil)
		}
	}
	if got := len(rec.all()); got != before {
		t.Errorf("empty backlog shortcuts issued %d requests", got-before)
	}
	m, _ = apply(m, keyMsg("?"))
	help := stripANSI(m.View())
	for _, unwanted := range []string{"Estimar la historia", "Registrar esfuerzo de la historia"} {
		if strings.Contains(help, unwanted) {
			t.Errorf("empty backlog help advertises %q: %q", unwanted, help)
		}
	}
}

func TestLowercasePAndEStillStartASearch(t *testing.T) {
	server, _ := shortcutServer(t, sectionStories())
	for _, key := range []string{"p", "e"} {
		m := openBacklogWithProject(t, server.URL, "p1")
		m, _ = apply(m, keyMsg(key))
		if m.Screen != "backlog" {
			t.Fatalf("%s left the backlog for %q", key, m.Screen)
		}
		if view := stripANSI(m.View()); !strings.Contains(view, "Buscar: "+key+"_") {
			t.Errorf("%s did not start a search: %q", key, view)
		}
		m = press(m, tea.KeyEsc) // clears the search, does not leave
		if m.Screen != "backlog" {
			t.Errorf("Esc after %s left %q: the key was not typed into a search", key, m.Screen)
		}
	}
}

func TestContextualShortcutFlowOnlyReads(t *testing.T) {
	server, rec := shortcutServer(t, sectionStories())
	m := shortcut(t, backlogOnSecondStory(t, server.URL), "P")
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEsc})
	m = run(t, m, cmd)
	m = press(m, tea.KeyDown)
	m = shortcut(t, m, "E")
	m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(t, m, cmd)
	_ = press(m, tea.KeyEsc)
	for _, request := range rec.all() {
		if request.Method != http.MethodGet {
			t.Errorf("shortcut flow issued %s %s", request.Method, request.Path)
		}
	}
}

func TestBacklogHelpListsPAndE(t *testing.T) {
	server, _ := shortcutServer(t, sectionStories())
	m := openBacklogWithProject(t, server.URL, "p1")
	m, _ = apply(m, keyMsg("?"))
	help := stripANSI(m.View())
	actions := help[strings.Index(help, "Acciones"):]
	if general := strings.Index(actions, "General"); general >= 0 {
		actions = actions[:general]
	}
	for _, want := range []string{"P", "Estimar la historia resaltada", "E", "Registrar esfuerzo de la historia resaltada"} {
		if !strings.Contains(actions, want) {
			t.Errorf("backlog help Acciones lacks %q: %q", want, actions)
		}
	}
}
