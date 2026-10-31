package tui_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

var effortFormDate = regexp.MustCompile(`Fecha:\s+(\d{4}|AAAA)\s+(\d{2}|MM)\s+(\d{2}|DD)`)

func TestEffortFormOpensWithUTCDateFromInjectedClock(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)

	if m.Screen != "effort-form" {
		t.Fatalf("Screen = %q, want effort-form", m.Screen)
	}
	view := stripANSI(m.View())
	date := effortFormDate.FindStringSubmatch(view)
	if date == nil || date[1] != "2026" || date[2] != "11" || date[3] != "01" {
		t.Fatalf("the date should open as the UTC day 2026-11-01, got %v in %q", date, view)
	}
	if line := focusedLine(view); !strings.Contains(line, "Fecha") {
		t.Fatalf("the focus should start on Fecha: %q", line)
	}
	for _, label := range []string{"Actividad", "Horas"} {
		if value := fieldValue(t, view, label); value != "" {
			t.Errorf("%s should start empty, got %q", label, value)
		}
	}
	for _, want := range []string{"Historia: Primera historia", "Tarea: Diseñar", "Integrante: Ada Lovelace"} {
		if !strings.Contains(view, want) {
			t.Errorf("the form should show the read-only %q: %q", want, view)
		}
	}
	for _, request := range rec.all() {
		if request.Method != http.MethodGet || strings.Contains(request.Path, "effort-records") {
			t.Errorf("opening the form emitted %s %s, want only navigation GETs", request.Method, request.Path)
		}
	}
}

func TestEffortFormShowsWholeStoryLabel(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 0, 0)

	if view := stripANSI(m.View()); m.Screen != "effort-form" || !strings.Contains(view, "Tarea: toda la historia") {
		t.Fatalf("the whole-story choice should read 'Tarea: toda la historia' (screen %q): %q", m.Screen, view)
	}
}

func TestEffortMemberSingleSelectionOpensFormForSecondMember(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortMembers(t, server.URL, 0, 1).WithClock(fixedClock)

	m = press(m, tea.KeyDown)
	m, _ = apply(m, spaceKey())
	if view := stripANSI(m.View()); strings.Contains(view, "[x]") || strings.Contains(view, "[ ]") {
		t.Fatalf("Space must not mark a member: %q", view)
	}
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if view := stripANSI(m.View()); m.Screen != "effort-form" || !strings.Contains(view, "Integrante: Grace Hopper") {
		t.Fatalf("Enter on the second member should open the form for Grace Hopper (screen %q): %q", m.Screen, view)
	}
}

// effortDate reads the three rendered date segments (hints included).
func effortDate(t *testing.T, view string) [3]string {
	t.Helper()
	match := effortFormDate.FindStringSubmatch(view)
	if match == nil {
		t.Fatalf("the form shows no date row: %q", view)
	}
	return [3]string{match[1], match[2], match[3]}
}

// assertFocus fails unless the focused row is the one labelled label.
func assertFocus(t *testing.T, m interface{ View() string }, label string) {
	t.Helper()
	if line := focusedLine(stripANSI(m.View())); !strings.Contains(line, label+":") {
		t.Fatalf("focus should be on %s, got %q", label, line)
	}
}

func TestEffortFormEditsDateBySegments(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)

	m = press(m, tea.KeyRight)
	m = press(m, tea.KeyRight) // day segment
	for i := 0; i < 7; i++ {   // "1" + "11" + "2026" digits, stepping back through the segments
		m = press(m, tea.KeyBackspace)
	}
	if got := effortDate(t, stripANSI(m.View())); got != [3]string{"AAAA", "MM", "DD"} {
		t.Fatalf("backspace should erase the whole date, got %v", got)
	}

	m = typeText(m, "2026")
	m = typeText(m, "1")
	m = press(m, tea.KeyBackspace) // lands on the month only if the focus advanced after the year
	if got := effortDate(t, stripANSI(m.View())); got != [3]string{"2026", "MM", "DD"} {
		t.Fatalf("the focus should have moved to the month after the year was complete, got %v", got)
	}

	m = typeText(m, "1005")
	if got := effortDate(t, stripANSI(m.View())); got != [3]string{"2026", "10", "05"} {
		t.Fatalf("typing 20261005 should fill year, month and day, got %v", got)
	}
}

func TestEffortFormEnterAdvancesWithoutRequest(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)
	before := len(rec.all())

	assertFocus(t, m, "Fecha")
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("Enter on Fecha must not emit a command")
	}
	assertFocus(t, m, "Actividad")
	m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("Enter on Actividad must not emit a command")
	}
	assertFocus(t, m, "Horas")
	if after := len(rec.all()); after != before {
		t.Fatalf("Enter emitted %d requests, want none", after-before)
	}
}

func TestEffortFormTabAndArrowsWrapAround(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)

	steps := []struct {
		name string
		key  tea.KeyType
		want string
	}{
		{"Tab", tea.KeyTab, "Actividad"},
		{"Tab", tea.KeyTab, "Horas"},
		{"Tab wraps to the first field", tea.KeyTab, "Fecha"},
		{"Shift+Tab wraps to the last field", tea.KeyShiftTab, "Horas"},
		{"Up", tea.KeyUp, "Actividad"},
		{"Down", tea.KeyDown, "Horas"},
		{"Down wraps to the first field", tea.KeyDown, "Fecha"},
		{"Up wraps to the last field", tea.KeyUp, "Horas"},
	}
	for _, step := range steps {
		m = press(m, step.key)
		if line := focusedLine(stripANSI(m.View())); !strings.Contains(line, step.want+":") {
			t.Fatalf("after %s the focus should be on %s, got %q", step.name, step.want, line)
		}
	}
}

func TestEffortFormTextFieldsAcceptSpacesAndRuneBackspace(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)

	m = press(m, tea.KeyTab) // Actividad
	m = typeText(m, "Revisión de diseño")
	if got := fieldValue(t, stripANSI(m.View()), "Actividad"); got != "Revisión de diseño" {
		t.Fatalf("Actividad = %q, want the typed text with its spaces", got)
	}
	m = press(m, tea.KeyBackspace)
	got := fieldValue(t, stripANSI(m.View()), "Actividad")
	if got != "Revisión de diseñ" || !utf8.ValidString(got) {
		t.Fatalf("Actividad = %q, want 'Revisión de diseñ' as valid UTF-8", got)
	}
}

func TestEffortFormTreatsGlobalKeysAsTextCharacters(t *testing.T) {
	for _, field := range []struct {
		label string
		tabs  int
	}{{"Actividad", 1}, {"Horas", 2}} {
		t.Run(field.label, func(t *testing.T) {
			server, rec := effortFlowServer(t, effortFlowOptions{})
			m := openEffortFormFor(t, server.URL, 0, 1, 0)
			for i := 0; i < field.tabs; i++ {
				m = press(m, tea.KeyTab)
			}
			before := len(rec.all())

			for _, r := range "qQ?pr" {
				var cmd tea.Cmd
				m, cmd = apply(m, keyMsg(string(r)))
				if isQuit(cmd) {
					t.Fatalf("%q must not quit the TUI inside the form", r)
				}
			}
			view := stripANSI(m.View())
			if got := fieldValue(t, view, field.label); got != "qQ?pr" {
				t.Fatalf("%s = %q, want qQ?pr", field.label, got)
			}
			if m.Screen != "effort-form" || strings.Contains(view, "Atajos de teclado") {
				t.Fatalf("the form must stay open without help (screen %q): %q", m.Screen, view)
			}
			if after := len(rec.all()); after != before {
				t.Fatalf("typing emitted %d requests, want none", after-before)
			}
			if _, cmd := apply(m, tea.KeyMsg{Type: tea.KeyCtrlC}); !isQuit(cmd) {
				t.Fatal("Ctrl+C must still quit")
			}
		})
	}
}

func TestEffortFormFooterOnlyOffersFormKeys(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)

	view := strings.Join(strings.Fields(stripANSI(m.View())), " ")
	for _, want := range []string{"Tab/↑/↓ campo", "Enter siguiente/enviar", "Esc volver"} {
		if !strings.Contains(view, want) {
			t.Errorf("footer lacks %q: %q", want, view)
		}
	}
	for _, banned := range []string{"q salir", "p proyecto", "r refrescar", "? ayuda"} {
		if strings.Contains(view, banned) {
			t.Errorf("footer must not offer %q: %q", banned, view)
		}
	}
}

func TestEffortFormCrumbAndTitle(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)

	view := stripANSI(m.View())
	for _, want := range []string{"Esfuerzo › Registrar", "Proyecto: Alpha", "Registrar esfuerzo"} {
		if !strings.Contains(view, want) {
			t.Errorf("form lacks %q: %q", want, view)
		}
	}
}

// valueColumn returns the display column where the value after "label:" starts.
func valueColumn(t *testing.T, view, label string) int {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if before, after, found := strings.Cut(line, label+":"); found {
			rest := strings.TrimLeft(after, " ")
			return utf8.RuneCountInString(before) + utf8.RuneCountInString(label+":") + utf8.RuneCountInString(after) - utf8.RuneCountInString(rest)
		}
	}
	t.Fatalf("label %q not found in view %q", label, view)
	return 0
}

func TestEffortFormAlignsTheValueColumn(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)
	m = press(m, tea.KeyTab)
	m = typeText(m, "x")
	m = press(m, tea.KeyTab)
	m = typeText(m, "1")

	view := stripANSI(m.View())
	activity, hours, date := valueColumn(t, view, "Actividad"), valueColumn(t, view, "Horas"), valueColumn(t, view, "Fecha")
	if activity != hours || hours+1 != date {
		t.Fatalf("value columns differ: Fecha %d (segments start one cell later), Actividad %d, Horas %d: %q", date, activity, hours, view)
	}
}
