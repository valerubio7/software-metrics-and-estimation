package tui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// returnTarget is where a flow goes back to when it is cancelled from its first
// step or completes. The zero value keeps each flow's own destination.
type returnTarget int

const (
	returnDefault returnTarget = iota
	returnStoryDetail
)

// storyDetailState references the story by ID; the story itself is always read
// from m.backlogStories so the detail never shows stale data.
type storyDetailState struct {
	storyID string
	cursor  int // highlighted action row
}

// storyAction is one action of the detail: its row, its shortcut and its help
// come from this single table, so they cannot drift apart.
type storyAction struct {
	key, label, help string
	open             func(Model, Story) (Model, tea.Cmd)
}

var storyActions = []storyAction{
	{key: "p", label: "Estimar", help: "Estimar la historia con Planning Poker", open: openPokerForStory},
	{key: "t", label: "Registrar esfuerzo", help: "Registrar esfuerzo de la historia", open: openEffortForStory},
	{key: "d", label: "Reportar defecto", help: "Reportar un defecto de la historia", open: openDefectForStory},
}

func openPokerForStory(m Model, story Story) (Model, tea.Cmd) {
	return m.openStoryFlow(sectionPlanningPoker, story)
}

func openEffortForStory(m Model, story Story) (Model, tea.Cmd) {
	return m.openStoryFlow(sectionEffort, story)
}

func openDefectForStory(m Model, story Story) (Model, tea.Cmd) {
	return m.beginDefectCreationForStory(story)
}

// beginDefectCreationForStory opens the defect form with the story of the detail
// already chosen, so the story selector is skipped (the form jumps from its last
// field to the Sprint selector while storyID is set) and the story cannot change.
// The draft always starts empty, and the flow goes back to the detail.
func (m Model) beginDefectCreationForStory(story Story) (Model, tea.Cmd) {
	m, cmd := m.beginDefectCreation()
	item := storySelectorItem(story)
	m.defectCreateForm.storyID, m.defectCreateForm.storyLabel = item.ID, item.Label
	m.returnTo = returnStoryDetail
	return m, cmd
}

// openStoryDetail opens the detail of the story highlighted in the backlog.
// stories is the list as the user sees it (filtered), so the cursor indexes it
// directly. An empty list or an out-of-range cursor does nothing. Opening the
// detail never issues a request: the story is already in the loaded backlog.
func (m Model) openStoryDetail(stories []Story) (Model, tea.Cmd) {
	if m.backlogCursor < 0 || m.backlogCursor >= len(stories) {
		return m, nil
	}
	m.storyDetail = storyDetailState{storyID: stories[m.backlogCursor].ID}
	m.Screen = "story-detail"
	return m, nil
}

// detailStory rereads the story of the detail from the loaded backlog by ID, so
// a reload or an estimate is reflected the next time the detail is drawn.
func (m Model) detailStory() (Story, bool) {
	for _, story := range m.backlogStories {
		if story.ID == m.storyDetail.storyID {
			return story, true
		}
	}
	return Story{}, false
}

// storyActionsAvailable reports whether the detail may offer actions: the backlog
// is not reloading, the reload did not fail and the story is still in it.
func (m Model) storyActionsAvailable() bool {
	if m.backlogLoading || m.backlogMessage != "" {
		return false
	}
	_, found := m.detailStory()
	return found
}

// updateStoryDetail handles the keys of the story detail. Esc goes back to the
// backlog without reloading it, so its filter and cursor are kept. Every other
// key acts only while the detail is ready: during a reload, after a failed one
// or when the story is gone there are no actions to run.
func (m Model) updateStoryDetail(key tea.KeyMsg) (Model, tea.Cmd) {
	if key.Type == tea.KeyEsc {
		m.Screen = "backlog"
		return m, nil
	}
	if !m.storyActionsAvailable() {
		return m, nil
	}
	story, _ := m.detailStory()
	switch key.Type {
	case tea.KeyUp:
		if m.storyDetail.cursor > 0 {
			m.storyDetail.cursor--
		}
	case tea.KeyDown:
		if m.storyDetail.cursor < len(storyActions)-1 {
			m.storyDetail.cursor++
		}
	case tea.KeyEnter:
		return storyActions[m.storyDetail.cursor].open(m, story)
	case tea.KeyRunes:
		for _, action := range storyActions {
			if string(key.Runes) == action.key {
				return action.open(m, story)
			}
		}
	}
	return m, nil
}

// openStoryFlow starts an action of the detail with the story already chosen:
// the flow lives in the "section" shell and begins at its first step, so no
// story selector is shown. Nothing happens unless the detail is ready.
func (m Model) openStoryFlow(kind sectionKind, story Story) (Model, tea.Cmd) {
	if !m.storyActionsAvailable() {
		return m, nil
	}
	m.section = sectionState{kind: kind, selected: storySelectorItem(story)}
	m.effort = effortState{seq: m.effort.seq}
	m.pokerVotes = map[string]string{}
	m.returnTo = returnStoryDetail
	m.Screen = "section"
	return m.beginSectionStep(kind.firstStep())
}

// reloadBacklogInPlace reloads the backlog without leaving the current screen,
// so the detail rereads its story (and shows its loading and error states) when
// something the detail displays has changed. It is the standard reload of the
// backlog: the filter is kept and the cursor goes back to the top.
func (m Model) reloadBacklogInPlace() (Model, tea.Cmd) {
	return m.startBacklogLoad()
}

// backToStoryDetail returns to the detail of the story the flow started from,
// keeping the highlighted action row. It never issues a request.
func (m Model) backToStoryDetail() (Model, tea.Cmd) {
	m.Screen = "story-detail"
	return m, nil
}

// storyDetailBindings lists the keys valid on the detail. Once it is ready the
// footer offers moving, Enter and Esc (the action shortcuts are visible in the
// rows and in the help, so they never push Esc out of the footer); while it
// reloads, shows an error or misses the story only Esc is valid. The global keys
// (?, q, Ctrl+C) are added by bindings().
func (m Model) storyDetailBindings(l *keyList) {
	ready := m.storyActionsAvailable()
	if ready {
		l.add(keyNav, true, "↑/↓", "mover", "Mover la selección")
		l.add(keyAction, true, "Enter", "elegir acción", "Ejecutar la acción resaltada")
	}
	l.add(keyNav, true, "Esc", "backlog", "Volver a la lista del backlog")
	if ready {
		for _, action := range storyActions {
			l.add(keyAction, false, action.key, strings.ToLower(action.label), action.help)
		}
	}
}

func (m Model) viewStoryDetail() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Detalle de la historia"))
	b.WriteString("\n\n")
	// The four states are evaluated in this order: loading, error, missing, ready.
	switch {
	case m.backlogLoading:
		b.WriteString(statusMutedStyle.Render("Cargando backlog…") + "\n")
		return b.String()
	case m.backlogMessage != "":
		b.WriteString(m.errorBlock(m.backlogMessage) + "\n")
		return b.String()
	}
	story, found := m.detailStory()
	if !found {
		b.WriteString(emptyState("◇", "La historia ya no está en el backlog.", "Presione Esc para volver a la lista.") + "\n")
		return b.String()
	}
	points, hours := "", ""
	if story.StoryPoints != nil {
		points = strconv.Itoa(*story.StoryPoints)
	}
	if story.EstimatedHours != nil {
		hours = strconv.FormatFloat(*story.EstimatedHours, 'f', -1, 64)
	}
	for _, field := range []struct{ label, value string }{
		{"Título", story.Title},
		{"Descripción", story.Description},
		{"Prioridad", story.Priority},
		{"Estado", story.Status},
		{"Story points", points},
		{"Criterios de aceptación", strings.Join(story.AcceptanceCriteria, "; ")},
		{"Horas estimadas", hours},
	} {
		b.WriteString(detailField(field.label, field.value) + "\n")
	}
	b.WriteString("\n" + panelTitleStyle.Render("Acciones") + "\n")
	for i, action := range storyActions {
		b.WriteString(m.row(i == m.storyDetail.cursor, action.key+"  "+action.label) + "\n")
	}
	return b.String()
}

// detailField renders one "label: value" line, with a dash for an empty value.
func detailField(label, value string) string {
	if strings.TrimSpace(value) == "" {
		value = "—"
	}
	return hintStyle.Render(label+": ") + value
}
