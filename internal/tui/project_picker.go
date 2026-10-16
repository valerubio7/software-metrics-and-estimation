package tui

import (
	"context"
	"encoding/json"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// openProjectPicker shows the searchable project list. purpose selects what
// happens once a project is confirmed, so the same screen serves every flow.
func (m Model) openProjectPicker(purpose string) (Model, tea.Cmd) {
	m.pickerReturn = m.Screen
	m.pickerPurpose = purpose
	m.Screen = "project-picker"
	m.projectFilter = ""
	m.projectCursor = 0
	m.message = ""
	m.projectsLoading = true
	return m, m.listProjectsCmd()
}

func (m Model) listProjectsCmd() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		response, err := client.List(context.Background())
		return ProjectsResultMsg{Response: response, Err: err}
	}
}

// applyProjectsResult stores the collection. Failures stay on the picker as an
// inline banner so the user can retry or go back without losing the flow.
func (m Model) applyProjectsResult(result ProjectsResultMsg) (Model, tea.Cmd) {
	if m.Screen != "project-picker" || !m.projectsLoading {
		return m, nil
	}
	m.projectsLoading = false
	if result.Err != nil {
		m.message = "Error de conexión: " + result.Err.Error()
		return m, nil
	}
	if result.Response.StatusCode < 200 || result.Response.StatusCode >= 300 {
		m.message = describeBacklogFailure(result.Response)
		return m, nil
	}
	var listing struct {
		Projects []ProjectSummary `json:"projects"`
	}
	if err := json.Unmarshal(result.Response.Body, &listing); err != nil {
		m.message = "No se pudo interpretar la respuesta de la API: " + err.Error()
		return m, nil
	}
	m.projects = listing.Projects
	return m, nil
}

func (m Model) filteredProjects() []ProjectSummary {
	filter := strings.ToLower(strings.TrimSpace(m.projectFilter))
	if filter == "" {
		return m.projects
	}
	matches := make([]ProjectSummary, 0, len(m.projects))
	for _, project := range m.projects {
		if strings.Contains(strings.ToLower(project.Name), filter) || strings.Contains(strings.ToLower(project.ID), filter) {
			matches = append(matches, project)
		}
	}
	return matches
}

func (m Model) updateProjectPicker(key tea.KeyMsg) (Model, tea.Cmd) {
	list := m.filteredProjects()
	switch key.Type {
	case tea.KeyEsc:
		m.Screen, m.message = m.pickerReturn, ""
		if m.Screen == "" {
			m.Screen = "menu"
		}
		return m, nil
	case tea.KeyUp:
		if m.projectCursor > 0 {
			m.projectCursor--
		}
		return m, nil
	case tea.KeyDown:
		if m.projectCursor < len(list)-1 {
			m.projectCursor++
		}
		return m, nil
	case tea.KeyEnter:
		if len(list) == 0 {
			return m, nil
		}
		return m.selectProject(list[m.projectCursor])
	}
	if value, ok := editText(m.projectFilter, key); ok {
		m.projectFilter = value
		m.projectCursor = 0
	}
	return m, nil
}

// selectProject stores the session context and resumes the flow that opened the
// picker.
func (m Model) selectProject(project ProjectSummary) (Model, tea.Cmd) {
	m.activeProjectID = project.ID
	m.activeProjectName = project.Name
	returnTo, purpose := m.pickerReturn, m.pickerPurpose
	m.message = ""

	switch purpose {
	case "backlog":
		m.Screen = "backlog"
		m.backlogFilter, m.backlogFiltering = "", false
		return m.startBacklogLoad()
	case "sprint-create":
		m.Screen, m.sprintAction, m.sprintGoal = "sprint-goal", "create", ""
		return m, nil
	case "sprint-assign", "sprint-tasks", "sprint-complete", "sprint-close", "sprint-history":
		m.sprintAction = strings.TrimPrefix(purpose, "sprint-")
		m.Screen = "sprints"
		if m.sprintAction == "history" {
			return m, m.historySprintsCmd()
		}
		return m, m.listOpenSprintsCmd()
	default:
		m.Screen = returnTo
		if m.Screen == "" {
			m.Screen = "projects"
		}
		return m, nil
	}
}

func (m Model) viewProjectPicker() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Elegí un proyecto"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Buscá por nombre o identificador. Escribí para filtrar; Enter para elegir."))
	b.WriteString("\n\n")
	b.WriteString(cursorStyle.Render("⌕ ") + hintStyle.Render("Buscar: ") + m.projectFilter + "_")
	b.WriteString("\n\n")

	if m.message != "" {
		b.WriteString(m.errorBlock(m.message))
		b.WriteString("\n\n")
	}
	switch {
	case m.projectsLoading:
		b.WriteString(statusMutedStyle.Render("Cargando proyectos…"))
		return b.String()
	case len(m.projects) == 0:
		b.WriteString(emptyState("◇", "No hay proyectos todavía.", "Creá uno desde Proyectos para empezar."))
		return b.String()
	}

	list := m.filteredProjects()
	if len(list) == 0 {
		b.WriteString(emptyState("⌕", "Sin coincidencias.", "Probá con otro nombre o borrá el filtro."))
		return b.String()
	}
	for i, project := range list {
		b.WriteString(m.row(i == m.projectCursor, projectPickerRow(project)) + "\n")
	}
	return b.String()
}

func projectPickerRow(project ProjectSummary) string {
	name := project.Name
	if project.Status != "" {
		name += "  " + projectStatusBadge(project.Status)
	}
	return name + hintStyle.Render("   "+shortID(project.ID))
}
