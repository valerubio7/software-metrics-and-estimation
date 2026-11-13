package tui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestRootTabsExplainTheirPurpose checks the plain-language block every root
// tab shows, so a user without domain knowledge can read what the section is.
func TestRootTabsExplainTheirPurpose(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := openWorkspace(t, server.URL, "p1")
	m, _ = apply(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	cases := []struct{ name, tab, what, can string }{
		{"dashboard", tabDashboard, "el avance del proyecto", "cambiar el ámbito"},
		{"backlog", tabBacklog, "la lista priorizada", "abrir una historia"},
		{"sprints", tabSprints, "un período de trabajo", "asignarle historias"},
		{"defects", tabDefects, "un problema detectado", "cambiar el estado"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.tab != tabDashboard {
				var cmd tea.Cmd
				m, cmd = apply(m, keyMsg(tc.tab))
				if cmd == nil {
					t.Fatalf("opening %s should issue its load command", tc.name)
				}
				m = run(t, m, cmd)
			}
			view := stripANSI(m.View())
			if !strings.Contains(view, "Qué es:") || !strings.Contains(view, tc.what) {
				t.Errorf("%s lacks its Qué es definition %q: %q", tc.name, tc.what, view)
			}
			if !strings.Contains(view, "Qué podés hacer:") || !strings.Contains(view, tc.can) {
				t.Errorf("%s lacks its Qué podés hacer hint %q: %q", tc.name, tc.can, view)
			}
		})
	}
}

// TestSectionIntroAppearsOnceInWideLayout guards the master-detail composition:
// the block belongs above both panes, not repeated inside the list.
func TestSectionIntroAppearsOnceInWideLayout(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m, cmd := openTab(t, server.URL, "p1", tabBacklog)
	m = run(t, m, cmd)
	m, _ = apply(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	view := stripANSI(m.View())
	if got := strings.Count(view, "Qué es:"); got != 1 {
		t.Errorf("wide Backlog shows Qué es %d times, want 1: %q", got, view)
	}
	if got := strings.Count(view, "Qué podés hacer:"); got != 1 {
		t.Errorf("wide Backlog shows Qué podés hacer %d times, want 1: %q", got, view)
	}
}

// TestSectionIntroStaysOutOfForms keeps the orientation block on root screens:
// a flow uses that height for its own content.
func TestSectionIntroStaysOutOfForms(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m, cmd := openTab(t, server.URL, "p1", tabBacklog)
	m = run(t, m, cmd)
	m, _ = apply(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = apply(m, keyMsg("n"))
	view := stripANSI(m.View())
	if strings.Contains(view, "Qué es:") || strings.Contains(view, "Qué podés hacer:") {
		t.Errorf("story form repeats the section intro: %q", view)
	}
}
