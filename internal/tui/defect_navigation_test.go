package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDefectMutationLoadingBlocksProjectSwitchAndQuit(t *testing.T) {
	for _, screen := range []string{"defect-create-confirm", "defect-status-select", "defect-resolution-select"} {
		t.Run(screen, func(t *testing.T) {
			m := NewModel("http://localhost:8080")
			m.Screen, m.defectCreateLoading, m.defectStatusLoading = screen, true, true
			for _, key := range []string{"q", "Q", "p"} {
				updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
				if updated.(Model).Screen != screen || isQuitCommand(cmd) {
					t.Errorf("%q escaped in-flight %s: screen=%q quit=%v", key, screen, updated.(Model).Screen, isQuitCommand(cmd))
				}
			}
			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
			if !updated.(Model).showHelp {
				t.Error("? should remain available during the request")
			}
		})
	}
}

func TestDefectStatusScreensSupportGlobalNavigationWhenIdle(t *testing.T) {
	for _, screen := range []string{"defect-status-select", "defect-resolution-select"} {
		t.Run(screen, func(t *testing.T) {
			m := NewModel("http://localhost:8080")
			m.Screen, m.activeProjectID = screen, "p-1"
			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Q")})
			if !isQuitCommand(cmd) {
				t.Fatal("Q should quit from an idle defect status screen")
			}
			if updated.(Model).Screen != screen {
				t.Fatalf("Q changed screen to %q", updated.(Model).Screen)
			}

			m = NewModel("http://localhost:8080")
			m.Screen, m.activeProjectID = screen, "p-1"
			updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
			if updated.(Model).Screen != screen || cmd != nil {
				t.Fatalf("p = screen %q, cmd nil %v; the removed project shortcut must not leave the screen", updated.(Model).Screen, cmd == nil)
			}

			m = NewModel("http://localhost:8080")
			m.Screen = screen
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
			if !updated.(Model).showHelp {
				t.Fatal("? should open help from a defect status screen")
			}
			updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
			if !isQuitCommand(cmd) {
				t.Fatal("Ctrl+C should quit from a defect status screen")
			}
		})
	}
}

func TestDefectFlowTextAndFilterOwnershipSuppressGlobalShortcuts(t *testing.T) {
	tests := []struct {
		name   string
		screen string
		setup  func(*Model)
	}{
		{name: "description form", screen: "defect-create-form", setup: func(m *Model) { m.defectCreateForm.field = 0 }},
		{name: "story filtering", screen: "defect-story-select", setup: func(m *Model) { m.defectStorySelector.filtering = true }},
		{name: "detection Sprint filtering", screen: "defect-sprint-select", setup: func(m *Model) { m.defectSprintSelector.filtering = true }},
		{name: "resolution Sprint filtering", screen: "defect-resolution-select", setup: func(m *Model) { m.defectResolutionSelector.filtering = true }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{"q", "p", "?"} {
				m := NewModel("http://localhost:8080")
				m.Screen = tc.screen
				tc.setup(&m)
				updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
				got := updated.(Model)
				if isQuitCommand(cmd) || got.Screen != tc.screen || got.showHelp {
					t.Errorf("%q escaped %s: screen=%q help=%v quit=%v", key, tc.name, got.Screen, got.showHelp, isQuitCommand(cmd))
				}
				if key == "q" && tc.screen == "defect-create-form" && !strings.Contains(got.defectCreateForm.description, "q") {
					t.Error("q was not retained by the description editor")
				}
			}
		})
	}
}

func isQuitCommand(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}
