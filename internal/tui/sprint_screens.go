package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// updateSprintTaskForm drives the two-field task form of the highlighted story.
func (m Model) updateSprintTaskForm(key tea.KeyMsg) (Model, tea.Cmd) {
	value := &m.taskTitle
	if m.taskField == 1 {
		value = &m.taskHours
	}
	if key.Type == tea.KeyEsc {
		m.Screen = "sprint-assigned-list"
		return m, nil
	}
	if key.Type == tea.KeyTab || key.Type == tea.KeyDown || key.Type == tea.KeyUp {
		m.taskField = (m.taskField + 1) % 2
		return m, nil
	}
	if text, ok := editText(*value, key); ok {
		*value = text
		m.sprintMessage = ""
		return m, nil
	}
	if key.Type == tea.KeyEnter {
		if m.taskField == 0 {
			m.taskField = 1
			return m, nil
		}
		var hours *float64
		if strings.TrimSpace(m.taskHours) != "" {
			parsed, err := parseEstimatedHours(m.taskHours)
			if err != nil {
				m.sprintMessage = err.Error()
				return m, nil
			}
			hours = parsed
		}
		client, projectID, sprintID, storyID, title := m.taskClient, m.activeProjectID, m.selectedSprintID, m.taskStoryID, m.taskTitle
		return m, func() tea.Msg {
			response, err := client.Create(context.Background(), projectID, sprintID, storyID, []TaskInput{{Title: title, EstimatedHours: hours}})
			return SprintResultMsg{Op: "tasks", Response: response, Err: err}
		}
	}
	return m, nil
}

// viewSprints renders the browser of open Sprints: Enter opens its stories and
// the secondary keys assign, close, list the history and create.
func (m Model) viewSprints() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Sprints de " + m.projectLabel()))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Enter historias · a asignar · x cerrar · h cerrados · n nuevo · / busca."))
	b.WriteString("\n\n")
	if m.sprintFiltering || m.sprintFilter != "" {
		b.WriteString(searchLine(m.sprintFilter, m.sprintFiltering))
		b.WriteString("\n\n")
	}
	if m.sprintLoading {
		b.WriteString(statusMutedStyle.Render("Cargando Sprints…") + "\n")
		return b.String()
	}
	if len(m.openSprints) == 0 {
		b.WriteString(emptyState("◇", "No hay Sprints abiertos.", "Presioná n para crear el primero.") + "\n")
		return b.String()
	}
	sprints := m.visibleOpenSprints()
	b.WriteString(sectionLabel("Abiertos") + "  " + hintStyle.Render(fmt.Sprintf("(%d)", len(sprints))) + "\n")
	if len(sprints) == 0 {
		b.WriteString(emptyState("⌕", "Sin coincidencias.", "Probá con otro objetivo o borrá la búsqueda.") + "\n")
		return b.String()
	}
	for i, sprint := range sprints {
		b.WriteString(m.row(i == m.sprintCursor, sprintRow(sprint)) + "\n")
	}
	return b.String()
}

// sprintRow is the shared row of an open Sprint: goal plus short identifier.
func sprintRow(sprint Sprint) string {
	goal := sprint.SprintGoal
	if goal == "" {
		goal = "(sin objetivo)"
	}
	return goal + hintStyle.Render("   "+shortID(sprint.ID))
}

// viewSprintConfirm asks for confirmation before closing a Sprint.
func (m Model) viewSprintConfirm() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Cerrar Sprint"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Proyecto: " + m.projectLabel()))
	b.WriteString("\n\n")
	b.WriteString("Sprint: " + m.sprintGoalByID(m.selectedSprintID) + hintStyle.Render("   "+shortID(m.selectedSprintID)))
	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render("¿Cerrar este Sprint? Las historias planificadas y completadas quedarán en el historial."))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("s confirma · n o Esc cancela."))
	return b.String()
}

// sprintGoalByID resolves the goal of an open Sprint for the confirmation screen.
func (m Model) sprintGoalByID(id string) string {
	for _, sprint := range m.openSprints {
		if sprint.ID == id {
			if sprint.SprintGoal == "" {
				return "(sin objetivo)"
			}
			return sprint.SprintGoal
		}
	}
	return "(sin objetivo)"
}

// viewSprintHistory renders the closed Sprints with their progress. Enter expands
// the highlighted Sprint and reveals the stories already present in the response.
func (m Model) viewSprintHistory() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Historial de Sprints cerrados"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Proyecto: " + m.projectLabel()))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Enter despliega las historias · / busca · Esc vuelve."))
	b.WriteString("\n\n")
	if m.sprintFiltering || m.sprintFilter != "" {
		b.WriteString(searchLine(m.sprintFilter, m.sprintFiltering))
		b.WriteString("\n\n")
	}
	if m.sprintHistoryLoading {
		b.WriteString(statusMutedStyle.Render("Cargando historial…") + "\n")
		return b.String()
	}
	if len(m.sprintHistory.Sprints) == 0 {
		b.WriteString(emptyState("◇", "No hay Sprints cerrados en este proyecto.", "Cerrá un Sprint para verlo acá.") + "\n")
		return b.String()
	}
	sprints := m.visibleHistorySprints()
	if len(sprints) == 0 {
		b.WriteString(emptyState("⌕", "Sin coincidencias.", "Probá con otro objetivo o borrá la búsqueda.") + "\n")
		return b.String()
	}
	for i, sprint := range sprints {
		completed, total := sprintProgress(sprint)
		row := sprintHistoryTitle(sprint.SprintGoal, sprint.ID) + hintStyle.Render(fmt.Sprintf("   %d/%d historias completadas", completed, total))
		b.WriteString(m.row(i == m.historyCursor, row) + "\n")
		if m.historyExpanded == sprint.ID {
			for _, story := range sprint.Stories {
				b.WriteString("     " + sprintHistoryStoryRow(story) + "\n")
			}
		}
	}
	return b.String()
}

// sprintProgress counts the completed stories of a closed Sprint.
func sprintProgress(sprint ClosedSprint) (int, int) {
	completed := 0
	for _, story := range sprint.Stories {
		if story.Completed {
			completed++
		}
	}
	return completed, len(sprint.Stories)
}

func sprintHistoryTitle(goal, id string) string {
	if goal == "" {
		goal = "(sin objetivo)"
	}
	return titleStyle.Render(goal) + hintStyle.Render("   "+shortID(id))
}

func sprintHistoryStoryRow(story SprintStory) string {
	status := storyStatusBadge("pendiente")
	if story.Completed {
		status = storyStatusBadge("completada")
	}
	row := story.Title + "  " + status
	if story.Priority != "" {
		row += "  " + priorityBadge(story.Priority)
	}
	if story.StoryPoints != nil {
		row += hintStyle.Render(fmt.Sprintf("   %d pts", *story.StoryPoints))
	}
	return row
}

// viewSprintGoal renders the create-Sprint form.
func (m Model) viewSprintGoal() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Crear Sprint"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Proyecto: " + m.projectLabel()))
	b.WriteString("\n\n")
	b.WriteString(m.formRow(true, "Objetivo", m.sprintGoal))
	return b.String()
}

// viewSprintStorySelect renders the multiselect of backlog stories to assign.
func (m Model) viewSprintStorySelect() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Elegí historias para asignar"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Space marca o desmarca · Enter confirma · / busca · Esc vuelve."))
	b.WriteString("\n\n")
	if m.sprintFiltering || m.sprintFilter != "" {
		b.WriteString(searchLine(m.sprintFilter, m.sprintFiltering))
		b.WriteString("\n\n")
	}
	if m.sprintStoriesLoading {
		b.WriteString(statusMutedStyle.Render("Cargando historias…") + "\n")
		return b.String()
	}
	if len(m.sprintStories) == 0 {
		b.WriteString(emptyState("◇", "El backlog no tiene historias.", "Agregá historias al backlog primero.") + "\n")
		return b.String()
	}
	stories := m.visibleSprintStories()
	if len(stories) == 0 {
		b.WriteString(emptyState("⌕", "Sin coincidencias.", "Probá con otro título o borrá la búsqueda.") + "\n")
		return b.String()
	}
	for i, story := range stories {
		mark := "[ ]"
		if m.selectedStories[story.ID] {
			mark = "[x]"
		}
		row := mark + " " + story.Title
		if story.Priority != "" {
			row += "  " + priorityBadge(story.Priority)
		}
		b.WriteString(m.row(i == m.sprintCursor, row) + "\n")
	}
	return b.String()
}

// viewAssignedStories renders the stories of the open Sprint: Enter creates tasks,
// c completes the highlighted story.
func (m Model) viewAssignedStories() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Historias del Sprint · " + m.sprintGoalByID(m.selectedSprintID)))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Enter crea tareas · c completa la historia · / busca · Esc vuelve."))
	b.WriteString("\n\n")
	if m.sprintFiltering || m.sprintFilter != "" {
		b.WriteString(searchLine(m.sprintFilter, m.sprintFiltering))
		b.WriteString("\n\n")
	}
	if m.sprintStoriesLoading {
		b.WriteString(statusMutedStyle.Render("Cargando historias…") + "\n")
		return b.String()
	}
	if len(m.sprintStories) == 0 {
		b.WriteString(emptyState("◇", "El Sprint no tiene historias asignadas.", "Asigná historias con a desde el navegador de Sprints.") + "\n")
		return b.String()
	}
	stories := m.visibleSprintStories()
	if len(stories) == 0 {
		b.WriteString(emptyState("⌕", "Sin coincidencias.", "Probá con otro título o borrá la búsqueda.") + "\n")
		return b.String()
	}
	for i, story := range stories {
		row := story.Title
		if story.Priority != "" {
			row += "  " + priorityBadge(story.Priority)
		}
		if story.Status != "" {
			row += "  " + storyStatusBadge(story.Status)
		}
		b.WriteString(m.row(i == m.sprintCursor, row) + "\n")
	}
	return b.String()
}

func (m Model) viewSprintTaskForm() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Crear tarea"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Historia: " + shortID(m.taskStoryID)))
	b.WriteString("\n\n")
	b.WriteString(m.formRow(m.taskField == 0, "Título", m.taskTitle) + "\n")
	b.WriteString(m.formRow(m.taskField == 1, "Horas estimadas (opcional)", m.taskHours) + "\n")
	if m.sprintMessage != "" {
		b.WriteString("\n" + m.errorBlock(m.sprintMessage) + "\n")
	}
	return b.String()
}

func (m Model) viewSprintResult() string {
	return screenTitleStyle.Render("Resultado") + "\n\n" + m.resultBlock(m.sprintMessage, m.sprintOK)
}
