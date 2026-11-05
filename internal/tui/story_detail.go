package tui

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// storyDetailOrigin is the list the detail was opened from. It decides where
// Esc returns, which tab stays highlighted and whether the detail also shows
// the tasks of the Sprint.
type storyDetailOrigin int

const (
	storyDetailFromBacklog storyDetailOrigin = iota
	storyDetailFromSprint
)

// storyDetailState references the story by ID; the story itself is always read
// from the list it came from, so the detail never shows stale data.
type storyDetailState struct {
	storyID string
	cursor  int // highlighted action row
	origin  storyDetailOrigin

	// The tasks of the story are only loaded when the detail is opened from a
	// Sprint: the Backlog detail stays a pure read of the backlog collection.
	tasks        []Task
	tasksLoading bool
	tasksMessage string
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

var sprintStoryActions = []storyAction{
	{key: "n", label: "Agregar tarea", help: "Agregar una tarea a la historia del Sprint", open: openTaskForStory},
	{key: "c", label: "Completar historia", help: "Marcar la historia del Sprint como completada", open: completeStoryForStory},
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

func openTaskForStory(m Model, story Story) (Model, tea.Cmd) {
	m.taskStoryID = story.ID
	m.taskTitle, m.taskHours, m.taskField, m.Screen = "", "", 0, "sprint-task-form"
	m.sprintMessage = ""
	return m, nil
}

func completeStoryForStory(m Model, story Story) (Model, tea.Cmd) {
	return m.completeStory(story)
}

// storyDetailActions returns the actions of the detail in display order. The
// Sprint context adds the two execution actions after the shared ones.
func (m Model) storyDetailActions() []storyAction {
	actions := make([]storyAction, 0, len(storyActions)+len(sprintStoryActions))
	actions = append(actions, storyActions...)
	if m.storyDetail.origin == storyDetailFromSprint {
		actions = append(actions, sprintStoryActions...)
	}
	return actions
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

// openSprintStoryDetail opens the detail of the story highlighted in a Sprint.
// The detail reuses the loaded Sprint stories and queries only the tasks of
// that story, because the Sprint context shows and creates them.
func (m Model) openSprintStoryDetail(stories []Story) (Model, tea.Cmd) {
	if m.sprintCursor < 0 || m.sprintCursor >= len(stories) {
		return m, nil
	}
	story := stories[m.sprintCursor]
	m.storyDetail = storyDetailState{
		storyID:      story.ID,
		origin:       storyDetailFromSprint,
		tasksLoading: true,
	}
	m.sprintMessage = ""
	m.Screen = "story-detail"
	return m, m.loadStoryTasksCmd(story.ID)
}

// detailStory rereads the story of the detail from the list it came from by ID,
// so a reload is reflected the next time the detail is drawn.
func (m Model) detailStory() (Story, bool) {
	stories := m.backlogStories
	if m.storyDetail.origin == storyDetailFromSprint {
		stories = m.sprintStories
	}
	for _, story := range stories {
		if story.ID == m.storyDetail.storyID {
			return story, true
		}
	}
	return Story{}, false
}

// storyActionsAvailable reports whether the detail may offer actions: its
// source list is not reloading, the reload did not fail and the story is still
// in it.
func (m Model) storyActionsAvailable() bool {
	if m.storyDetail.origin == storyDetailFromSprint {
		if m.sprintStoriesLoading || m.sprintMessage != "" {
			return false
		}
	} else if m.backlogLoading || m.backlogMessage != "" {
		return false
	}
	_, found := m.detailStory()
	return found
}

// updateStoryDetail handles the keys of the story detail. Esc goes back to the
// list it came from without reloading it, so its filter and cursor are kept.
// Every other key acts only while the detail is ready: during a reload, after a
// failed one or when the story is gone there are no actions to run.
func (m Model) updateStoryDetail(key tea.KeyMsg) (Model, tea.Cmd) {
	if key.Type == tea.KeyEsc {
		return m.backToStoryList()
	}
	if !m.storyActionsAvailable() {
		return m, nil
	}
	story, _ := m.detailStory()
	actions := m.storyDetailActions()
	switch key.Type {
	case tea.KeyUp:
		if m.storyDetail.cursor > 0 {
			m.storyDetail.cursor--
		}
	case tea.KeyDown:
		if m.storyDetail.cursor < len(actions)-1 {
			m.storyDetail.cursor++
		}
	case tea.KeyEnter:
		if m.storyDetail.cursor >= 0 && m.storyDetail.cursor < len(actions) {
			return actions[m.storyDetail.cursor].open(m, story)
		}
	case tea.KeyRunes:
		if string(key.Runes) == "r" && m.storyDetail.origin == storyDetailFromSprint {
			return m.reloadStoryTasks()
		}
		for _, action := range actions {
			if string(key.Runes) == action.key {
				return action.open(m, story)
			}
		}
	}
	return m, nil
}

// backToStoryList returns to the list that opened the detail, keeping its
// filter and cursor. It never issues a request.
func (m Model) backToStoryList() (Model, tea.Cmd) {
	if m.storyDetail.origin == storyDetailFromSprint {
		m.Screen = "sprint-assigned-list"
	} else {
		m.Screen = "backlog"
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
	m.Screen = "section"
	return m.beginSectionStep(kind.firstStep())
}

// reloadDetailStoryInPlace reloads the list that owns the story of the detail,
// so the detail rereads it when something it displays has changed. The Backlog
// detail keeps the Backlog reload contract; the Sprint detail reloads the
// assigned stories without leaving the detail.
func (m Model) reloadDetailStoryInPlace() (Model, tea.Cmd) {
	if m.storyDetail.origin == storyDetailFromSprint {
		m.sprintStoriesLoading, m.sprintMessage = true, ""
		return m, m.listAssignedSprintStoriesCmd(m.selectedSprintID)
	}
	return m.startBacklogLoad()
}

// backToStoryDetail returns to the detail of the story the flow started from,
// keeping the highlighted action row. It never issues a request.
func (m Model) backToStoryDetail() (Model, tea.Cmd) {
	m.Screen = "story-detail"
	return m, nil
}

// loadStoryTasksCmd fetches the tasks of the story shown by a Sprint detail.
func (m Model) loadStoryTasksCmd(storyID string) tea.Cmd {
	client, projectID := m.taskClient, m.activeProjectID
	return func() tea.Msg {
		response, err := client.ListByStory(context.Background(), projectID, storyID)
		return storyTasksResult{StoryID: storyID, Response: response, Err: err}
	}
}

type storyTasksResult struct {
	StoryID  string
	Response APIResponse
	Err      error
}

// applyStoryTasks stores the tasks only while the same Sprint detail is showing
// them, discarding answers that belong to a detail that was already left.
func (m Model) applyStoryTasks(result storyTasksResult) (Model, tea.Cmd) {
	if m.Screen != "story-detail" || m.storyDetail.origin != storyDetailFromSprint ||
		m.storyDetail.storyID != result.StoryID || !m.storyDetail.tasksLoading {
		return m, nil
	}
	m.storyDetail.tasksLoading = false
	if message := readFailure(result.Response, result.Err); message != "" {
		m.storyDetail.tasksMessage = message
		return m, nil
	}
	var listing struct {
		Tasks []Task `json:"tasks"`
	}
	if err := json.Unmarshal(result.Response.Body, &listing); err != nil {
		m.storyDetail.tasksMessage = unreadableResponse(err)
		return m, nil
	}
	m.storyDetail.tasks, m.storyDetail.tasksMessage = listing.Tasks, ""
	return m, nil
}

// reloadStoryTasks refreshes the task list of the Sprint detail in place.
func (m Model) reloadStoryTasks() (Model, tea.Cmd) {
	m.storyDetail.tasksLoading, m.storyDetail.tasksMessage = true, ""
	return m, m.loadStoryTasksCmd(m.storyDetail.storyID)
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
	if m.storyDetail.origin == storyDetailFromSprint {
		l.add(keyNav, true, "Esc", "Sprint", "Volver a las historias del Sprint")
	} else {
		l.add(keyNav, true, "Esc", "backlog", "Volver a la lista del backlog")
	}
	if ready {
		for _, action := range m.storyDetailActions() {
			l.add(keyAction, false, action.key, strings.ToLower(action.label), action.help)
		}
		if m.storyDetail.origin == storyDetailFromSprint {
			l.add(keyAction, false, "r", "refrescar tareas", "Volver a consultar las tareas de la historia")
		}
	}
}

func (m Model) viewStoryDetail() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Detalle de la historia"))
	if m.storyDetail.origin == storyDetailFromSprint {
		b.WriteString("  " + appPaneMeta.Render("Sprint: "+m.sprintGoalByID(m.selectedSprintID)))
	}
	b.WriteString("\n\n")
	// The four states are evaluated in this order: loading, error, missing, ready.
	switch {
	case m.storyDetail.origin == storyDetailFromSprint && m.sprintStoriesLoading:
		b.WriteString(statusMutedStyle.Render("Cargando historias…") + "\n")
		return b.String()
	case m.storyDetail.origin == storyDetailFromSprint && m.sprintMessage != "":
		b.WriteString(m.errorBlock(m.sprintMessage) + "\n")
		return b.String()
	case m.storyDetail.origin == storyDetailFromBacklog && m.backlogLoading:
		b.WriteString(statusMutedStyle.Render("Cargando backlog…") + "\n")
		return b.String()
	case m.storyDetail.origin == storyDetailFromBacklog && m.backlogMessage != "":
		b.WriteString(m.errorBlock(m.backlogMessage) + "\n")
		return b.String()
	}
	story, found := m.detailStory()
	if !found {
		if m.storyDetail.origin == storyDetailFromSprint {
			b.WriteString(emptyState("◇", "La historia ya no está en el Sprint.", "Presione Esc para volver a la lista.") + "\n")
		} else {
			b.WriteString(emptyState("◇", "La historia ya no está en el backlog.", "Presione Esc para volver a la lista.") + "\n")
		}
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
	if m.storyDetail.origin == storyDetailFromSprint {
		b.WriteString("\n" + panelTitleStyle.Render("Tareas") + "\n")
		b.WriteString(m.storyTasksView())
	}
	b.WriteString("\n" + panelTitleStyle.Render("Acciones") + "\n")
	for i, action := range m.storyDetailActions() {
		b.WriteString(m.row(i == m.storyDetail.cursor, action.key+"  "+action.label) + "\n")
	}
	return b.String()
}

// storyTasksView renders the read-only task list of the Sprint detail.
func (m Model) storyTasksView() string {
	switch {
	case m.storyDetail.tasksLoading:
		return statusMutedStyle.Render("Cargando tareas…") + "\n"
	case m.storyDetail.tasksMessage != "":
		return m.errorBlock(m.storyDetail.tasksMessage) + "\n"
	case len(m.storyDetail.tasks) == 0:
		return hintStyle.Render("Sin tareas todavía.") + "\n"
	}
	var b strings.Builder
	for _, task := range m.storyDetail.tasks {
		b.WriteString("• " + task.Title)
		if task.EstimatedHours != nil {
			b.WriteString(hintStyle.Render("  " + strconv.FormatFloat(*task.EstimatedHours, 'f', -1, 64) + " h"))
		}
		b.WriteString("\n")
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
