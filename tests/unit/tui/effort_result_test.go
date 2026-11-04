package tui_test

import (
	"net/http"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

// registerEffort fills and confirms the open effort form and applies the 201,
// returning the model back on the story detail the flow started from together
// with the command the success issued (none: the detail is not reloaded).
func registerEffort(t *testing.T, m tui.Model) (tui.Model, tea.Cmd) {
	t.Helper()
	m = fillEffortForm(m, [3]string{"2026", "10", "05"}, "  Implementar login  ", "3.5")
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("a valid form should emit the POST command")
	}
	return apply(m, saveMessage(t, cmd))
}

// wideView renders m on a wide terminal so a toast fits on one line.
func wideView(m tui.Model) string {
	m, _ = apply(m, tea.WindowSizeMsg{Width: 220, Height: 40})
	return stripANSI(m.View())
}

func TestEffortToastShowsRecordFromAPIAndSelection(t *testing.T) {
	// The API answers with values that differ from what the form sent, so the
	// toast can only get them from the response.
	created := `{"id":"e9","project_id":"p1","story_id":"s1","task_id":"t1","member_id":"m1","work_date":"2026-10-06","description":"Implementar el login","hours":3.75}`
	tests := []struct {
		name      string
		taskIndex int
		wantTask  string
	}{
		{"single task", 1, "Diseñar"},
		{"whole story", 0, "toda la historia"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := effortFlowServer(t, effortFlowOptions{postStatus: http.StatusCreated, postBody: created})
			m, _ := registerEffort(t, openEffortFormFor(t, server.URL, 0, tt.taskIndex, 0))

			view := wideView(m)
			if m.Screen != "story-detail" {
				t.Fatalf("Screen = %q, want the story detail", m.Screen)
			}
			want := "✓ Esfuerzo registrado: 3.75 h el 2026-10-06 · Primera historia · " + tt.wantTask + " · Ada Lovelace"
			if !strings.Contains(view, want) {
				t.Errorf("the toast should read %q: %q", want, view)
			}
			for _, sent := range []string{"2026-10-05", "3.5 h"} {
				if strings.Contains(view, sent) {
					t.Errorf("the record must come from the API, not from what the form sent (%q): %q", sent, view)
				}
			}
			if strings.Contains(view, "✗") {
				t.Errorf("a successful registration must not show an error: %q", view)
			}
		})
	}
}

func TestEffortToastFallsBackToFormValuesOnUnreadableBody(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{postStatus: http.StatusCreated, postBody: "no-json"})
	m, cmd := registerEffort(t, openEffortFormFor(t, server.URL, 0, 1, 0))

	view := wideView(m)
	if !strings.Contains(view, "✓ Esfuerzo registrado: 3.5 h el 2026-10-05 · Primera historia · Diseñar · Ada Lovelace") {
		t.Errorf("an unreadable 201 body should still confirm with the sent values: %q", view)
	}
	for _, banned := range []string{"✗", "No se pudo confirmar el registro"} {
		if strings.Contains(view, banned) {
			t.Errorf("an unreadable 201 body is not a registration error, found %q: %q", banned, view)
		}
	}
	if cmd != nil {
		t.Fatal("the registration must not issue a follow-up command: the detail is not reloaded")
	}
	if got := requestCount(rec, http.MethodGet, "/projects/p1/metrics"); got != 0 {
		t.Fatalf("the registration must not query the metrics, GET /metrics = %d", got)
	}
}

// registeredFromDetail registers an effort launched from the detail of the first
// story (task Diseñar, member Ada Lovelace) and applies the 201.
func registeredFromDetail(t *testing.T, url string) (tui.Model, tea.Cmd) {
	t.Helper()
	return registerEffort(t, openEffortFormFor(t, url, 0, 1, 0))
}

func TestEffortSuccessFromDetailReturnsToStoryDetailWithToast(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	before := len(rec.all())
	m := openEffortFormFor(t, server.URL, 0, 1, 0)
	flowRequests := len(rec.all())

	m, cmd := registerEffort(t, m)

	if m.Screen != "story-detail" || cmd != nil {
		t.Fatalf("Screen = %q, cmd nil = %v, want the detail without a reload", m.Screen, cmd == nil)
	}
	view := wideView(m)
	for _, want := range []string{"✓ Esfuerzo registrado: 3.5 h el 2026-10-05 · Primera historia · Diseñar · Ada Lovelace", "Primera historia"} {
		if !strings.Contains(view, want) {
			t.Errorf("view lacks %q:\n%s", want, view)
		}
	}
	// Only the POST left after the form opened: no story list, no metrics.
	if got := requestLines(rec, flowRequests); len(got) != 1 || got[0] != "POST /projects/p1/stories/s1/effort-records" {
		t.Errorf("requests after the form = %v (from %d), want only the POST", got, before)
	}
}

func TestEffortSecondRegistrationFromDetailStartsWithInitialForm(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m, _ := registeredFromDetail(t, server.URL)
	m = stepRunning(t, m.WithClock(fixedClock), keyMsg("t")) // tasks of Primera historia again
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeyDown) // Probar
	m = stepRunning(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = press(m, tea.KeyDown) // Grace Hopper
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})

	view := stripANSI(m.View())
	if m.Screen != "effort-form" || !strings.Contains(view, "Tarea: Probar") || !strings.Contains(view, "Integrante: Grace Hopper") {
		t.Fatalf("a second registration should open its own form (screen %q): %q", m.Screen, view)
	}
	if date := effortDate(t, view); date != [3]string{"2026", "11", "01"} {
		t.Errorf("the date should be today's, got %v", date)
	}
	for _, label := range []string{"Actividad", "Horas"} {
		if value := fieldValue(t, view, label); value != "" {
			t.Errorf("%s should start empty, got %q", label, value)
		}
	}
	assertFocus(t, m, "Fecha")
}

func TestEffortProjectSwitchAfterRegistrationOpensBacklogAndDiscardsFlow(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m, _ := registeredFromDetail(t, server.URL)

	m = switchProjectToBacklog(t, m, "p2")

	view := stripANSI(m.View())
	if m.Screen != "backlog" || !strings.Contains(view, "Historia de Beta") || !strings.Contains(view, "Proyectos › Beta › Backlog") {
		t.Fatalf("the Backlog should open on the stories of Beta (screen %q): %q", m.Screen, view)
	}
	for _, stale := range []string{"Esfuerzo registrado", "Implementar login", "Ada Lovelace", "Diseñar", "Primera historia"} {
		if strings.Contains(view, stale) {
			t.Errorf("nothing from the previous project should remain, found %q: %q", stale, view)
		}
	}

	// A new flow in the new project starts from a clean form.
	m = press(m, tea.KeyEnter)                            // detail of the story of Beta
	m = stepRunning(t, m, keyMsg("t"))                    // tasks of the story
	m = stepRunning(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // whole story -> members
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	form := stripANSI(m.View())
	if m.Screen != "effort-form" || !strings.Contains(form, "Historia: Historia de Beta") {
		t.Fatalf("a new flow should open a form for the new project (screen %q): %q", m.Screen, form)
	}
	for _, label := range []string{"Actividad", "Horas"} {
		if value := fieldValue(t, form, label); value != "" {
			t.Errorf("%s should start empty, got %q", label, value)
		}
	}

	// Cancelling goes back to the detail of the story of Beta, never to the one of Alpha.
	for i := 0; i < 3; i++ {
		m = press(m, tea.KeyEsc)
	}
	if got := fieldValue(t, stripANSI(m.View()), "Título"); m.Screen != "story-detail" || got != "Historia de Beta" {
		t.Errorf("cancelling should return to the detail of Beta (screen %q, Título %q)", m.Screen, got)
	}
}

// switchProjectToBacklog walks back with Esc to the project browser, picks pid and
// opens its Backlog, applying the story load.
func switchProjectToBacklog(t *testing.T, m tui.Model, pid string) tui.Model {
	t.Helper()
	var load tea.Cmd
	for i := 0; i < 6 && m.Screen != "project-picker"; i++ {
		m, load = apply(m, tea.KeyMsg{Type: tea.KeyEsc})
	}
	if m.Screen != "project-picker" {
		t.Fatalf("Esc did not reach the project browser (screen %q)", m.Screen)
	}
	m, _ = confirmProjectInPicker(t, m, load, pid)
	m, cmd := apply(m, keyMsg(tabBacklog))
	if cmd == nil {
		t.Fatal("opening the Backlog should load the stories of the new project")
	}
	return run(t, m, cmd)
}

func TestEffortProjectSwitchFromSelectorDiscardsFlow(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortMembers(t, server.URL, 0, 1)

	m = switchProjectToBacklog(t, m, "p2")

	view := stripANSI(m.View())
	if m.Screen != "backlog" || !strings.Contains(view, "Historia de Beta") {
		t.Fatalf("the Backlog should open on the stories of Beta (screen %q): %q", m.Screen, view)
	}
	for _, stale := range []string{"Ada Lovelace", "Diseñar", "Primera historia"} {
		if strings.Contains(view, stale) {
			t.Errorf("nothing from the previous project should remain, found %q: %q", stale, view)
		}
	}
}
