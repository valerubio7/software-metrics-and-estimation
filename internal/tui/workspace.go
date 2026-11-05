package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// workspaceTab is one of the sections of an open project, shown as a tab bar.
type workspaceTab int

const (
	tabNone workspaceTab = iota
	tabDashboard
	tabBacklog
	tabSprints
	tabDefects
)

// workspaceTabs is the tab bar order; the digit shortcut of a tab is its
// position plus one.
var workspaceTabs = []workspaceTab{tabDashboard, tabBacklog, tabSprints, tabDefects}

func (t workspaceTab) title() string {
	switch t {
	case tabDashboard:
		return "Dashboard"
	case tabBacklog:
		return "Backlog"
	case tabSprints:
		return "Sprints"
	case tabDefects:
		return "Defectos"
	default:
		return ""
	}
}

// openTab enters the root screen of tab for the open project.
func (m Model) openTab(t workspaceTab) (Model, tea.Cmd) {
	switch t {
	case tabBacklog:
		return m.openBacklog()
	case tabSprints:
		return m.enterSprints()
	case tabDefects:
		return m.openDefects()
	default:
		return m.openDashboard()
	}
}

// currentTab derives the tab that owns the active screen, or tabNone outside
// the workspace.
func (m Model) currentTab() workspaceTab {
	switch {
	case m.Screen == "dashboard" || m.Screen == "dashboard-sprint-select":
		return tabDashboard
	case strings.HasPrefix(m.Screen, "backlog"):
		return tabBacklog
	case m.Screen == "story-detail":
		return m.storyDetail.tab()
	case isSprintScreen(m.Screen):
		return tabSprints
	case m.inStoryFlow():
		// A flow keeps the tab of the detail it started from: Backlog or Sprints.
		return m.storyDetail.tab()
	case m.Screen == "defects" || strings.HasPrefix(m.Screen, "defect-"):
		return tabDefects
	}
	return tabNone
}

// inStoryFlow reports whether the active screen is a step of an action launched
// from the story detail (Estimar, Registrar esfuerzo or Reportar defecto).
func (m Model) inStoryFlow() bool {
	switch m.Screen {
	case "section", "planning-poker", "effort-form",
		"defect-create-form", "defect-sprint-select", "defect-create-confirm":
		return true
	}
	return false
}

// tab returns the workspace tab that owns the detail, based on where it was
// opened from.
func (s storyDetailState) tab() workspaceTab {
	if s.origin == storyDetailFromSprint {
		return tabSprints
	}
	return tabBacklog
}

// tabRoot reports whether the active screen is the first screen of its tab,
// where the tab shortcuts apply.
func (m Model) tabRoot() bool {
	switch m.Screen {
	case "dashboard", "backlog", "sprints", "defects":
		return true
	}
	return false
}

// tabForKey maps Tab, Shift+Tab and the digits 1–4 to the target tab.
func tabForKey(key tea.KeyMsg, current workspaceTab) (workspaceTab, bool) {
	index := 0
	for i, t := range workspaceTabs {
		if t == current {
			index = i
		}
	}
	switch key.Type {
	case tea.KeyTab:
		return workspaceTabs[(index+1)%len(workspaceTabs)], true
	case tea.KeyShiftTab:
		return workspaceTabs[(index+len(workspaceTabs)-1)%len(workspaceTabs)], true
	case tea.KeyRunes:
		if len(key.Runes) == 1 && key.Runes[0] >= '1' && int(key.Runes[0]-'0') <= len(workspaceTabs) {
			return workspaceTabs[key.Runes[0]-'1'], true
		}
	}
	return tabNone, false
}

// leaveWorkspace returns to the project browser, the parent of every tab.
func (m Model) leaveWorkspace() (Model, tea.Cmd) {
	return m.openProjectPicker()
}

// inWorkspace reports whether the chrome must show the project and its tabs.
func (m Model) inWorkspace() bool {
	return m.activeProjectID != "" && m.currentTab() != tabNone
}

// workspaceHeader renders the open project with its status and the tab bar,
// highlighting the active tab. Narrow terminals drop the digit labels first.
func (m Model) workspaceHeader() string {
	width := m.contentWidth()
	project := titleStyle.Render("◆ " + m.projectLabel())
	if m.activeProjectStatus != "" {
		project += "  " + projectStatusBadge(m.activeProjectStatus)
	}
	current := m.currentTab()
	render := func(numbered bool) string {
		parts := make([]string, 0, len(workspaceTabs))
		for i, t := range workspaceTabs {
			label := t.title()
			if numbered {
				label = fmt.Sprintf("%d %s", i+1, label)
			}
			if t == current {
				parts = append(parts, tabActiveStyle.Render(" "+label+" "))
			} else {
				parts = append(parts, tabIdleStyle.Render(" "+label+" "))
			}
		}
		return strings.Join(parts, "")
	}
	tabs := render(true)
	if lipgloss.Width(tabs) > width {
		tabs = render(false)
	}
	return ansi.Truncate(project, width, "…") + "\n" + ansi.Truncate(tabs, width, "…")
}
