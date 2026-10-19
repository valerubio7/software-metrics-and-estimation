package tui_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

func TestHelpOverlayToggles(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")
	m, _ = apply(m, keyMsg("?"))
	view := m.View()
	if !strings.Contains(view, "Atajos de teclado") {
		t.Fatalf("View() = %q, want the help overlay", view)
	}
	for _, want := range []string{"Mover la selección", "Cambiar el proyecto activo", "Salir"} {
		if !strings.Contains(view, want) {
			t.Errorf("help overlay missing %q: %q", want, view)
		}
	}
	m = press(m, tea.KeyEsc)
	if strings.Contains(m.View(), "Atajos de teclado") {
		t.Errorf("View() = %q, want the help overlay closed", m.View())
	}
}

func TestQuestionMarkIsTypedInTextFields(t *testing.T) {
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m := typeText(openBacklogWithProject(t, server.URL, "p1"), "n") // create form
	m = typeText(m, "?")
	if strings.Contains(m.View(), "Atajos de teclado") {
		t.Fatalf("View() = %q, want no help overlay inside a text field", m.View())
	}
	if got := fieldValue(t, m.View(), "Título"); got != "?" {
		t.Errorf("title field = %q, want %q", got, "?")
	}
}

func TestProjectShortcutOpensPicker(t *testing.T) {
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m := openBacklogWithProject(t, server.URL, "p1")
	m, cmd := apply(m, keyMsg("p"))
	if m.Screen != "project-picker" {
		t.Fatalf("Screen = %q, want project-picker", m.Screen)
	}
	if cmd == nil {
		t.Fatal("opening the picker should load the project collection")
	}
}

func TestActiveProjectContextIsShownAcrossScreens(t *testing.T) {
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m := openBacklogWithProject(t, server.URL, "p1")
	if !strings.Contains(m.View(), "Proyecto: Alpha") {
		t.Errorf("backlog View() = %q, want the active project in the breadcrumb", m.View())
	}
	m = press(m, tea.KeyEsc)
	if !strings.Contains(m.View(), "Proyecto activo: Alpha") {
		t.Errorf("menu View() = %q, want the active project on the panel", m.View())
	}
}

func TestProjectClientListUsesCollectionEndpoint(t *testing.T) {
	server, rec := recordingServer(t, http.StatusOK, `{"projects":[]}`)
	response, err := tui.NewProjectClient(server.URL).List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := onlyRequest(t, rec)
	if got.Method != http.MethodGet || got.Path != "/projects" || len(got.Body) != 0 {
		t.Errorf("request = %+v, want GET /projects without body", got)
	}
	if response.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", response.StatusCode)
	}
}

func TestPickerShowsTransportErrorWithoutCrashing(t *testing.T) {
	m, _ := startBacklog("http://localhost:8080")
	m, _ = apply(m, tui.ProjectsResultMsg{Err: errors.New("connection refused")})
	if m.Screen != "project-picker" {
		t.Fatalf("Screen = %q, want project-picker", m.Screen)
	}
	if view := m.View(); !strings.Contains(view, "Error de conexión") || !strings.Contains(view, "connection refused") {
		t.Errorf("View() = %q, want the transport error banner", view)
	}
}
