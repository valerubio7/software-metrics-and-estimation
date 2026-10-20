package tui_test

import (
	"encoding/json"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

// openProjectForm opens the create form and leaves the name filled, on field 0.
func openProjectForm(t *testing.T, url string) tui.Model {
	t.Helper()
	return toCreateProject(tui.NewModel(url))
}

// typeDateInto types digits into the focused date field; the segments
// auto-advance, so two-digit month and day values need no navigation.
func typeDateInto(m tui.Model, year, month, day string) tui.Model {
	m = typeText(m, year)
	m = typeText(m, month)
	m = typeText(m, day)
	return m
}

func TestProjectFormFillsDateSegments(t *testing.T) {
	server, bodies := capturingProjectServer(t)
	m := openProjectForm(t, server.URL)
	m = typeText(m, "Alfa")
	m = press(m, tea.KeyTab) // -> fecha inicio
	m = typeDateInto(m, "2026", "10", "16")
	m = press(m, tea.KeyTab) // -> fecha fin
	m = typeDateInto(m, "2026", "12", "31")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("a complete form should issue the create command")
	}
	cmd()

	var sent map[string]string
	if err := json.Unmarshal((*bodies)[0], &sent); err != nil {
		t.Fatal(err)
	}
	if sent["start_date"] != "2026-10-16" || sent["planned_finish_date"] != "2026-12-31" {
		t.Errorf("dates = %q/%q, want 2026-10-16/2026-12-31", sent["start_date"], sent["planned_finish_date"])
	}
}

func TestProjectFormZeroPadsShortSegments(t *testing.T) {
	server, bodies := capturingProjectServer(t)
	m := openProjectForm(t, server.URL)
	m = typeText(m, "Alfa")
	m = press(m, tea.KeyTab) // -> fecha inicio
	m = typeText(m, "2026")  // year auto-advances to month
	m = typeText(m, "1")     // month left short
	m = press(m, tea.KeyRight)
	m = typeText(m, "5") // day left short
	m = press(m, tea.KeyTab)
	m = typeDateInto(m, "2026", "12", "31")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("a complete form should issue the create command")
	}
	cmd()

	var sent map[string]string
	if err := json.Unmarshal((*bodies)[0], &sent); err != nil {
		t.Fatal(err)
	}
	if sent["start_date"] != "2026-01-05" {
		t.Errorf("start_date = %q, want 2026-01-05", sent["start_date"])
	}
}

func TestProjectFormSeparatesSegmentsWithDashes(t *testing.T) {
	server, bodies := capturingProjectServer(t)
	m := openProjectForm(t, server.URL)
	m = typeText(m, "Alfa")
	m = press(m, tea.KeyTab) // -> fecha inicio
	m = typeText(m, "2026-1-5")
	m = press(m, tea.KeyTab)
	m = typeText(m, "2026-12-31")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("a complete form should issue the create command")
	}
	cmd()

	var sent map[string]string
	if err := json.Unmarshal((*bodies)[0], &sent); err != nil {
		t.Fatal(err)
	}
	if sent["start_date"] != "2026-01-05" {
		t.Errorf("start_date = %q, want 2026-01-05", sent["start_date"])
	}
}

func TestProjectFormRejectsIncompleteDateInPlace(t *testing.T) {
	m := openProjectForm(t, "http://localhost:8080")
	m = typeText(m, "Alfa")
	m = press(m, tea.KeyTab) // -> fecha inicio
	m = typeText(m, "2026")  // only the year
	m = press(m, tea.KeyTab) // -> fecha fin

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	if cmd != nil {
		t.Fatal("an incomplete date must not reach the API")
	}
	view := stripANSI(m.View())
	if !strings.Contains(view, "está incompleta") || !strings.Contains(view, "Fecha inicio") {
		t.Errorf("View() = %q, want an in-place error about the start date", view)
	}
}

func TestProjectFormRejectsInvalidDateInPlace(t *testing.T) {
	m := openProjectForm(t, "http://localhost:8080")
	m = typeText(m, "Alfa")
	m = press(m, tea.KeyTab)
	m = typeText(m, "2026") // year
	m = typeText(m, "13")   // month
	m = typeText(m, "40")   // day
	m = press(m, tea.KeyTab)
	m = typeDateInto(m, "2026", "12", "31")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	if cmd != nil {
		t.Fatal("an impossible date must not reach the API")
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "no es una fecha válida") {
		t.Errorf("View() = %q, want an invalid-date error", view)
	}
}

func TestProjectFormRejectsFinishBeforeStart(t *testing.T) {
	m := openProjectForm(t, "http://localhost:8080")
	m = typeText(m, "Alfa")
	m = press(m, tea.KeyTab)
	m = typeDateInto(m, "2026", "10", "16")
	m = press(m, tea.KeyTab)
	m = typeDateInto(m, "2026", "01", "01")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	if cmd != nil {
		t.Fatal("a finish date before the start must not reach the API")
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "igual o posterior") {
		t.Errorf("View() = %q, want an ordering error", view)
	}
}

func TestProjectDateBackspaceStepsSegments(t *testing.T) {
	m := openProjectForm(t, "http://localhost:8080")
	m = press(m, tea.KeyTab) // -> fecha inicio
	m = typeDateInto(m, "2026", "10", "16")

	// Backspacing five times walks day -> month -> year, erasing each in turn.
	for i := 0; i < 5; i++ {
		m = press(m, tea.KeyBackspace)
	}
	view := stripANSI(m.View())
	if !strings.Contains(view, "Fecha inicio") {
		t.Fatalf("View() = %q, want the start date row", view)
	}
	if strings.Contains(view, "2026") {
		t.Errorf("View() = %q, want the year erased after five backspaces", view)
	}
}

func TestProjectFormShowsDateSegmentHints(t *testing.T) {
	m := openProjectForm(t, "http://localhost:8080")
	view := stripANSI(m.View())
	for _, want := range []string{"Fecha inicio", "Fecha fin planificada", "AAAA", "MM", "DD"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() = %q, want date hint %q", view, want)
		}
	}
}

func TestProjectDateRowIsHighlightedWhenFocused(t *testing.T) {
	forceColor(t)
	m := openProjectForm(t, "http://localhost:8080")
	m = press(m, tea.KeyTab) // focus fecha inicio

	line := wantStyled(t, m.View(), "Fecha inicio")
	if !hasBackground(line) {
		t.Errorf("focused date row %q has no background, want the selection bar", line)
	}
}

// openBrowser opens the project browser from the menu and seeds it with projects.
func openBrowser(t *testing.T, projects ...tui.ProjectSummary) tui.Model {
	t.Helper()
	return seedProjects(t, press(tui.NewModel("http://localhost:8080"), tea.KeyEnter), projects...)
}

func TestEditProjectPrefillsFromSelectedSummary(t *testing.T) {
	m := openBrowser(t, tui.ProjectSummary{ID: "p1", Name: "Portal Alfa", StartDate: "2026-03-05", PlannedFinishDate: "2026-09-30", Status: "active"})
	m = typeText(m, "e")
	if m.Screen != "edit-project" {
		t.Fatalf("Screen = %q, want edit-project", m.Screen)
	}
	if got := fieldValue(t, m.View(), "Nombre"); got != "Portal Alfa" {
		t.Errorf("name = %q, want the selected project prefilled", got)
	}
	view := stripANSI(m.View())
	for _, want := range []string{"2026", "03", "05", "09", "30"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() = %q, want the prefilled date segment %q", view, want)
		}
	}
}

func TestProjectResultFailureReturnsToFormWithError(t *testing.T) {
	m := openProjectForm(t, "http://localhost:8080")
	m = typeText(m, "Alfa")
	m = press(m, tea.KeyTab)
	m = typeDateInto(m, "2026", "10", "01")
	m = press(m, tea.KeyTab)
	m = typeDateInto(m, "2026", "12", "01")
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(m, tui.ProjectResultMsg{Response: tui.APIResponse{StatusCode: 422, Body: []byte(`{"error":"validation_failed","message":"fechas inválidas"}`)}})
	if m.Screen != "project-result" {
		t.Fatalf("Screen = %q, want project-result", m.Screen)
	}
	m = press(m, tea.KeyEnter)
	if m.Screen != "create-project" {
		t.Fatalf("Screen = %q, want create-project after a failed request", m.Screen)
	}
	view := stripANSI(m.View())
	if !strings.Contains(view, "422") || !strings.Contains(view, "fechas inválidas") {
		t.Errorf("View() = %q, want the API error kept in the form", view)
	}
	if got := fieldValue(t, m.View(), "Nombre"); got != "Alfa" {
		t.Errorf("name = %q, want the typed value kept", got)
	}
}

func TestProjectResultSuccessReturnsToBrowser(t *testing.T) {
	m := openBrowser(t, tui.ProjectSummary{ID: "p1", Name: "Alpha", StartDate: "2026-01-01", PlannedFinishDate: "2026-12-31", Status: "active"})
	m = typeText(m, "e")
	m = press(m, tea.KeyEnter) // -> fecha inicio
	m = press(m, tea.KeyEnter) // -> fecha fin
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submitting the edit form should issue a command")
	}
	m, _ = apply(m, tui.ProjectResultMsg{Response: tui.APIResponse{StatusCode: 200, Body: []byte(`{"id":"p1","name":"Alpha"}`)}})
	if m.Screen != "project-result" {
		t.Fatalf("Screen = %q, want project-result", m.Screen)
	}
	m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != "project-picker" {
		t.Fatalf("Screen = %q, want project-picker after success", m.Screen)
	}
	if cmd == nil {
		t.Fatal("returning to the browser should reload the collection")
	}
}
