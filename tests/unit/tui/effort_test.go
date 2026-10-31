package tui_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

const (
	effortTasksBody   = `{"project_id":"p1","story_id":"s1","tasks":[{"id":"t1","project_id":"p1","sprint_id":"sp1","story_id":"s1","title":"Diseñar","estimated_hours":4.5},{"id":"t2","project_id":"p1","sprint_id":"sp1","story_id":"s1","title":"Probar"}]}`
	effortMembersBody = `{"project_id":"p1","members":[{"id":"m1","project_id":"p1","full_name":"Ada Lovelace","email":"ada@example.com"},{"id":"m2","project_id":"p1","full_name":"Grace Hopper","email":null}]}`
	effortRecordBody  = `{"id":"e1","project_id":"p1","story_id":"s1","task_id":"t1","member_id":"m1","work_date":"2026-10-05","description":"Implementar login","hours":3.5}`
	effortMetricsBody = `{"project_id":"p1","actual_hours":30.5,"deviation":-4}`
)

// effortFlowOptions tunes the answers of effortFlowServer. The zero value serves
// a happy path: two tasks in s1, two members and a 201 registration.
type effortFlowOptions struct {
	noMembers       bool
	tasksFailures   int // leading GETs of the s1 tasks answered with 500
	membersFailures int // leading GETs of the members answered with 500
	postStatus      int // status of the effort POST (default 201)
	postBody        string
	metricsStatus   int      // status of GET /metrics (default 200)
	metricsBodies   []string // served in order; the last one repeats
}

// effortFlowServer records every request and declares each route the effort flow
// can reach. Anything undeclared answers 404, unlike sprintServer, whose
// {"ok":true} fallback would parse as an all-nil metrics payload.
func effortFlowServer(t *testing.T, opts effortFlowOptions) (*httptest.Server, *recorder) {
	t.Helper()
	rec := &recorder{}
	var mu sync.Mutex
	var tasksCalls, membersCalls, metricsCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		rec.mu.Lock()
		rec.requests = append(rec.requests, recordedRequest{Method: r.Method, Path: r.URL.EscapedPath(), Body: raw})
		rec.mu.Unlock()
		mu.Lock()
		defer mu.Unlock()

		status, body := http.StatusOK, ""
		switch r.Method + " " + r.URL.Path {
		case "GET /projects":
			body = projectsBody(t, sampleProjects())
		case "GET /projects/p1/stories":
			body = storiesBodyForSection(sectionStories())
		case "GET /projects/p2/stories":
			body = `{"project_id":"p2","stories":[{"id":"s9","project_id":"p2","title":"Historia de Beta","priority":"alta","status":"pendiente"}]}`
		case "GET /projects/p1/stories/s1/tasks":
			tasksCalls++
			body = effortTasksBody
			if tasksCalls <= opts.tasksFailures {
				status, body = http.StatusInternalServerError, `{"error":"internal_error","message":"tasks down"}`
			}
		case "GET /projects/p1/stories/s2/tasks":
			body = `{"project_id":"p1","story_id":"s2","tasks":[]}`
		case "GET /projects/p1/members":
			membersCalls++
			body = effortMembersBody
			if opts.noMembers {
				body = `{"project_id":"p1","members":[]}`
			}
			if membersCalls <= opts.membersFailures {
				status, body = http.StatusInternalServerError, `{"error":"internal_error","message":"members down"}`
			}
		case "POST /projects/p1/stories/s1/effort-records", "POST /projects/p1/stories/s2/effort-records":
			status, body = http.StatusCreated, effortRecordBody
			if opts.postStatus != 0 {
				status, body = opts.postStatus, opts.postBody
			}
		case "GET /projects/p1/metrics":
			status, body = http.StatusOK, effortMetricsBody
			if opts.metricsStatus != 0 {
				status = opts.metricsStatus
			}
			if len(opts.metricsBodies) > 0 {
				body = opts.metricsBodies[min(metricsCalls, len(opts.metricsBodies)-1)]
			}
			metricsCalls++
		default:
			status, body = http.StatusNotFound, `{"error":"not_found"}`
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, rec
}

// openEffortTasks opens Esfuerzo, chooses the story at storyIndex and applies
// its task list.
func openEffortTasks(t *testing.T, url string, storyIndex int) tui.Model {
	t.Helper()
	m := openSectionWithProject(t, url, "p1", menuEffort)
	for i := 0; i < storyIndex; i++ {
		m = press(m, tea.KeyDown)
	}
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	return run(t, m, cmd)
}

// openEffortMembers chooses the row at taskIndex (0 is the whole-story row) and
// applies the member list.
func openEffortMembers(t *testing.T, url string, storyIndex, taskIndex int) tui.Model {
	t.Helper()
	m := openEffortTasks(t, url, storyIndex)
	for i := 0; i < taskIndex; i++ {
		m = press(m, tea.KeyDown)
	}
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	return run(t, m, cmd)
}

// focusedLine returns the line carrying the selection marker.
func focusedLine(view string) string {
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "▸") {
			return line
		}
	}
	return ""
}

// requestCount counts the recorded requests of one method and escaped path.
func requestCount(rec *recorder, method, path string) int {
	count := 0
	for _, request := range rec.all() {
		if request.Method == method && request.Path == path {
			count++
		}
	}
	return count
}

func TestEffortTaskSelectorListsWholeStoryRowFirst(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortTasks(t, server.URL, 0)

	view := stripANSI(m.View())
	whole, design, test := strings.Index(view, "Toda la historia (sin tarea)"), strings.Index(view, "Diseñar"), strings.Index(view, "Probar")
	if whole < 0 || design < 0 || test < 0 || !(whole < design && design < test) {
		t.Fatalf("rows must be whole story, Diseñar, Probar in that order (offsets %d, %d, %d): %q", whole, design, test, view)
	}
	if line := focusedLine(view); !strings.Contains(line, "Toda la historia") {
		t.Fatalf("the whole-story row should be highlighted first: %q", line)
	}
	if !strings.Contains(view, "Tarea de Primera historia o la historia completa.") {
		t.Fatalf("task step lacks the Esfuerzo help text: %q", view)
	}
}

func TestEffortTaskSelectorOffersWholeStoryWhenStoryHasNoTasks(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openEffortTasks(t, server.URL, 1) // s2 has no tasks

	view := stripANSI(m.View())
	if !strings.Contains(view, "Toda la historia (sin tarea)") || !strings.Contains(view, "La historia no tiene tareas.") {
		t.Fatalf("an empty story should still offer the whole-story row plus an informative line: %q", view)
	}
	if strings.Contains(view, "◇") {
		t.Fatalf("the empty state must not replace the whole-story row: %q", view)
	}
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter on the whole-story row should load the members")
	}
	m = run(t, m, cmd)
	if requestCount(rec, http.MethodGet, "/projects/p1/members") != 1 {
		t.Fatalf("requests = %+v, want one GET members", rec.all())
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "Ada Lovelace") {
		t.Fatalf("members were not shown: %q", view)
	}
}

func TestEffortTaskSelectorRefreshKeepsSyntheticRow(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openEffortTasks(t, server.URL, 0)

	m, cmd := apply(m, keyMsg("r"))
	if cmd == nil {
		t.Fatal("r should reload the tasks")
	}
	m = run(t, m, cmd)
	if got := requestCount(rec, http.MethodGet, "/projects/p1/stories/s1/tasks"); got != 2 {
		t.Fatalf("GET tasks = %d, want 2 after the refresh", got)
	}
	view := stripANSI(m.View())
	if whole, design := strings.Index(view, "Toda la historia (sin tarea)"), strings.Index(view, "Diseñar"); whole < 0 || design < 0 || whole > design {
		t.Fatalf("whole-story row should stay first after the refresh: %q", view)
	}
}

func TestEffortTaskSelectorOmitsSyntheticRowWhenTasksFail(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{tasksFailures: 1})
	m := openEffortTasks(t, server.URL, 0)

	view := stripANSI(m.View())
	if m.Screen != "section" || !strings.Contains(view, "500") || strings.Contains(view, "Toda la historia") {
		t.Fatalf("a failed task query should show the error without the whole-story row (screen %q): %q", m.Screen, view)
	}
	if _, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		t.Fatal("Enter must not open the members after a failed task query")
	}
	if requestCount(rec, http.MethodGet, "/projects/p1/members") != 0 {
		t.Fatalf("requests = %+v, want no GET members", rec.all())
	}

	m, cmd := apply(m, keyMsg("r"))
	m = run(t, m, cmd)
	if view := stripANSI(m.View()); !strings.Contains(view, "Toda la historia (sin tarea)") || !strings.Contains(view, "Diseñar") {
		t.Fatalf("the row should appear once the retry succeeds: %q", view)
	}
}

func TestEffortMemberStepUsesSingleSelectionView(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openEffortMembers(t, server.URL, 0, 1)

	if requestCount(rec, http.MethodGet, "/projects/p1/members") != 1 {
		t.Fatalf("requests = %+v, want one GET members", rec.all())
	}
	view := stripANSI(m.View())
	for _, want := range []string{"Esfuerzo · Integrante", "Ada Lovelace", "Grace Hopper"} {
		if !strings.Contains(view, want) {
			t.Fatalf("member selector lacks %q: %q", want, view)
		}
	}
	assertSingleSelection := func(view string) {
		t.Helper()
		lower := strings.ToLower(view)
		for _, banned := range []string{"[ ]", "space", "participantes"} {
			if strings.Contains(lower, banned) {
				t.Fatalf("a single-selection view must not mention %q: %q", banned, view)
			}
		}
	}
	assertSingleSelection(view)

	afterSpace, _ := apply(m, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	if got := stripANSI(afterSpace.View()); got != view {
		t.Fatalf("Space must not mark any row:\nbefore %q\nafter  %q", view, got)
	}
	help, _ := apply(m, keyMsg("?"))
	assertSingleSelection(stripANSI(help.View()))
}

func TestEffortMemberStepEmptyStateDoesNotPointToMenu(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{noMembers: true})
	m := openEffortMembers(t, server.URL, 0, 1)

	view := stripANSI(m.View())
	if !strings.Contains(view, "El proyecto no tiene integrantes registrados.") {
		t.Fatalf("empty member list lacks its own message: %q", view)
	}
	if strings.Contains(strings.ToLower(view), "menú") {
		t.Fatalf("the empty state must not point to a menu option: %q", view)
	}

	m, cmd := apply(m, keyMsg("r"))
	m = run(t, m, cmd)
	if got := requestCount(rec, http.MethodGet, "/projects/p1/members"); got != 2 {
		t.Fatalf("GET members = %d, want 2 after r", got)
	}

	m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatal("Esc must not emit a command")
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "Diseñar") || !strings.Contains(view, "Tarea de Primera historia") {
		t.Fatalf("Esc should return to the task selector: %q", view)
	}
}

func TestEffortMemberStepShowsAPIErrorAndKeepsFormClosed(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{membersFailures: 1})
	m := openEffortMembers(t, server.URL, 0, 1)

	view := stripANSI(m.View())
	if m.Screen != "section" || !strings.Contains(view, "500") || !strings.Contains(view, "Integrante") {
		t.Fatalf("a failed member query should show the error inside the section (screen %q): %q", m.Screen, view)
	}

	t.Run("r retries", func(t *testing.T) {
		retried, cmd := apply(m, keyMsg("r"))
		retried = run(t, retried, cmd)
		if view := stripANSI(retried.View()); !strings.Contains(view, "Ada Lovelace") || retried.Screen != "section" {
			t.Fatalf("the retry should list the members (screen %q): %q", retried.Screen, view)
		}
	})
	t.Run("Esc goes back", func(t *testing.T) {
		back, _ := apply(m, tea.KeyMsg{Type: tea.KeyEsc})
		if view := stripANSI(back.View()); !strings.Contains(view, "Diseñar") {
			t.Fatalf("Esc should return to the tasks: %q", view)
		}
	})
}

func TestEffortEscFromMembersRestoresTasksSnapshotWithoutRequests(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openEffortTasks(t, server.URL, 0)
	m = typeText(m, "dis")
	m = press(m, tea.KeyEnter) // stop typing, keep the filter
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(t, m, cmd) // members of "Diseñar"
	before := len(rec.all())

	m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatal("Esc must not emit a command")
	}
	view := stripANSI(m.View())
	if !strings.Contains(view, "dis") || strings.Contains(view, "Probar") || !strings.Contains(focusedLine(view), "Diseñar") {
		t.Fatalf("filter and cursor of the task selector should be restored: %q", view)
	}

	m = press(m, tea.KeyEsc) // clears the filter
	view = stripANSI(m.View())
	if !strings.Contains(view, "Toda la historia (sin tarea)") || !strings.Contains(view, "Probar") {
		t.Fatalf("the next Esc should clear the filter first: %q", view)
	}
	m = press(m, tea.KeyEsc)
	if view := stripANSI(m.View()); !strings.Contains(view, "Segunda historia") {
		t.Fatalf("the following Esc should reach the story selector: %q", view)
	}
	if after := len(rec.all()); after != before {
		t.Fatalf("walking back emitted %d new requests, want none", after-before)
	}
}
