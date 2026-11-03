package tui_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

// toastServer answers GET /projects with projects and every write with
// status/body, recording each request.
func toastServer(t *testing.T, projects []tui.ProjectSummary, status int, body string) (*httptest.Server, *recorder) {
	t.Helper()
	rec := &recorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		rec.mu.Lock()
		rec.requests = append(rec.requests, recordedRequest{Method: r.Method, Path: r.URL.EscapedPath(), Body: raw})
		rec.mu.Unlock()
		if r.Method == http.MethodGet && r.URL.Path == "/projects" {
			_, _ = w.Write([]byte(projectsBody(t, projects)))
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, rec
}

// assertToast checks the view shows the success notice and that the next key
// removes it.
func assertToast(t *testing.T, m tui.Model, want string) {
	t.Helper()
	m, _ = apply(m, tea.WindowSizeMsg{Width: 200, Height: 40})
	view := stripANSI(m.View())
	if !strings.Contains(view, "✓ "+want) {
		t.Fatalf("view lacks the toast %q:\n%s", "✓ "+want, view)
	}
	m = press(m, tea.KeyDown)
	if view := stripANSI(m.View()); strings.Contains(view, want) {
		t.Fatalf("the toast %q survived the next key:\n%s", want, view)
	}
}

func TestProjectCreateReturnsToBrowserWithToast(t *testing.T) {
	server, rec := toastServer(t, sampleProjects(), http.StatusCreated, `{"id":"p3","name":"Gamma","status":"planned"}`)
	m := startTUI(t, server.URL)
	m, cmd := submitProject(t, m, "Gamma")
	m, reload := apply(m, cmd())
	if m.Screen != "project-picker" || reload == nil {
		t.Fatalf("Screen = %q reload nil %v, want the browser reloading", m.Screen, reload == nil)
	}
	before := countMethod(rec, http.MethodGet)
	m = run(t, m, reload)
	if countMethod(rec, http.MethodGet) != before+1 || lastRequest(t, rec).Path != "/projects" {
		t.Fatalf("reload did not request GET /projects: %+v", rec.all())
	}
	assertToast(t, m, "Proyecto creado: Gamma")
}

func TestProjectEditReturnsToBrowserWithToast(t *testing.T) {
	server, _ := toastServer(t, sampleProjects(), http.StatusOK, `{"id":"p1","name":"Alpha 2","status":"active"}`)
	m := toEditProject(t, tui.NewModel(server.URL), sampleProjects()[0])
	m = press(press(m, tea.KeyTab), tea.KeyTab)
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m, reload := apply(m, cmd())
	if m.Screen != "project-picker" || reload == nil {
		t.Fatalf("Screen = %q reload nil %v, want the browser reloading", m.Screen, reload == nil)
	}
	assertToast(t, run(t, m, reload), "Proyecto modificado: Alpha 2")
}

func TestProjectSaveReloadNeverAutoOpensTheOnlyProject(t *testing.T) {
	one := []tui.ProjectSummary{sampleProjects()[0]}
	server, _ := toastServer(t, one, http.StatusCreated, `{"id":"p1","name":"Alpha","status":"active"}`)
	m := tui.NewModel(server.URL) // the startup load has not answered yet
	m, cmd := submitProject(t, m, "Alpha")
	m, reload := apply(m, cmd())
	m = run(t, m, reload)
	if m.Screen != "project-picker" {
		t.Fatalf("Screen = %q, the reload after a save must stay on the browser", m.Screen)
	}
}

func TestProjectRejectionKeepsTheForm(t *testing.T) {
	server, _ := toastServer(t, sampleProjects(), http.StatusUnprocessableEntity, `{"error":"validation_failed","message":"name already taken"}`)
	m := startTUI(t, server.URL)
	m, cmd := submitProject(t, m, "Gamma")
	m, _ = apply(m, cmd())
	m, _ = apply(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	view := stripANSI(m.View())
	if m.Screen != "create-project" || fieldValue(t, view, "Nombre") != "Gamma" || !strings.Contains(view, "name already taken") {
		t.Fatalf("Screen = %q, want the form with Gamma and the API reason:\n%s", m.Screen, view)
	}
	if strings.Contains(view, "Resultado") {
		t.Fatalf("a rejection must not open a result screen:\n%s", view)
	}
}

func TestProjectFormIgnoresRepeatsWhileSaving(t *testing.T) {
	server, _ := toastServer(t, sampleProjects(), http.StatusCreated, `{"id":"p3","name":"Gamma"}`)
	m, _ := submitProject(t, startTUI(t, server.URL), "Gamma")
	if !strings.Contains(stripANSI(m.View()), "Guardando…") {
		t.Fatalf("the form should show the saving indicator:\n%s", stripANSI(m.View()))
	}
	if _, again := apply(m, tea.KeyMsg{Type: tea.KeyEnter}); again != nil {
		t.Fatal("Enter while saving must not send the project again")
	}
}

func TestStoryCreateReturnsToBacklogWithToast(t *testing.T) {
	server, rec := fakeBacklogAPI(t, apiStubs{})
	m := openBacklogWithProject(t, server.URL, "p1")
	m, cmd := submitStory(m, "Historia nueva", "d", "alta", "c")
	m, reload := apply(m, cmd())
	if m.Screen != "backlog" || reload == nil {
		t.Fatalf("Screen = %q reload nil %v, want the backlog reloading", m.Screen, reload == nil)
	}
	before := countMethod(rec, http.MethodGet)
	m = run(t, m, reload)
	if countMethod(rec, http.MethodGet) != before+1 || lastRequest(t, rec).Path != "/projects/p1/stories" {
		t.Fatalf("reload did not list the stories: %+v", rec.all())
	}
	assertToast(t, m, "Historia creada: Historia nueva")
}

func TestStoryRejectionKeepsTheForm(t *testing.T) {
	server, _ := fakeBacklogAPI(t, apiStubs{create: stub{http.StatusUnprocessableEntity, `{"error":"validation_failed","message":"title too long"}`}})
	m := openBacklogWithProject(t, server.URL, "p1")
	m, cmd := submitStory(m, "Historia nueva", "d", "alta", "c")
	m, _ = apply(m, cmd())
	view := stripANSI(m.View())
	if m.Screen != "backlog-form" || !strings.Contains(view, "Historia nueva") || !strings.Contains(view, "title too long") {
		t.Fatalf("Screen = %q, want the form with the title and the reason:\n%s", m.Screen, view)
	}
}

func sprintCreateServer(t *testing.T, status int, body string) (*httptest.Server, *recorder) {
	t.Helper()
	rec := &recorder{}
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		rec.mu.Lock()
		rec.requests = append(rec.requests, recordedRequest{Method: r.Method, Path: r.URL.EscapedPath()})
		rec.mu.Unlock()
		switch r.Method + " " + r.URL.Path {
		case "GET /projects/p1/sprints/open":
			_, _ = w.Write([]byte(`{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Primero","is_closed":false}]}`))
			return true
		case "POST /projects/p1/sprints":
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
			return true
		}
		return false
	})
	return server, rec
}

func TestSprintCreateReturnsToSprintsWithToast(t *testing.T) {
	server, rec := sprintCreateServer(t, http.StatusCreated, `{"id":"sp9","project_id":"p1","sprint_goal":"Meta","is_closed":false}`)
	m := openSprintsWithProject(t, server.URL, "p1")
	m = typeText(m, "n")
	m = typeText(m, "Meta")
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(stripANSI(m.View()), "Guardando…") {
		t.Fatalf("the Sprint form should show the saving indicator:\n%s", stripANSI(m.View()))
	}
	if _, again := apply(m, tea.KeyMsg{Type: tea.KeyEnter}); again != nil {
		t.Fatal("Enter while saving must not create the Sprint again")
	}
	m, reload := apply(m, cmd())
	if m.Screen != "sprints" || reload == nil {
		t.Fatalf("Screen = %q reload nil %v, want the Sprints list reloading", m.Screen, reload == nil)
	}
	m = run(t, m, reload)
	if last := rec.all()[len(rec.all())-1]; last.Method != http.MethodGet || last.Path != "/projects/p1/sprints/open" {
		t.Fatalf("reload did not list the open Sprints: %+v", rec.all())
	}
	assertToast(t, m, "Sprint creado: Meta")
}

func TestSprintCreateRejectionKeepsTheForm(t *testing.T) {
	server, _ := sprintCreateServer(t, http.StatusUnprocessableEntity, `{"error":"validation_failed","message":"goal is required"}`)
	m := openSprintsWithProject(t, server.URL, "p1")
	m = typeText(m, "n")
	m = typeText(m, "Meta")
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(m, cmd())
	view := stripANSI(m.View())
	if m.Screen != "sprint-goal" || !strings.Contains(view, "Meta") || !strings.Contains(view, "goal is required") {
		t.Fatalf("Screen = %q, want the Sprint form with its goal and the reason:\n%s", m.Screen, view)
	}
}

func TestEffortRegistrationReturnsToEffortRootWithToast(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)
	m = fillEffortForm(m, [3]string{"2026", "10", "05"}, "Implementar login", "3.5")
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m, reload := apply(m, cmd())
	if m.Screen != "section" || reload == nil {
		t.Fatalf("Screen = %q reload nil %v, want the Esfuerzo root reloading", m.Screen, reload == nil)
	}
	stories := requestCount(rec, http.MethodGet, "/projects/p1/stories")
	m = run(t, m, reload)
	if requestCount(rec, http.MethodGet, "/projects/p1/stories") != stories+1 {
		t.Fatalf("reload did not list the stories: %+v", rec.all())
	}
	if requestCount(rec, http.MethodGet, "/projects/p1/metrics") != 0 {
		t.Fatalf("the save must not query the project metrics: %+v", rec.all())
	}
	view := stripANSI(m.View())
	if !strings.Contains(view, "Primera historia") || !strings.Contains(view, "Segunda historia") {
		t.Fatalf("the story selector should be back:\n%s", view)
	}
	assertToast(t, m, "Esfuerzo registrado: 3.5 h el 2026-10-05 · Primera historia · Diseñar · Ada Lovelace")
}
