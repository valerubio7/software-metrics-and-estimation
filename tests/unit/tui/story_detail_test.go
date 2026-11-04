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
