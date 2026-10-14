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

func TestStyledBacklogListUsesMenuStyle(t *testing.T) {
	forceColor(t)
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m := openBacklogWithProject(t, server.URL, "p1")
	view := m.View()

	wantPlainBody(t, view, "Backlog del proyecto p1\n\n"+
		"▸ Crear historia\n  Cambiar proyecto\n  Volver\n\n"+
		"  Login · prioridad alta · estado pendiente · puntos —\n"+
		"  Alta de usuarios · prioridad media · estado en curso · puntos 3\n\n"+
		"↑/↓ para moverte, Enter para elegir, Esc para volver al menú.\n")

	wantStyled(t, view, "Backlog del proyecto p1")
	selected := wantStyled(t, view, "▸ Crear historia")
	if !hasBackground(selected) {
		t.Errorf("selected row %q has no background, want the menu selection style", selected)
	}
	for _, needle := range []string{"Cambiar proyecto", "Login · prioridad"} {
		line := wantStyled(t, view, needle)
		if hasBackground(line) {
			t.Errorf("unselected row %q has a background, want the item style", line)
		}
	}
	footer := rawLine(t, view, "para moverte")
	if strings.Contains(footer, "Enter para elegir") {
		t.Errorf("footer %q is one unstyled run, want the keys styled apart from the text", footer)
	}
	wantStyled(t, view, "para moverte")
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
		wantPlainBody(t, view, "Cargando backlog…\n")
		wantStyled(t, view, "Cargando backlog…")
	})
	t.Run("empty", func(t *testing.T) {
		server, _ := fakeBacklogAPI(t, apiStubs{list: stub{http.StatusOK, storiesBody(t, nil)}})
		view := openBacklogWithProject(t, server.URL, "p1").View()
		wantPlainBody(t, view, "El backlog no tiene historias.\n")
		wantStyled(t, view, "El backlog no tiene historias.")
	})
	t.Run("no project", func(t *testing.T) {
		view := startBacklog("http://localhost:8080").View()
		wantPlainBody(t, view, "Backlog\n\nTodavía no elegiste un proyecto.\n\n▸ Elegir proyecto\n  Volver\n")
		wantStyled(t, view, "Todavía no elegiste un proyecto.")
		wantStyled(t, view, "Backlog")
	})
}

func TestStyledBacklogProjectScreen(t *testing.T) {
	forceColor(t)
	m := press(startBacklog("http://localhost:8080"), tea.KeyEnter) // Elegir proyecto
	m = typeText(m, "p1")
	view := m.View()
	wantPlainBody(t, view, "Proyecto del backlog\n\nID del proyecto: p1_\n\nEnter para continuar, Esc para volver.\n")
	wantStyled(t, view, "Proyecto del backlog")
	footer := rawLine(t, view, "para continuar")
	if strings.Contains(footer, "Enter para continuar") {
		t.Errorf("footer %q is one unstyled run, want the keys styled apart from the text", footer)
	}
	wantStyled(t, view, "para continuar")
}

func TestStyledBacklogFormUsesSelectionStyleAndErrorBanner(t *testing.T) {
	forceColor(t)
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m := press(openBacklogWithProject(t, server.URL, "p1"), tea.KeyEnter) // Crear historia
	view := m.View()

	wantPlainBody(t, view, "Crear historia\n\n"+
		"▸ Título: \n  Descripción: \n  Prioridad: \n  Criterios de aceptación: \n\n"+
		"Tab/↑/↓ para cambiar campo; Enter en el último para enviar; Esc para volver.\n"+
		"Separá los criterios de aceptación con punto y coma.\n")
	wantStyled(t, view, "Crear historia")
	if line := wantStyled(t, view, "▸ Título"); !hasBackground(line) {
		t.Errorf("selected field %q has no background, want the selection style", line)
	}
	if line := wantStyled(t, view, "Descripción:"); hasBackground(line) {
		t.Errorf("unselected field %q has a background, want the item style", line)
	}
	wantStyled(t, view, "Separá los criterios")

	t.Run("local error uses the error banner", func(t *testing.T) {
		edit := selectStory(openBacklogWithProject(t, server.URL, "p1"), 0)
		edit = advanceFields(edit, 5)
		edit = typeText(edit, "abc")
		edit, _ = submitUpdate(edit)
		view := edit.View()
		wantPlainBody(t, view, "Horas estimadas debe ser un número")
		line := wantStyled(t, view, "Horas estimadas debe ser un número")
		if !strings.Contains(line, "│") {
			t.Errorf("error line %q has no banner border, want the error banner style", line)
		}
	})
}

func TestStyledBacklogFailureResultUsesErrorBanner(t *testing.T) {
	forceColor(t)
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m := openBacklogWithProject(t, server.URL, "p1")
	m, _ = apply(m, failure(http.StatusUnprocessableEntity,
		`{"error":"validation_failed","message":"datos inválidos","fields":{"title":"es obligatorio","priority":"valor inválido"}}`))
	view := m.View()

	for _, want := range []string{"Resultado", "La API rechazó los datos (422): datos inválidos", "- priority: valor inválido", "- title: es obligatorio", "Enter/Esc para volver."} {
		wantPlainBody(t, view, want)
	}
	wantStyled(t, view, "Resultado")
	for _, needle := range []string{"La API rechazó los datos (422)", "- priority: valor inválido", "- title: es obligatorio"} {
		line := wantStyled(t, view, needle)
		if !strings.Contains(line, "│") {
			t.Errorf("error line %q has no banner border, want the error banner style", line)
		}
	}
	if !strings.Contains(stripANSI(view), "╭") {
		t.Errorf("stripped View() = %q, want the rounded error banner border", stripANSI(view))
	}
	footer := rawLine(t, view, "para volver")
	if strings.Contains(footer, "Enter/Esc para volver") {
		t.Errorf("footer %q is one unstyled run, want the keys styled apart from the text", footer)
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

	wantPlainBody(t, view, "Resultado\n\nHistoria creada correctamente (201)\n\n"+
		"Historia nueva · prioridad alta · estado pendiente · puntos —\n\nEnter/Esc para volver.\n")
	wantStyled(t, view, "Resultado")
	ok := wantStyled(t, view, "Historia creada correctamente (201)")
	if strings.Contains(ok, "│") {
		t.Errorf("success line %q has a banner border, want only the ok style", ok)
	}
	if strings.Contains(stripANSI(view), "╭") {
		t.Errorf("stripped View() = %q, want no error banner on success", stripANSI(view))
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
