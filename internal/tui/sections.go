package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// sectionKind identifies one of the Sprint 3 navigation sections.
type sectionKind string

const (
	sectionPlanningPoker sectionKind = "planning-poker"
	sectionEffort        sectionKind = "effort"
	sectionDefects       sectionKind = "defects"
	sectionMetrics       sectionKind = "metrics"
	sectionDashboard     sectionKind = "dashboard"
)

// title returns the user-facing section name.
func (k sectionKind) title() string {
	switch k {
	case sectionPlanningPoker:
		return "Planning Poker"
	case sectionEffort:
		return "Esfuerzo"
	case sectionDefects:
		return "Defectos"
	case sectionMetrics:
		return "Métricas"
	case sectionDashboard:
		return "Dashboard"
	default:
		return string(k)
	}
}

// sectionStep is the selector currently active inside a section.
type sectionStep int

const (
	sectionStepPrimary sectionStep = iota
	sectionStepMembers
	sectionStepTasks
	sectionStepSprints
	sectionStepNotice
)

// sectionState is the navigation state of a Sprint 3 section. The primary
// selector is preserved while a secondary selector is displayed, so Esc can
// restore the previous list without another request.
type sectionState struct {
	kind     sectionKind
	step     sectionStep
	selector selector
	primary  selector
	selected selectorItem
	detail   selectorItem
	notice   string
}

// SectionResultMsg is the asynchronous result of a section API read.
type SectionResultMsg struct {
	Kind     sectionKind
	Step     sectionStep
	Response APIResponse
	Err      error
}

func (m Model) openSection(kind sectionKind) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) loadSectionStepCmd() tea.Cmd {
	return nil
}

func (m Model) applySectionResult(result SectionResultMsg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) updateSection(key tea.KeyMsg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) viewSection() string {
	return ""
}

func (m Model) viewSelector(s *selector, title, hint, emptyTitle, emptyHint string) string {
	return ""
}

func (m Model) sectionFooter() string {
	return ""
}

func (m Model) sectionHelpRows() [][2]string {
	return nil
}
