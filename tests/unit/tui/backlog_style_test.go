package tui_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

var ansiSequence = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// forceColor makes lipgloss emit ANSI sequences even without a TTY and restores
// the previous profile when the test ends. Tests that use it must not run in parallel.
func forceColor(t *testing.T) {
	t.Helper()
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })
}

func stripANSI(s string) string { return ansiSequence.ReplaceAllString(s, "") }

// rawLine returns the first line of view whose ANSI-stripped text contains needle.
func rawLine(t *testing.T, view, needle string) string {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(stripANSI(line), needle) {
			return line
		}
	}
	t.Fatalf("no line containing %q in view %q", needle, stripANSI(view))
	return ""
}

// wantStyled asserts that the line holding needle carries ANSI sequences.
func wantStyled(t *testing.T, view, needle string) string {
	t.Helper()
	line := rawLine(t, view, needle)
	if !ansiSequence.MatchString(line) {
		t.Errorf("line %q has no ANSI styling, want it styled", line)
	}
	return line
}

// wantPlainBody asserts that the view, once stripped of ANSI, still holds the
// unchanged plain text of the screen.
func wantPlainBody(t *testing.T, view, body string) {
	t.Helper()
	if got := stripANSI(view); !strings.Contains(got, body) {
		t.Errorf("stripped View() = %q, want it to contain %q", got, body)
	}
}

// hasBackground reports whether the line carries a 24-bit background color.
func hasBackground(line string) bool { return strings.Contains(line, "48;2;") }

func TestSelectionRowSpansContentWidth(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	view := stripANSI(updated.(tui.Model).View())
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "▸ Proyectos") {
			if w := lipgloss.Width(line); w != 60 {
				t.Errorf("selected row width = %d, want 60 (line %q)", w, line)
			}
			if !strings.HasPrefix(line, "│") || !strings.HasSuffix(line, "│") {
				t.Errorf("selected row = %q, want it inside the content panel", line)
			}
			return
		}
	}
	t.Fatalf("selected row not found in %q", view)
}

func TestStyledChromeShowsBrandAndBreadcrumb(t *testing.T) {
	forceColor(t)
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m := openBacklogWithProject(t, server.URL, "p1")
	view := m.View()

	wantPlainBody(t, view, "Software Metrics & Estimation")
	wantPlainBody(t, view, "API conectada")
	wantPlainBody(t, view, "Inicio › Backlog › Proyecto: Alpha")
	styled := wantStyled(t, view, "Software Metrics & Estimation")
	if !hasBackground(styled) && !ansiSequence.MatchString(styled) {
		t.Errorf("brand line %q has no styling", styled)
	}
}

func TestStyledBacklogListUsesMenuStyle(t *testing.T) {
	forceColor(t)
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m := openBacklogWithProject(t, server.URL, "p1")
	view := m.View()

	wantPlainBody(t, view, "Backlog del proyecto Alpha")
	for _, action := range []string{"Crear historia", "Cambiar proyecto", "Refrescar", "Volver"} {
		wantPlainBody(t, view, action)
	}
	wantPlainBody(t, view, "Login")
	wantPlainBody(t, view, "Alta de usuarios")

	wantStyled(t, view, "Backlog del proyecto Alpha")
	selected := wantStyled(t, view, "▸ Crear historia")
	if !hasBackground(selected) {
		t.Errorf("selected row %q has no background, want the menu selection style", selected)
	}
	if w := lipgloss.Width(stripANSI(selected)); w != 80 {
		t.Errorf("selected row width = %d, want it to fill the 80-column content width", w)
	}
	for _, needle := range []string{"Cambiar proyecto"} {
		line := wantStyled(t, view, needle)
		if hasBackground(line) {
			t.Errorf("unselected row %q has a background, want the item style", line)
		}
	}
	wantStyled(t, view, "mover")
}

func TestStyledBacklogSelectedStoryRowUsesSelectionStyle(t *testing.T) {
	forceColor(t)
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m := openBacklogWithProject(t, server.URL, "p1")
	for i := 0; i < backlogActionCount; i++ {
		m = press(m, tea.KeyDown)
	}
	view := m.View()
	selected := wantStyled(t, view, "▸ Login")
	if !hasBackground(selected) {
		t.Errorf("selected story row %q has no background, want the selection style", selected)
	}
	if line := rawLine(t, view, "Crear historia"); hasBackground(line) {
		t.Errorf("action row %q has a background while a story is selected", line)
	}
}

func TestStyledBacklogEmptyAndLoadingMessagesAreMuted(t *testing.T) {
	forceColor(t)
	t.Run("loading", func(t *testing.T) {
		server, _ := fakeBacklogAPI(t, apiStubs{})
		view := loadingBacklog(t, server.URL, "p1").View()
		wantPlainBody(t, view, "Cargando backlog…")
		wantStyled(t, view, "Cargando backlog…")
	})
	t.Run("empty", func(t *testing.T) {
		server, _ := fakeBacklogAPI(t, apiStubs{list: stub{http.StatusOK, storiesBody(t, nil)}})
		view := openBacklogWithProject(t, server.URL, "p1").View()
		wantPlainBody(t, view, "El backlog no tiene historias.")
		wantStyled(t, view, "El backlog no tiene historias.")
	})
	t.Run("picker loading", func(t *testing.T) {
		view, _ := startBacklog("http://localhost:8080")
		wantPlainBody(t, view.View(), "Elegí un proyecto")
		wantPlainBody(t, view.View(), "Cargando proyectos…")
		wantStyled(t, view.View(), "Cargando proyectos…")
	})
}

func TestStyledProjectPickerShowsStatusBadges(t *testing.T) {
	forceColor(t)
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m, load := startBacklog(server.URL)
	m = run(t, m, load)
	view := m.View()

	wantPlainBody(t, view, "Elegí un proyecto")
	wantPlainBody(t, view, "Alpha")
	wantPlainBody(t, view, "EN CURSO")
	wantStyled(t, view, "Alpha")
}

func TestStyledBacklogFormUsesSelectionStyleAndErrorBanner(t *testing.T) {
	forceColor(t)
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m := press(openBacklogWithProject(t, server.URL, "p1"), tea.KeyEnter) // Crear historia
	view := m.View()

	wantPlainBody(t, view, "Crear historia")
	wantPlainBody(t, view, "▸ Título: ")
	wantPlainBody(t, view, "  Descripción: ")
	wantStyled(t, view, "Crear historia")
	if line := wantStyled(t, view, "▸ Título"); !hasBackground(line) {
		t.Errorf("selected field %q has no background, want the selection style", line)
	}
	if line := wantStyled(t, view, "Descripción:"); hasBackground(line) {
		t.Errorf("unselected field %q has a background, want the item style", line)
	}
	wantStyled(t, view, "Separá los criterios")

	t.Run("local error uses the error style", func(t *testing.T) {
		edit := selectStory(openBacklogWithProject(t, server.URL, "p1"), 0)
		edit = advanceFields(edit, 5)
		edit = typeText(edit, "abc")
		edit, _ = submitUpdate(edit)
		view := edit.View()
		wantPlainBody(t, view, "Horas estimadas debe ser un número")
		line := wantStyled(t, view, "Horas estimadas debe ser un número")
		if !strings.Contains(stripANSI(line), "✗") {
			t.Errorf("error line %q lacks the error marker, want the error style", line)
		}
	})
}

func TestStyledBacklogFailureResultUsesErrorStyle(t *testing.T) {
	forceColor(t)
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m := openBacklogWithProject(t, server.URL, "p1")
	m, _ = apply(m, failure(http.StatusUnprocessableEntity,
		`{"error":"validation_failed","message":"datos inválidos","fields":{"title":"es obligatorio","priority":"valor inválido"}}`))
	view := m.View()

	for _, want := range []string{"Resultado", "La API rechazó los datos (422): datos inválidos", "- priority: valor inválido", "- title: es obligatorio"} {
		wantPlainBody(t, view, want)
	}
	wantStyled(t, view, "Resultado")
	head := wantStyled(t, view, "La API rechazó los datos (422)")
	if !strings.Contains(stripANSI(head), "✗") {
		t.Errorf("error heading %q lacks the error marker", head)
	}
	if strings.Contains(stripANSI(view), "Historia creada") {
		t.Errorf("stripped View() = %q, must not communicate success", stripANSI(view))
	}
}

func TestStyledBacklogSuccessResultUsesOkStyle(t *testing.T) {
	forceColor(t)
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m := openBacklogWithProject(t, server.URL, "p1")
	m, _ = apply(m, tui.BacklogResultMsg{Op: tui.BacklogOpCreate, Response: tui.APIResponse{
		StatusCode: http.StatusCreated,
		Body:       []byte(storyBody(t, tui.Story{ID: "s9", Title: "Historia nueva", Priority: "alta", Status: "pendiente"})),
	}})
	view := m.View()

	wantPlainBody(t, view, "Resultado")
	wantPlainBody(t, view, "Historia creada correctamente (201)")
	wantPlainBody(t, view, "Historia nueva")
	wantStyled(t, view, "Resultado")
	ok := wantStyled(t, view, "Historia creada correctamente (201)")
	if strings.Contains(stripANSI(ok), "✗") {
		t.Errorf("success line %q carries the error marker, want only the ok style", ok)
	}
}

func TestStyledBacklogRespectsNoColorProfile(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })
	server, _ := fakeBacklogAPI(t, apiStubs{})
	view := openBacklogWithProject(t, server.URL, "p1").View()
	if ansiSequence.MatchString(view) {
		t.Errorf("View() = %q, want no ANSI sequences without color support", view)
	}
}
