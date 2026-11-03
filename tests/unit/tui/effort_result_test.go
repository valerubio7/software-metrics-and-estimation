package tui_test

import (
	"net/http"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

// registerEffort fills and confirms the open effort form and applies the 201,
// returning the model back on the Esfuerzo root together with its pending
// story reload, which has not been executed yet.
func registerEffort(t *testing.T, m tui.Model) (tui.Model, tea.Cmd) {
	t.Helper()
	m = fillEffortForm(m, [3]string{"2026", "10", "05"}, "  Implementar login  ", "3.5")
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("a valid form should emit the POST command")
	}
	return apply(m, saveMessage(t, cmd))
}

// registered registers an effort and applies the story reload, so the model
// rests on the loaded story selector of the Esfuerzo tab.
func registered(t *testing.T, url string) tui.Model {
	t.Helper()
	m, cmd := registerEffort(t, openEffortFormFor(t, url, 0, 1, 0))
	return run(t, m, cmd)
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
			if m.Screen != "section" {
				t.Fatalf("Screen = %q, want the Esfuerzo root", m.Screen)
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
	run(t, m, cmd)
	if got := requestCount(rec, http.MethodGet, "/projects/p1/metrics"); got != 0 {
		t.Fatalf("the registration must not query the metrics, GET /metrics = %d", got)
	}
}

func TestEffortSecondRegistrationStartsWithInitialForm(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := registered(t, server.URL).WithClock(fixedClock)
	m = stepRunning(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // tasks of Primera historia
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

func TestEffortProjectSwitchAfterRegistrationReopensStoriesAndDiscardsFlow(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := registered(t, server.URL).WithClock(fixedClock)

	m = switchEffortProject(t, m, "p2")

	view := stripANSI(m.View())
	if m.Screen != "section" || !strings.Contains(view, "Historia de Beta") || !strings.Contains(view, "Proyectos › Beta › Esfuerzo") {
		t.Fatalf("Esfuerzo should reopen on the stories of Beta (screen %q): %q", m.Screen, view)
	}
	for _, stale := range []string{"Esfuerzo registrado", "Implementar login", "Ada Lovelace", "Diseñar", "Primera historia"} {
		if strings.Contains(view, stale) {
			t.Errorf("nothing from the previous project should remain, found %q: %q", stale, view)
		}
	}

	// A new flow in the new project starts from a clean form.
	m = stepRunning(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // tasks of the story
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
}

// switchEffortProject walks back with Esc to the project browser, picks pid and
// reopens the Esfuerzo tab, applying its story load.
func switchEffortProject(t *testing.T, m tui.Model, pid string) tui.Model {
	t.Helper()
	var load tea.Cmd
	for i := 0; i < 6 && m.Screen != "project-picker"; i++ {
		m, load = apply(m, tea.KeyMsg{Type: tea.KeyEsc})
	}
	if m.Screen != "project-picker" {
		t.Fatalf("Esc did not reach the project browser (screen %q)", m.Screen)
	}
	m, _ = confirmProjectInPicker(t, m, load, pid)
	m, cmd := apply(m, keyMsg(tabEffort))
	if cmd == nil {
		t.Fatal("opening Esfuerzo should load the stories of the new project")
	}
	return run(t, m, cmd)
}

func TestEffortProjectSwitchFromSelectorDiscardsFlow(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortMembers(t, server.URL, 0, 1)

	m = switchEffortProject(t, m, "p2")

	view := stripANSI(m.View())
	if m.Screen != "section" || !strings.Contains(view, "Historia de Beta") {
		t.Fatalf("Esfuerzo should reopen on the stories of Beta (screen %q): %q", m.Screen, view)
	}
	for _, stale := range []string{"Ada Lovelace", "Diseñar", "Primera historia"} {
		if strings.Contains(view, stale) {
			t.Errorf("nothing from the previous project should remain, found %q: %q", stale, view)
		}
	}
}
