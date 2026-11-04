package tui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

// detailStories is the backlog used by the story detail tests: a complete story,
// one with every optional value missing and two more for the filter scenarios.
func detailStories() []tui.Story {
	return []tui.Story{
		{ID: "s1", ProjectID: "p1", Title: "Login", Description: "Ingreso de usuarios", Priority: "alta", Status: "pendiente",
			StoryPoints: intPtr(5), AcceptanceCriteria: []string{"A", "B"}, EstimatedHours: floatPtr(8)},
		{ID: "s2", ProjectID: "p1", Title: "Pago", Priority: "media", Status: "en progreso", AcceptanceCriteria: []string{}},
		{ID: "s3", ProjectID: "p1", Title: "Perfil", Description: "Datos personales", Priority: "baja", Status: "pendiente",
			AcceptanceCriteria: []string{"C"}},
	}
}

// detailServer serves the project collection and the given backlog.
func detailServer(t *testing.T, stories []tui.Story) (string, *recorder) {
	t.Helper()
	server, rec := fakeBacklogAPI(t, apiStubs{list: stub{status: 200, body: storiesBody(t, stories)}})
	return server.URL, rec
}

// openStoryDetailWithProject goes browser -> pid -> Backlog, highlights the
// storyIndex-th story and presses Enter. It requires the detail screen and no
// command, because opening the detail must not issue any request.
func openStoryDetailWithProject(t *testing.T, url, pid string, storyIndex int) tui.Model {
	t.Helper()
	m := openBacklogWithProject(t, url, pid)
	for i := 0; i < storyIndex; i++ {
		m = press(m, tea.KeyDown)
	}
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != "story-detail" {
		t.Fatalf("Screen = %q after Enter on story %d, want story-detail", m.Screen, storyIndex)
	}
	if cmd != nil {
		t.Fatal("opening the story detail must not issue a command")
	}
	return m
}

func TestStoryDetailOpensWithEnterShowingTheSevenFields(t *testing.T) {
	url, rec := detailServer(t, detailStories())
	m := openBacklogWithProject(t, url, "p1")
	before := len(rec.all())

	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.Screen != "story-detail" {
		t.Fatalf("Screen = %q, want story-detail", m.Screen)
	}
	if cmd != nil || len(rec.all()) != before {
		t.Fatalf("opening the detail must not emit requests (cmd nil = %v, requests %d -> %d)", cmd == nil, before, len(rec.all()))
	}
	view := stripANSI(m.View())
	for label, want := range map[string]string{
		"Título":                  "Login",
		"Descripción":             "Ingreso de usuarios",
		"Prioridad":               "alta",
		"Estado":                  "pendiente",
		"Story points":            "5",
		"Criterios de aceptación": "A; B",
		"Horas estimadas":         "8",
	} {
		if got := fieldValue(t, view, label); got != want {
			t.Errorf("%s = %q, want %q", label, got, want)
		}
	}
}

func TestStoryDetailShowsDashForMissingValues(t *testing.T) {
	stories := []tui.Story{{ID: "s1", ProjectID: "p1", Title: "Sin datos", Priority: "alta", Status: "pendiente", AcceptanceCriteria: []string{}}}
	url, _ := detailServer(t, stories)
	m := openStoryDetailWithProject(t, url, "p1", 0)

	view := stripANSI(m.View())
	for _, label := range []string{"Descripción", "Story points", "Criterios de aceptación", "Horas estimadas"} {
		if got := fieldValue(t, view, label); got != "—" {
			t.Errorf("%s = %q, want a dash", label, got)
		}
	}
	// Triangulation: the fields that do have a value keep showing it.
	for label, want := range map[string]string{"Título": "Sin datos", "Prioridad": "alta", "Estado": "pendiente"} {
		if got := fieldValue(t, view, label); got != want {
			t.Errorf("%s = %q, want %q", label, got, want)
		}
	}
}

func TestStoryDetailShowsOnlyStoryFieldsAndNoExtraRequests(t *testing.T) {
	url, rec := detailServer(t, detailStories())
	m := openBacklogWithProject(t, url, "p1")
	before := len(rec.all())

	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.Screen != "story-detail" || cmd != nil {
		t.Fatalf("Screen = %q, cmd nil = %v, want story-detail without a command", m.Screen, cmd == nil)
	}
	for _, request := range rec.all()[before:] {
		t.Errorf("unexpected request after opening the detail: %s %s", request.Method, request.Path)
	}
	for _, request := range rec.all() {
		for _, forbidden := range []string{"/tasks", "/defects", "/effort-records"} {
			if strings.Contains(request.Path, forbidden) {
				t.Errorf("the detail must not query %s, got %s %s", forbidden, request.Method, request.Path)
			}
		}
	}
	view := stripANSI(m.View())
	for _, forbidden := range []string{"Tareas", "Defectos asociados", "Registros de esfuerzo"} {
		if strings.Contains(view, forbidden) {
			t.Errorf("the detail must not show %q:\n%s", forbidden, view)
		}
	}
}

func TestBacklogEnterOnEmptyListDoesNotOpenDetail(t *testing.T) {
	url, rec := detailServer(t, nil)
	m := openBacklogWithProject(t, url, "p1")
	before := len(rec.all())

	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.Screen != "backlog" {
		t.Fatalf("Screen = %q, want backlog", m.Screen)
	}
	if cmd != nil || len(rec.all()) != before {
		t.Fatalf("Enter on an empty list must not emit requests (cmd nil = %v)", cmd == nil)
	}
}

func TestStoryDetailOpensTheHighlightedStoryUnderFilter(t *testing.T) {
	for _, tc := range []struct {
		name   string
		filter string
		downs  int
		want   string
	}{
		{"un solo resultado", "pago", 0, "Pago"},
		{"segundo resultado del filtro", "p", 1, "Perfil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			url, _ := detailServer(t, detailStories())
			m := openBacklogWithProject(t, url, "p1")
			m, _ = apply(m, keyMsg("/"))
			m = typeText(m, tc.filter)
			m = press(m, tea.KeyEnter) // stop searching, keep the filter
			for i := 0; i < tc.downs; i++ {
				m = press(m, tea.KeyDown)
			}

			m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})

			if m.Screen != "story-detail" {
				t.Fatalf("Screen = %q, want story-detail", m.Screen)
			}
			if got := fieldValue(t, stripANSI(m.View()), "Título"); got != tc.want {
				t.Errorf("Título = %q, want the highlighted story %q", got, tc.want)
			}
		})
	}
}

func TestBacklogEnterWhileTypingFilterDoesNotOpenDetail(t *testing.T) {
	url, _ := detailServer(t, detailStories())
	m := openBacklogWithProject(t, url, "p1")
	m, _ = apply(m, keyMsg("/"))
	m = typeText(m, "pago")

	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.Screen != "backlog" || cmd != nil {
		t.Fatalf("first Enter: Screen = %q, cmd nil = %v, want backlog and no command", m.Screen, cmd == nil)
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "Buscar: pago") || strings.Contains(view, "Buscar: pago_") {
		t.Fatalf("the first Enter should stop editing but keep the filter:\n%s", view)
	}

	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.Screen != "story-detail" {
		t.Fatalf("second Enter: Screen = %q, want story-detail", m.Screen)
	}
	if got := fieldValue(t, stripANSI(m.View()), "Título"); got != "Pago" {
		t.Errorf("Título = %q, want Pago", got)
	}
}

// [caracterización] The detail never offered to edit the story: it is born green
// because no key does anything there yet, and it keeps pinning the spec rule
// once the detail gets its own keys.
func TestStoryDetailEKeyDoesNothing(t *testing.T) {
	url, rec := detailServer(t, detailStories())
	m := openStoryDetailWithProject(t, url, "p1", 0)
	before := len(rec.all())

	m, cmd := apply(m, keyMsg("e"))

	if m.Screen != "story-detail" || cmd != nil || len(rec.all()) != before {
		t.Fatalf("Screen = %q, cmd nil = %v, want the detail untouched", m.Screen, cmd == nil)
	}
	if got := fieldValue(t, stripANSI(m.View()), "Título"); got != "Login" {
		t.Errorf("Título = %q, want Login", got)
	}
}

// footerOf is the key hints under the content panel of a rendered view.
func footerOf(t *testing.T, view string) string {
	t.Helper()
	end := strings.LastIndex(view, "╯")
	if end < 0 {
		t.Fatalf("view has no content panel:\n%s", view)
	}
	return strings.Join(strings.Fields(view[end:]), " ")
}

// [caracterización] The detail is not a tab root, so Tab, Shift+Tab and the
// digits never reached the tab switch; the test pins that rule for the new screen.
func TestStoryDetailTabAndDigitsDoNotSwitchTab(t *testing.T) {
	url, rec := detailServer(t, detailStories())
	m := openStoryDetailWithProject(t, url, "p1", 0)
	before := len(rec.all())

	for name, key := range map[string]tea.KeyMsg{
		"Tab":       {Type: tea.KeyTab},
		"Shift+Tab": {Type: tea.KeyShiftTab},
		"1":         keyMsg("1"),
		"2":         keyMsg("2"),
		"3":         keyMsg("3"),
		"4":         keyMsg("4"),
	} {
		t.Run(name, func(t *testing.T) {
			next, cmd := apply(m, key)
			if next.Screen != "story-detail" || cmd != nil {
				t.Errorf("Screen = %q, cmd nil = %v, want the detail without a command", next.Screen, cmd == nil)
			}
		})
	}
	if len(rec.all()) != before {
		t.Errorf("tab keys emitted %d requests, want none", len(rec.all())-before)
	}
}

func TestStoryDetailQuitsOnQ(t *testing.T) {
	url, _ := detailServer(t, detailStories())
	m := openStoryDetailWithProject(t, url, "p1", 0)

	for _, key := range []string{"q", "Q"} {
		if _, cmd := apply(m, keyMsg(key)); !isQuit(cmd) {
			t.Errorf("%s in the story detail should quit", key)
		}
	}
}

func TestStoryDetailQuestionMarkOpensHelp(t *testing.T) {
	url, _ := detailServer(t, detailStories())
	m := openStoryDetailWithProject(t, url, "p1", 0)

	m, _ = apply(m, keyMsg("?"))

	if view := stripANSI(m.View()); !strings.Contains(view, "Atajos de teclado") {
		t.Fatalf("? should open the help:\n%s", view)
	}
	m = press(m, tea.KeyEsc)
	if m.Screen != "story-detail" || strings.Contains(stripANSI(m.View()), "Atajos de teclado") {
		t.Errorf("Esc should close the help and stay on the detail (Screen = %q)", m.Screen)
	}
}

func TestStoryDetailHighlightsBacklogTabAndCrumb(t *testing.T) {
	url, _ := detailServer(t, detailStories())
	m := openStoryDetailWithProject(t, url, "p1", 0)

	view := stripANSI(m.View())

	// The workspace header (project and tab bar) is drawn only on a tab screen.
	for _, want := range []string{"◆ Alpha", "1 Dashboard  2 Backlog  3 Sprints", "Proyectos › Alpha › Backlog › Detalle"} {
		if !strings.Contains(view, want) {
			t.Errorf("view lacks %q:\n%s", want, view)
		}
	}
}

func TestStoryDetailFooterOffersEscAndHelp(t *testing.T) {
	url, _ := detailServer(t, detailStories())
	m := openStoryDetailWithProject(t, url, "p1", 0)

	footer := footerOf(t, stripANSI(m.View()))

	for _, want := range []string{"Esc backlog", "? más"} {
		if !strings.Contains(footer, want) {
			t.Errorf("footer %q lacks %q", footer, want)
		}
	}
	for _, unwanted := range []string{"Tab", "1–4"} {
		if strings.Contains(footer, unwanted) {
			t.Errorf("footer %q must not offer %q", footer, unwanted)
		}
	}
}

func TestBacklogFooterOffersDetailAndEditWithoutExceedingTheLimit(t *testing.T) {
	url, _ := detailServer(t, detailStories())
	m := openBacklogWithProject(t, url, "p1")

	footer := footerOf(t, stripANSI(m.View()))

	for _, want := range []string{"Enter detalle", "e editar", "n crear", "Esc proyectos", "? más"} {
		if !strings.Contains(footer, want) {
			t.Errorf("footer %q lacks %q", footer, want)
		}
	}
	if strings.Contains(footer, "Enter editar") || strings.Contains(footer, "/ buscar") {
		t.Errorf("footer %q still offers Enter as edit or the search among the primary keys", footer)
	}

	m, _ = apply(m, keyMsg("?"))
	help := stripANSI(m.View())
	for _, want := range []string{"Abrir el detalle de la historia resaltada", "Editar la historia resaltada", "Buscar o filtrar historias"} {
		if !strings.Contains(help, want) {
			t.Errorf("help lacks %q:\n%s", want, help)
		}
	}
}

func TestBacklogHintAnnouncesDetailAndEdit(t *testing.T) {
	url, _ := detailServer(t, detailStories())
	m := openBacklogWithProject(t, url, "p1")

	view := stripANSI(m.View())

	if !strings.Contains(view, "Enter abre el detalle · e edita") {
		t.Errorf("the backlog hint should announce Enter as detail and e as edit:\n%s", view)
	}
}

// requestLines renders the requests recorded after the first `from` ones as
// "METHOD /path" lines.
func requestLines(rec *recorder, from int) []string {
	var lines []string
	for _, request := range rec.all()[from:] {
		lines = append(lines, request.Method+" "+request.Path)
	}
	return lines
}

func TestStoryDetailActionRowsInOrder(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openStoryDetailWithProject(t, server.URL, "p1", 0)

	view := stripANSI(m.View())

	estimate, effort, defect := strings.Index(view, "Estimar"), strings.Index(view, "Registrar esfuerzo"), strings.Index(view, "Reportar defecto")
	if estimate < 0 || effort < estimate || defect < effort {
		t.Fatalf("the rows must be Estimar, Registrar esfuerzo and Reportar defecto in that order (offsets %d, %d, %d):\n%s", estimate, effort, defect, view)
	}
	if line := focusedLine(view); !strings.Contains(line, "Estimar") {
		t.Errorf("the first row should be highlighted, got %q", line)
	}
}

func TestStoryDetailArrowsMoveHighlightWithinBounds(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openStoryDetailWithProject(t, server.URL, "p1", 0)
	highlighted := func() string { return focusedLine(stripANSI(m.View())) }

	m = press(m, tea.KeyUp)
	if line := highlighted(); !strings.Contains(line, "Estimar") {
		t.Errorf("Up on the first row must stay on it, got %q", line)
	}
	m = press(m, tea.KeyDown)
	if line := highlighted(); !strings.Contains(line, "Registrar esfuerzo") {
		t.Errorf("Down should highlight the next row, got %q", line)
	}
	m = press(m, tea.KeyDown)
	if line := highlighted(); !strings.Contains(line, "Reportar defecto") {
		t.Errorf("Down should highlight the third row, got %q", line)
	}
	for i := 0; i < 5; i++ {
		m = press(m, tea.KeyDown)
	}
	if line := highlighted(); !strings.Contains(line, "Reportar defecto") {
		t.Errorf("Down on the last row must stay on it, got %q", line)
	}
	m = press(m, tea.KeyUp)
	if line := highlighted(); !strings.Contains(line, "Registrar esfuerzo") {
		t.Errorf("Up should highlight the previous row, got %q", line)
	}
	m = press(m, tea.KeyUp)
	if line := highlighted(); !strings.Contains(line, "Estimar") {
		t.Errorf("Up should highlight the first row, got %q", line)
	}
	if m.Screen != "story-detail" {
		t.Errorf("moving the highlight must not leave the detail, Screen = %q", m.Screen)
	}
}

func TestStoryDetailEnterRunsHighlightedAction(t *testing.T) {
	for _, tc := range []struct {
		name      string
		downs     int
		wantGET   string
		wantTitle string
		wantRow   string
	}{
		{"registrar esfuerzo abre las tareas de la misma historia", 1, "GET /projects/p1/stories/s1/tasks", "Esfuerzo · Tareas", "Toda la historia (sin tarea)"},
		{"estimar abre los participantes de la misma historia", 0, "GET /projects/p1/members", "Planning Poker · Participantes", "Ada Lovelace"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, rec := effortFlowServer(t, effortFlowOptions{})
			m := openStoryDetailWithProject(t, server.URL, "p1", 0)
			for i := 0; i < tc.downs; i++ {
				m = press(m, tea.KeyDown)
			}
			before := len(rec.all())

			m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
			if cmd == nil {
				t.Fatal("Enter on an action row should load the first step of the flow")
			}
			m = run(t, m, cmd)

			if m.Screen != "section" {
				t.Fatalf("Screen = %q, want section", m.Screen)
			}
			view := stripANSI(m.View())
			for _, want := range []string{tc.wantTitle, tc.wantRow} {
				if !strings.Contains(view, want) {
					t.Errorf("view lacks %q:\n%s", want, view)
				}
			}
			if strings.Contains(view, "Segunda historia") {
				t.Errorf("the flow must not offer a story selector:\n%s", view)
			}
			if got := requestLines(rec, before); len(got) != 1 || got[0] != tc.wantGET {
				t.Errorf("requests = %v, want only %q", got, tc.wantGET)
			}
		})
	}
}

func TestEscFromPokerMembersReturnsToStoryDetailWithoutRequests(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openStoryFlowWithProject(t, server.URL, "p1", 0, "p")
	before := len(rec.all())

	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEsc})

	if m.Screen != "story-detail" || cmd != nil {
		t.Fatalf("Screen = %q, cmd nil = %v, want story-detail without a command", m.Screen, cmd == nil)
	}
	if got := requestLines(rec, before); len(got) != 0 {
		t.Errorf("Esc emitted requests: %v", got)
	}
	if got := fieldValue(t, stripANSI(m.View()), "Título"); got != "Primera historia" {
		t.Errorf("Título = %q, want the story the flow started from", got)
	}
}

func TestEscFromFirstStepAfterAPIErrorReturnsToStoryDetail(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  string
		opts effortFlowOptions
		want string
	}{
		{"participantes", "p", effortFlowOptions{membersFailures: 1}, "members down"},
		{"tareas", "t", effortFlowOptions{tasksFailures: 1}, "tasks down"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, _ := effortFlowServer(t, tc.opts)
			m := openStoryFlowWithProject(t, server.URL, "p1", 0, tc.key)
			if view := stripANSI(m.View()); !strings.Contains(view, tc.want) || strings.Contains(view, "Toda la historia") {
				t.Fatalf("the failed step should show the API error without the synthetic row:\n%s", view)
			}

			m = press(m, tea.KeyEsc)

			if m.Screen != "story-detail" {
				t.Errorf("Esc after the error: Screen = %q, want story-detail", m.Screen)
			}
		})
	}
}

func TestEscFromFirstStepWithFilterClearsFilterFirst(t *testing.T) {
	for _, tc := range []struct {
		name, key, filter, kept, hidden string
	}{
		{"participantes", "p", "gra", "Grace Hopper", "Ada Lovelace"},
		{"tareas", "t", "dis", "Diseñar", "Probar"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, _ := effortFlowServer(t, effortFlowOptions{})
			m := openStoryFlowWithProject(t, server.URL, "p1", 0, tc.key)
			m = typeText(m, tc.filter)
			m = press(m, tea.KeyEnter) // stop typing, keep the filter
			if view := stripANSI(m.View()); !strings.Contains(view, tc.kept) || strings.Contains(view, tc.hidden) {
				t.Fatalf("the filter should narrow the list to %q and hide %q:\n%s", tc.kept, tc.hidden, view)
			}

			m = press(m, tea.KeyEsc)

			if view := stripANSI(m.View()); m.Screen != "section" || !strings.Contains(view, tc.kept) || !strings.Contains(view, tc.hidden) {
				t.Fatalf("first Esc: Screen = %q, want the same step with the filter cleared:\n%s", m.Screen, view)
			}
			if m = press(m, tea.KeyEsc); m.Screen != "story-detail" {
				t.Errorf("second Esc: Screen = %q, want story-detail", m.Screen)
			}
		})
	}
}

func TestLateTasksResponseOfAnotherStoryDoesNotPopulateTheSelector(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openStoryDetailWithProject(t, server.URL, "p1", 0)
	m, tasksOfA := apply(m, keyMsg("t")) // story A: the query is in flight
	lateA := tasksOfA()                  // its answer, held back until after the switch
	m = press(m, tea.KeyEsc)             // detail of A
	m = press(m, tea.KeyEsc)             // backlog
	m = press(m, tea.KeyDown)
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter}) // detail of story B
	m, tasksOfB := apply(m, keyMsg("t"))
	if m.Screen != "section" || tasksOfB == nil {
		t.Fatalf("Screen = %q, cmd nil = %v, want the task selector of B loading", m.Screen, tasksOfB == nil)
	}

	m, _ = apply(m, lateA)

	view := stripANSI(m.View())
	if strings.Contains(view, "Diseñar") || !strings.Contains(view, "Cargando") {
		t.Fatalf("the tasks of A must not populate the selector of B, which keeps loading:\n%s", view)
	}
	m = run(t, m, tasksOfB)
	view = stripANSI(m.View())
	if !strings.Contains(view, "Segunda historia") || !strings.Contains(view, "Toda la historia (sin tarea)") || strings.Contains(view, "Diseñar") {
		t.Errorf("the selector of B should show its own tasks:\n%s", view)
	}
}

// openBacklogOfProject walks back with Esc to the project browser, opens pid and
// its Backlog tab, applying the list.
func openBacklogOfProject(t *testing.T, m tui.Model, pid string) tui.Model {
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
		t.Fatal("opening Backlog should load the stories of the new project")
	}
	return run(t, m, cmd)
}

// [caracterización] Born green: opening a detail and a flow overwrites the state
// of the previous project anyway. It pins the user-visible rule that the state
// reset in selectProject (tested in the tui package) is there to protect.
func TestChangingProjectDiscardsDetailAndReturnTarget(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openStoryDetailWithProject(t, server.URL, "p1", 0)

	m = openBacklogOfProject(t, m, "p2")

	view := stripANSI(m.View())
	if m.Screen != "backlog" || !strings.Contains(view, "Historia de Beta") || strings.Contains(view, "Detalle de la historia") {
		t.Fatalf("Screen = %q, want the backlog list of p2 and not a detail:\n%s", m.Screen, view)
	}
	m = press(m, tea.KeyEnter) // detail of the story of p2
	m = stepRunning(t, m, keyMsg("t"))
	if m.Screen != "section" {
		t.Fatalf("Screen = %q, want the task selector of p2", m.Screen)
	}

	m = press(m, tea.KeyEsc)

	if got := fieldValue(t, stripANSI(m.View()), "Título"); m.Screen != "story-detail" || got != "Historia de Beta" {
		t.Errorf("cancelling should return to the detail of the story of p2 (Screen %q, Título %q)", m.Screen, got)
	}
}

func TestFlowScreensHighlightBacklogTab(t *testing.T) {
	const flow = "Proyectos › Alpha › Backlog › Detalle"
	server, _ := effortFlowServer(t, effortFlowOptions{})
	for _, tc := range []struct {
		name    string
		screen  string
		reach   func(t *testing.T) tui.Model
		wantEnd string
	}{
		{"participantes de Estimar", "section", func(t *testing.T) tui.Model { return openStoryFlowWithProject(t, server.URL, "p1", 0, "p") }, " › Estimación › Participantes"},
		{"Planning Poker", "planning-poker", func(t *testing.T) tui.Model {
			m := openStoryFlowWithProject(t, server.URL, "p1", 0, "p")
			m, _ = apply(m, tea.KeyMsg{Type: tea.KeySpace})
			m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
			return m
		}, " › Estimación › Participantes"},
		{"tareas de Esfuerzo", "section", func(t *testing.T) tui.Model { return openStoryFlowWithProject(t, server.URL, "p1", 0, "t") }, " › Esfuerzo › Tareas"},
		{"integrantes de Esfuerzo", "section", func(t *testing.T) tui.Model {
			return stepRunning(t, openStoryFlowWithProject(t, server.URL, "p1", 0, "t"), tea.KeyMsg{Type: tea.KeyEnter})
		}, " › Esfuerzo › Integrante"},
		{"formulario de esfuerzo", "effort-form", func(t *testing.T) tui.Model { return openEffortFormFromDetail(t, server.URL, 0, 1, 0) }, " › Esfuerzo › Registrar"},
		{"formulario de defecto", "defect-create-form", func(t *testing.T) tui.Model {
			m, _ := apply(openStoryDetailWithProject(t, server.URL, "p1", 0), keyMsg("d"))
			return m
		}, " › Registrar defecto"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := tc.reach(t)

			view := stripANSI(m.View())

			if m.Screen != tc.screen {
				t.Fatalf("Screen = %q, want %q", m.Screen, tc.screen)
			}
			// The tab bar is drawn with the active tab styled, which a test
			// terminal strips; the breadcrumb carries the tab by name.
			for _, want := range []string{"2 Backlog", flow + tc.wantEnd} {
				if !strings.Contains(view, want) {
					t.Errorf("view lacks %q:\n%s", want, view)
				}
			}
		})
	}
}

func TestStoryDetailFooterShowsAtMostFiveKeysIncludingEnterAndEsc(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openStoryDetailWithProject(t, server.URL, "p1", 0)

	footer := footerOf(t, stripANSI(m.View()))

	// Exactly three primary keys plus the help hint: the action shortcuts live in
	// the rows and in the help, so Esc is never pushed out of the footer.
	if want := "╯ ↑/↓ mover · Enter elegir acción · Esc backlog · ? más"; footer != want {
		t.Errorf("footer = %q, want %q", footer, want)
	}
}

func TestStoryDetailHelpListsActionShortcutsAndNotTabs(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openStoryDetailWithProject(t, server.URL, "p1", 0)

	m, _ = apply(m, keyMsg("?"))
	help := stripANSI(m.View())

	actions, general := strings.Index(help, "Acciones"), strings.Index(help, "General")
	if actions < 0 || general < actions {
		t.Fatalf("the help should have an Acciones group before General:\n%s", help)
	}
	section := help[actions:general]
	for key, text := range map[string]string{"p": "Estimar la historia con Planning Poker", "t": "Registrar esfuerzo de la historia", "d": "Reportar un defecto de la historia"} {
		var line string
		for _, candidate := range strings.Split(section, "\n") {
			if strings.Contains(candidate, text) {
				line = candidate
			}
		}
		if fields := strings.Fields(strings.ReplaceAll(line, "│", " ")); len(fields) == 0 || fields[0] != key {
			t.Errorf("the Acciones group should list %q for %q, got line %q:\n%s", key, text, line, help)
		}
	}
	for _, unwanted := range []string{"Tab/Shift+Tab", "1–4"} {
		if strings.Contains(help, unwanted) {
			t.Errorf("the help of the detail must not list %q:\n%s", unwanted, help)
		}
	}
}

func TestStoryDetailShortcutsStartActions(t *testing.T) {
	for _, tc := range []struct {
		key       string
		wantGET   string
		wantTitle string
	}{
		{"p", "GET /projects/p1/members", "Participantes"},
		{"t", "GET /projects/p1/stories/s1/tasks", "Tareas"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			server, rec := effortFlowServer(t, effortFlowOptions{})
			m := openStoryDetailWithProject(t, server.URL, "p1", 0)
			before := len(rec.all())

			m, cmd := apply(m, keyMsg(tc.key))
			if cmd == nil {
				t.Fatalf("%s should load the first step of its flow", tc.key)
			}
			m = run(t, m, cmd)

			if m.Screen != "section" {
				t.Fatalf("Screen = %q, want section", m.Screen)
			}
			if view := stripANSI(m.View()); !strings.Contains(view, tc.wantTitle) || !strings.Contains(view, "Primera historia") {
				t.Errorf("view lacks %q or the story:\n%s", tc.wantTitle, view)
			}
			if got := requestLines(rec, before); len(got) != 1 || got[0] != tc.wantGET {
				t.Errorf("requests = %v, want only %q", got, tc.wantGET)
			}
		})
	}
}
