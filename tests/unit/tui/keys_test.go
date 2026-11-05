package tui_test

import (
	"net/http"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

// openSprintList opens the Sprints tab of p1 with one open Sprint on a wide
// terminal, so the footer stays on a single line.
func openSprintList(t *testing.T) tui.Model {
	t.Helper()
	server := sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path == "/projects/p1/sprints/open" {
			_, _ = w.Write([]byte(`{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Ship","is_closed":false}]}`))
			return true
		}
		return false
	})
	m := openSprintsWithProject(t, server.URL, "p1")
	m, _ = apply(m, tea.WindowSizeMsg{Width: 200, Height: 40})
	return m
}

// footerPairs returns the shortcut pairs of the footer, the last line of the
// stripped view on a wide terminal.
func footerPairs(m tui.Model) []string {
	lines := strings.Split(stripANSI(m.View()), "\n")
	return strings.Split(strings.TrimSpace(lines[len(lines)-1]), "  ·  ")
}

func hasPair(pairs []string, key, label string) bool {
	for _, pair := range pairs {
		if strings.Join(strings.Fields(pair), " ") == key+" "+label {
			return true
		}
	}
	return false
}

func TestSprintListFooterShowsPrimaryShortcutsAndMore(t *testing.T) {
	pairs := footerPairs(openSprintList(t))
	if !hasPair(pairs, "?", "más") {
		t.Fatalf("footer %q lacks ? más", pairs)
	}
	if shortcuts := len(pairs) - 1; shortcuts > 8 {
		t.Errorf("footer shows %d shortcuts, want at most 8: %q", shortcuts, pairs)
	}
	for _, want := range [][2]string{{"↑/↓", "mover"}, {"Enter", "historias"}, {"n", "nuevo"}, {"/", "buscar"}, {"Esc", "proyectos"}, {"a", "asignar"}, {"x", "cerrar"}, {"h", "cerrados"}} {
		if !hasPair(pairs, want[0], want[1]) {
			t.Errorf("footer %q lacks %s %s", pairs, want[0], want[1])
		}
	}
}

func TestSprintListHelpGroupsEveryShortcut(t *testing.T) {
	m, _ := apply(openSprintList(t), keyMsg("?"))
	help := stripANSI(m.View())
	nav, actions, general := strings.Index(help, "Navegación"), strings.Index(help, "Acciones"), strings.Index(help, "General")
	if nav < 0 || actions < nav || general < actions {
		t.Fatalf("help lacks the ordered groups Navegación, Acciones, General: %q", help)
	}
	for _, want := range []string{"Cerrar el Sprint resaltado", "Ver los Sprints cerrados", "Descargar el reporte PDF del Sprint resaltado", "Asignar historias"} {
		if !strings.Contains(help, want) {
			t.Errorf("help lacks %q: %q", want, help)
		}
	}
	for _, key := range []string{"x", "h", "d"} {
		if !strings.Contains(help, "│   "+key+"   ") {
			t.Errorf("help lacks the key %q: %q", key, help)
		}
	}
}

func TestRootHelpListsOneToFourAndTabKeys(t *testing.T) {
	m, _ := apply(openSprintList(t), keyMsg("?"))
	help := stripANSI(m.View())
	for _, want := range []string{"Tab/Shift+Tab", "1–4"} {
		if !strings.Contains(help, want) {
			t.Errorf("tab root help lacks %q: %q", want, help)
		}
	}
	if strings.Contains(help, "1–6") {
		t.Errorf("tab root help still lists the six tabs: %q", help)
	}
}

func TestScreenWithFewShortcutsHasNoMore(t *testing.T) {
	server := sprintServer(t, func(http.ResponseWriter, *http.Request) bool { return false })
	m := openSprintsWithProject(t, server.URL, "p1")
	m, _ = apply(m, tea.WindowSizeMsg{Width: 200, Height: 40})
	m, _ = apply(m, keyMsg("n")) // Sprint goal form: Enter, Esc, Ctrl+C
	if m.Screen != "sprint-goal" {
		t.Fatalf("Screen = %q, want sprint-goal", m.Screen)
	}
	pairs := footerPairs(m)
	if hasPair(pairs, "?", "más") {
		t.Errorf("footer %q advertises ? más with every shortcut visible", pairs)
	}
	if !hasPair(pairs, "Enter", "enviar") || !hasPair(pairs, "Esc", "volver") {
		t.Errorf("footer %q lacks the form shortcuts", pairs)
	}
}

func TestFilterAndTextFieldFootersDoNotAdvertiseHelp(t *testing.T) {
	m, _ := apply(openSprintList(t), keyMsg("/"))
	if pairs := footerPairs(m); strings.Contains(strings.Join(pairs, "|"), "?") {
		t.Errorf("filter footer %q advertises ?", pairs)
	}
	m = press(m, tea.KeyEsc)
	m, _ = apply(m, keyMsg("n"))
	if pairs := footerPairs(m); strings.Contains(strings.Join(pairs, "|"), "?") {
		t.Errorf("text field footer %q advertises ?", pairs)
	}
}

func TestHelpShowsConceptsBeforeShortcuts(t *testing.T) {
	m, _ := apply(openSprintList(t), keyMsg("?"))
	help := stripANSI(m.View())
	concepts, shortcuts := strings.Index(help, "Conceptos"), strings.Index(help, "Atajos de teclado")
	if concepts < 0 || shortcuts < 0 || concepts > shortcuts {
		t.Fatalf("help must show Conceptos before Atajos de teclado: %q", help)
	}
	for _, want := range []string{"Proyecto", "Backlog", "Historia", "Sprint", "Tarea", "Story Points", "Esfuerzo", "Defecto", "Velocidad", "Desviación"} {
		if !strings.Contains(help, want) {
			t.Errorf("help lacks the concept %q: %q", want, help)
		}
	}
}

func TestHelpScrollsOnShortTerminal(t *testing.T) {
	m, _ := apply(openSprintList(t), keyMsg("?"))
	m, _ = apply(m, tea.WindowSizeMsg{Width: 120, Height: 18})
	if help := stripANSI(m.View()); strings.Contains(help, "Desviación") {
		t.Fatalf("short help should start at the top, before Desviación: %q", help)
	}
	m = press(m, tea.KeyPgDown)
	help := stripANSI(m.View())
	if !strings.Contains(help, "Desviación") {
		t.Errorf("PgDn did not reveal the end of the concepts: %q", help)
	}
	if !strings.Contains(help, "↑/↓") || !strings.Contains(help, "mover") {
		t.Errorf("help footer lacks the scroll keys: %q", help)
	}
}
