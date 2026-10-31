package tui_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

// registerEffort fills and confirms the open effort form, applies the 201 and
// returns the model on the result screen together with the pending metrics
// command, which has not been executed yet.
func registerEffort(t *testing.T, m tui.Model) (tui.Model, tea.Cmd) {
	t.Helper()
	m = fillEffortForm(m, [3]string{"2026", "10", "05"}, "  Implementar login  ", "3.5")
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("a valid form should emit the POST command")
	}
	return apply(m, saveMessage(t, cmd))
}

// resultWithMetrics registers an effort and also applies the metrics answer, so
// the model rests on a finished result screen.
func resultWithMetrics(t *testing.T, url string) tui.Model {
	t.Helper()
	m, cmd := registerEffort(t, openEffortFormFor(t, url, 0, 1, 0))
	return run(t, m, cmd)
}

// flatView joins the panel lines of a view into one line, dropping the borders,
// so a sentence the panel wrapped can still be matched as a whole.
func flatView(view string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(view, "│", " ")), " ")
}

// metricsRequests counts the GET /projects/p1/metrics requests recorded so far.
func metricsRequests(rec *recorder) int {
	return requestCount(rec, http.MethodGet, "/projects/p1/metrics")
}

func TestEffortResultShowsRecordFromAPIAndSelection(t *testing.T) {
	// The API answers with values that differ from what the form sent, so the
	// screen can only get them from the response.
	created := `{"id":"e9","project_id":"p1","story_id":"s1","task_id":"t1","member_id":"m1","work_date":"2026-10-06","description":"Implementar el login","hours":3.75}`
	tests := []struct {
		name      string
		taskIndex int
		wantTask  string
	}{
		{"single task", 1, "Tarea: Diseñar"},
		{"whole story", 0, "Tarea: Toda la historia"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := effortFlowServer(t, effortFlowOptions{postStatus: http.StatusCreated, postBody: created})
			m, _ := registerEffort(t, openEffortFormFor(t, server.URL, 0, tt.taskIndex, 0))

			view := stripANSI(m.View())
			if m.Screen != "effort-result" {
				t.Fatalf("Screen = %q, want effort-result", m.Screen)
			}
			for _, want := range []string{"Esfuerzo registrado", "2026-10-06", "3.75 h", "Implementar el login", "Historia: Primera historia", tt.wantTask, "Integrante: Ada Lovelace"} {
				if !strings.Contains(view, want) {
					t.Errorf("the result should show %q: %q", want, view)
				}
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

func TestEffortResultShowsLoadingThenMetrics(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m, cmd := registerEffort(t, openEffortFormFor(t, server.URL, 0, 1, 0))

	view := stripANSI(m.View())
	if !strings.Contains(view, "Cargando métricas…") || !strings.Contains(view, "Implementar login") {
		t.Fatalf("while the query is in flight the record and the loading state should show: %q", view)
	}
	if strings.Contains(view, "Horas reales") {
		t.Fatalf("no metric can show before the answer: %q", view)
	}
	if cmd == nil {
		t.Fatal("the 201 should start the metrics query")
	}
	if got := metricsRequests(rec); got != 0 {
		t.Fatalf("the metrics GET must wait for its command, got %d", got)
	}

	m = run(t, m, cmd)
	if got := metricsRequests(rec); got != 1 {
		t.Fatalf("GET /projects/p1/metrics = %d, want 1", got)
	}
	view = stripANSI(m.View())
	for _, want := range []string{"Métricas del proyecto", "Horas reales: 30.5 h", "Desviación: -4"} {
		if !strings.Contains(view, want) {
			t.Errorf("the answered metrics should show %q: %q", want, view)
		}
	}
	if strings.Contains(view, "Cargando métricas") {
		t.Errorf("the loading state must end with the answer: %q", view)
	}
}

func TestEffortResultDoesNotComputeMetrics(t *testing.T) {
	body := `{"project_id":"p1","actual_hours":10,"deviation":4}`
	server, _ := effortFlowServer(t, effortFlowOptions{metricsBodies: []string{body}})
	m := resultWithMetrics(t, server.URL) // 3.5 hours were registered

	view := stripANSI(m.View())
	for _, want := range []string{"Horas reales: 10 h", "Desviación: 4"} {
		if !strings.Contains(view, want) {
			t.Errorf("the metrics should show exactly what the API returned (%q): %q", want, view)
		}
	}
	for _, derived := range []string{"13.5", "6.5", "-6.5", "6.5 h"} {
		if strings.Contains(view, derived) {
			t.Errorf("the TUI must not sum or subtract hours, found %q: %q", derived, view)
		}
	}
}

func TestEffortResultNullMetricsShowSinDatos(t *testing.T) {
	body := `{"project_id":"p1","actual_hours":null,"deviation":null}`
	server, _ := effortFlowServer(t, effortFlowOptions{metricsBodies: []string{body}})
	m := resultWithMetrics(t, server.URL)

	view := stripANSI(m.View())
	for _, want := range []string{"Horas reales: sin datos", "Desviación: sin datos"} {
		if !strings.Contains(view, want) {
			t.Errorf("a null metric should read sin datos (%q): %q", want, view)
		}
	}
}

func TestEffortResultCrumbAndTitle(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := resultWithMetrics(t, server.URL)

	view := stripANSI(m.View())
	for _, want := range []string{"Esfuerzo › Resultado", "Proyecto: Alpha"} {
		if !strings.Contains(view, want) {
			t.Errorf("the breadcrumb should read %q: %q", want, view)
		}
	}
}

func TestEffortResultRefreshesMetricsWithR(t *testing.T) {
	bodies := []string{`{"project_id":"p1","actual_hours":7.5,"deviation":-2.5}`, `{"project_id":"p1","actual_hours":9,"deviation":-1}`}
	server, rec := effortFlowServer(t, effortFlowOptions{metricsBodies: bodies})
	m := resultWithMetrics(t, server.URL)
	if view := stripANSI(m.View()); !strings.Contains(view, "Horas reales: 7.5 h") {
		t.Fatalf("setup: the first answer should show 7.5 h: %q", view)
	}

	m, cmd := apply(m, keyMsg("r"))
	if cmd == nil {
		t.Fatal("r should query the metrics again")
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "Cargando métricas…") {
		t.Fatalf("a refresh should show the loading state: %q", view)
	}
	m = run(t, m, cmd)

	if got := metricsRequests(rec); got != 2 {
		t.Fatalf("GET /projects/p1/metrics = %d, want 2", got)
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "Horas reales: 9 h") || strings.Contains(view, "7.5") {
		t.Fatalf("the refreshed answer should replace the old one: %q", view)
	}
	if posts := effortPosts(rec); len(posts) != 1 {
		t.Fatalf("POSTs = %d, want only the original registration", len(posts))
	}
}

func TestEffortResultIgnoresRWhileMetricsLoading(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m, cmd := registerEffort(t, openEffortFormFor(t, server.URL, 0, 1, 0))

	m, again := apply(m, keyMsg("r"))
	if again != nil {
		t.Fatal("r while a query is in flight must not emit another one")
	}
	m = run(t, m, cmd)
	if got := metricsRequests(rec); got != 1 {
		t.Fatalf("GET /projects/p1/metrics = %d, want exactly 1", got)
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "Horas reales: 30.5 h") {
		t.Fatalf("the single query should still be answered: %q", view)
	}
}

func TestEffortResultMetricsFailureKeepsRegistrationSuccess(t *testing.T) {
	tests := []struct {
		name       string
		options    effortFlowOptions
		wantReason string
	}{
		{"schema without the route (404)", effortFlowOptions{metricsStatus: http.StatusNotFound}, "404"},
		{"server error (500)", effortFlowOptions{metricsStatus: http.StatusInternalServerError}, "500"},
		{"transport error", effortFlowOptions{metricsDrop: true}, "Error de conexión"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := effortFlowServer(t, tt.options)
			m := resultWithMetrics(t, server.URL)

			view := flatView(stripANSI(m.View()))
			if m.Screen != "effort-result" {
				t.Fatalf("Screen = %q, want the result screen to stay", m.Screen)
			}
			for _, want := range []string{"Esfuerzo registrado", "las métricas del proyecto no están disponibles", tt.wantReason, "r para reintentar", "Implementar login"} {
				if !strings.Contains(view, want) {
					t.Errorf("the screen should say %q: %q", want, view)
				}
			}
			for _, banned := range []string{"✗", "No se pudo confirmar el registro", "Cargando métricas", "Horas reales", "volver a registrar"} {
				if strings.Contains(view, banned) {
					t.Errorf("a metrics failure is not a registration error, found %q: %q", banned, view)
				}
			}
		})
	}
}

func TestEffortResultRetryAfterMetricsFailureSucceeds(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{metricsStatus: http.StatusNotFound, metricsFailures: 1})
	m := resultWithMetrics(t, server.URL)
	if view := flatView(stripANSI(m.View())); !strings.Contains(view, "no están disponibles") {
		t.Fatalf("setup: the first query should have failed: %q", view)
	}

	m, cmd := apply(m, keyMsg("r"))
	if cmd == nil {
		t.Fatal("r should retry after a failed query")
	}
	m = run(t, m, cmd)
	view := stripANSI(m.View())
	for _, want := range []string{"Horas reales: 30.5 h", "Desviación: -4"} {
		if !strings.Contains(view, want) {
			t.Errorf("the retry should show %q: %q", want, view)
		}
	}
	for _, banned := range []string{"no están disponibles", "r para reintentar", "404"} {
		if strings.Contains(view, banned) {
			t.Errorf("the failure notice must disappear, found %q: %q", banned, view)
		}
	}
	if got := metricsRequests(rec); got != 2 {
		t.Errorf("GET /projects/p1/metrics = %d, want 2", got)
	}
}

func TestEffortResultMetricsBodyNotInterpretableIsUnavailable(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{metricsBodies: []string{"["}})
	m := resultWithMetrics(t, server.URL)

	view := flatView(stripANSI(m.View()))
	for _, want := range []string{"Esfuerzo registrado", "no están disponibles", "No se pudo interpretar la respuesta de la API", "r para reintentar"} {
		if !strings.Contains(view, want) {
			t.Errorf("an unreadable metrics body should be reported (%q): %q", want, view)
		}
	}
	for _, banned := range []string{"Cargando métricas", "Horas reales", "✗"} {
		if strings.Contains(view, banned) {
			t.Errorf("found %q: %q", banned, view)
		}
	}
}

// D6: a 201 whose body is not a record still communicates the registration and
// shows the body as received. The raw-body branch was written with 5.2, ahead of
// this test; the label "Respuesta de la API:" is what this test drove.
func TestEffortResultUnreadableCreatedBodyShowsRawBody(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{postStatus: http.StatusCreated, postBody: "no-json"})
	m, cmd := registerEffort(t, openEffortFormFor(t, server.URL, 0, 1, 0))

	view := stripANSI(m.View())
	for _, want := range []string{"Esfuerzo registrado", "Respuesta de la API: no-json", "Historia: Primera historia", "Cargando métricas"} {
		if !strings.Contains(view, want) {
			t.Errorf("the screen should show %q: %q", want, view)
		}
	}
	for _, banned := range []string{"✗", "No se pudo confirmar el registro"} {
		if strings.Contains(view, banned) {
			t.Errorf("an unreadable 201 body is not a registration error, found %q: %q", banned, view)
		}
	}
	m = run(t, m, cmd)
	if got := metricsRequests(rec); got != 1 {
		t.Fatalf("the metrics should still be queried, GET /metrics = %d", got)
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "Horas reales: 30.5 h") || !strings.Contains(view, "no-json") {
		t.Errorf("the metrics should join the raw body: %q", view)
	}
}

// footerOf returns the key hints under the panel of a rendered view.
func footerOf(t *testing.T, view string) string {
	t.Helper()
	_, footer, found := strings.Cut(view, "╯")
	if !found {
		t.Fatalf("view has no panel border to split the footer from: %q", view)
	}
	return footer
}

func TestEffortResultKeys(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	result := func(t *testing.T) tui.Model { return resultWithMetrics(t, server.URL) }

	t.Run("q quits", func(t *testing.T) {
		if _, cmd := apply(result(t), keyMsg("q")); !isQuit(cmd) {
			t.Fatal("q should quit from the result screen")
		}
	})
	t.Run("question mark opens the help", func(t *testing.T) {
		m, _ := apply(result(t), keyMsg("?"))
		if view := stripANSI(m.View()); !strings.Contains(view, "Atajos de teclado") {
			t.Fatalf("? should open the help: %q", view)
		}
	})
	t.Run("p opens the project picker", func(t *testing.T) {
		m, cmd := apply(result(t), keyMsg("p"))
		if m.Screen != "project-picker" || cmd == nil {
			t.Fatalf("p should open the project picker and load it (screen %q, cmd %v)", m.Screen, cmd != nil)
		}
	})
	t.Run("Esc goes back", func(t *testing.T) {
		m, cmd := apply(result(t), tea.KeyMsg{Type: tea.KeyEsc})
		if m.Screen != "section" || cmd != nil {
			t.Fatalf("Esc should leave the result without a command (screen %q, cmd %v)", m.Screen, cmd != nil)
		}
	})
	t.Run("Enter does nothing", func(t *testing.T) {
		m := result(t)
		want := stripANSI(m.View())
		after, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
		if cmd != nil || after.Screen != "effort-result" || stripANSI(after.View()) != want {
			t.Fatalf("Enter must not be bound on the result screen (screen %q, cmd %v)", after.Screen, cmd != nil)
		}
	})
}

func TestEffortResultFooterAndHelpShowOnlyResultKeys(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := resultWithMetrics(t, server.URL)
	help, _ := apply(m, keyMsg("?"))

	footer := footerOf(t, stripANSI(m.View()))
	for _, want := range []string{"r refrescar métricas", "Esc volver", "p proyecto", "? ayuda", "q salir"} {
		if !strings.Contains(flatView(footer), want) {
			t.Errorf("the footer should offer %q: %q", want, footer)
		}
	}
	helpView := stripANSI(help.View())
	if !strings.Contains(helpView, "Atajos de teclado") {
		t.Fatalf("? should open the help: %q", helpView)
	}
	for _, key := range []string{"r", "Esc", "p", "?", "q"} {
		row := regexp.MustCompile(`│\s+` + regexp.QuoteMeta(key) + `\s{2,}\S`)
		if !row.MatchString(helpView) {
			t.Errorf("the help should list a %q row: %q", key, helpView)
		}
	}
	for name, text := range map[string]string{"footer": footer, "help": helpView} {
		for _, banned := range []string{"Tab", "←/→", "Enter"} {
			if strings.Contains(text, banned) {
				t.Errorf("the %s must not mention %q: %q", name, banned, text)
			}
		}
	}
}

func TestEffortResultEscReturnsToTasksOfSameStoryWithoutRequests(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := resultWithMetrics(t, server.URL)
	before := len(rec.all())

	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatal("Esc must not emit a command")
	}
	view := stripANSI(m.View())
	for _, want := range []string{"Tarea de Primera historia o la historia completa.", "Toda la historia (sin tarea)", "Diseñar", "Probar"} {
		if !strings.Contains(view, want) {
			t.Errorf("the task selector of the same story should show %q: %q", want, view)
		}
	}
	if line := focusedLine(view); !strings.Contains(line, "Diseñar") {
		t.Errorf("the cursor should be where the task was chosen: %q", line)
	}
	m = press(m, tea.KeyEsc)
	if view := stripANSI(m.View()); !strings.Contains(view, "Segunda historia") {
		t.Errorf("another Esc should reach the stories: %q", view)
	}
	if after := len(rec.all()); after != before {
		t.Fatalf("walking back emitted %d requests, want none", after-before)
	}
}

func TestEffortSecondRegistrationStartsWithInitialForm(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := resultWithMetrics(t, server.URL).WithClock(fixedClock)
	m = press(m, tea.KeyEsc)
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

// [caracterización] born green: leaveEffortResult clears the flow state, so the
// guard of applyEffortMetrics already drops this answer.
func TestEffortLateMetricsAfterLeavingResultIsDropped(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m, cmd := registerEffort(t, openEffortFormFor(t, server.URL, 0, 1, 0))
	m = press(m, tea.KeyEsc)
	want := stripANSI(m.View())

	m = run(t, m, cmd) // the late metrics answer
	if got := stripANSI(m.View()); got != want || m.Screen != "section" || !strings.Contains(got, "Diseñar") {
		t.Fatalf("a late metrics answer must not change the task selector (screen %q):\nwant %q\ngot  %q", m.Screen, want, got)
	}
}

// registerAgain walks from the restored task selector through the same task and
// member to a second confirmed registration, returning its pending metrics command.
func registerAgain(t *testing.T, m tui.Model) (tui.Model, tea.Cmd) {
	t.Helper()
	m = stepRunning(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // members of the task
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})       // opens the form
	return registerEffort(t, m)
}

// [caracterización] born green: the sequence guard shipped with 5.2. A mutant
// without the seq comparison shows «3 h» and fails both cases.
func TestEffortStaleMetricsFromPreviousRegistrationNeverShown(t *testing.T) {
	bodies := []string{`{"project_id":"p1","actual_hours":3,"deviation":0}`, `{"project_id":"p1","actual_hours":8,"deviation":1}`}
	// first answers to the first query and second to the second, in call order
	// (the fixture serves the bodies in the order the requests arrive).
	newSecondResult := func(t *testing.T) (m tui.Model, first, second tea.Msg) {
		t.Helper()
		server, _ := effortFlowServer(t, effortFlowOptions{metricsBodies: bodies})
		m, firstCmd := registerEffort(t, openEffortFormFor(t, server.URL, 0, 1, 0))
		first = firstCmd() // GET number one, answered with 3 h
		m = press(m, tea.KeyEsc)
		m, secondCmd := registerAgain(t, m)
		return m, first, secondCmd()
	}
	shows := func(m tui.Model, text string) bool { return strings.Contains(stripANSI(m.View()), text) }

	t.Run("old answer arrives first", func(t *testing.T) {
		m, first, second := newSecondResult(t)
		m, _ = apply(m, first)
		if shows(m, "Horas reales: 3 h") || !shows(m, "Cargando métricas…") {
			t.Fatalf("the answer of the first registration must be ignored while the second is loading: %q", stripANSI(m.View()))
		}
		m, _ = apply(m, second)
		if !shows(m, "Horas reales: 8 h") {
			t.Fatalf("the second answer should end up shown: %q", stripANSI(m.View()))
		}
	})
	t.Run("old answer arrives last", func(t *testing.T) {
		m, first, second := newSecondResult(t)
		m, _ = apply(m, second)
		m, _ = apply(m, first)
		if shows(m, "Horas reales: 3 h") || !shows(m, "Horas reales: 8 h") {
			t.Fatalf("a late answer of the first registration must not replace the second: %q", stripANSI(m.View()))
		}
	})
}

func TestEffortProjectSwitchFromResultReopensStoriesAndDiscardsFlow(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := resultWithMetrics(t, server.URL).WithClock(fixedClock)

	m, load := apply(m, keyMsg("p"))
	m, cmd := confirmProjectInPicker(t, m, load, "p2")
	if cmd == nil {
		t.Fatal("choosing the project should load its stories")
	}
	m = run(t, m, cmd)

	view := stripANSI(m.View())
	if m.Screen != "section" || !strings.Contains(view, "Historia de Beta") || !strings.Contains(view, "Proyecto: Beta") {
		t.Fatalf("Esfuerzo should reopen on the stories of Beta (screen %q): %q", m.Screen, view)
	}
	for _, stale := range []string{"Esfuerzo registrado", "Implementar login", "Horas reales", "Ada Lovelace", "Diseñar", "Primera historia"} {
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

// [caracterización] born green: a project change from a selector already reopened
// the section (returnTo == "section").
func TestEffortProjectSwitchFromSelectorDiscardsFlow(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortMembers(t, server.URL, 0, 1)

	m, load := apply(m, keyMsg("p"))
	m, cmd := confirmProjectInPicker(t, m, load, "p2")
	m = run(t, m, cmd)

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

func TestEffortResultMetricsAnsweredWhileProjectPickerIsOpenAreKept(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m, metrics := registerEffort(t, openEffortFormFor(t, server.URL, 0, 1, 0))

	m, _ = apply(m, keyMsg("p")) // the query is still in flight
	m = run(t, m, metrics)
	m = press(m, tea.KeyEsc) // back to the result without choosing a project

	view := stripANSI(m.View())
	if m.Screen != "effort-result" || !strings.Contains(view, "Horas reales: 30.5 h") || strings.Contains(view, "Cargando métricas") {
		t.Fatalf("the answer that arrived behind the picker should be shown (screen %q): %q", m.Screen, view)
	}
}
