package tui

import (
	"context"
	"encoding/json"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// openProjectPicker shows the searchable project list (the "browser"), the
// start screen and the parent of every workspace tab.
func (m Model) openProjectPicker() (Model, tea.Cmd) {
	m.Screen = "project-picker"
	m.projectFilter = ""
	m.projectFiltering = false
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
	autoOpen := m.autoOpen
	m.autoOpen = false
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
	if autoOpen && len(m.projects) == 1 {
		return m.selectProject(m.projects[0])
	}
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

	if m.projectFiltering {
		switch key.Type {
		case tea.KeyEsc:
			m.projectFiltering, m.projectFilter, m.projectCursor = false, "", 0
			return m, nil
		case tea.KeyEnter:
			// Stop searching but keep the filter, as the backlog does, so the
			// filtered list can then be acted on with Enter / e / n.
			m.projectFiltering = false
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
		}
		if value, ok := editText(m.projectFilter, key); ok {
			m.projectFilter = value
			m.projectCursor = 0
		}
		return m, nil
	}

	switch key.Type {
	case tea.KeyEsc:
		if m.projectFilter != "" {
			m.projectFilter, m.projectCursor = "", 0
			return m, nil
		}
	case tea.KeyUp:
		if m.projectCursor > 0 {
			m.projectCursor--
		}
	case tea.KeyDown:
		if m.projectCursor < len(list)-1 {
			m.projectCursor++
		}
	case tea.KeyRunes:
		switch string(key.Runes) {
		case "/":
			m.projectFiltering = true
		case "e":
			return m.editSelectedProject()
		case "n":
			return m.openProjectForm("create", ProjectSummary{})
		case "r":
			m.projectsLoading = true
			return m, m.listProjectsCmd()
		default:
			// Any other character starts a search seeded with it, so typing
			// still feels immediate while e and n stay reserved for actions.
			m.projectFiltering = true
			if value, ok := editText(m.projectFilter, key); ok {
				m.projectFilter = value
				m.projectCursor = 0
			}
		}
	case tea.KeyEnter:
		if len(list) == 0 {
			return m, nil
		}
		return m.selectProject(list[m.projectCursor])
	}
	return m, nil
}

// selectProject opens the workspace of project on its Dashboard tab.
func (m Model) selectProject(project ProjectSummary) (Model, tea.Cmd) {
	m.activeProjectID = project.ID
	m.activeProjectName = project.Name
	m.activeProjectStatus = project.Status
	m.message = ""
	// The detail and the return target belong to the previous project's flow.
	m.storyDetail = storyDetailState{}
	return m.openDashboard()
}

// isSprintScreen reports whether screen belongs to the Sprints flow.
func isSprintScreen(screen string) bool {
	switch screen {
	case "sprints", "sprint-goal", "sprint-history", "sprint-confirm", "sprint-story-select", "sprint-assigned-list", "sprint-task-form":
		return true
	}
	return false
}

func (m Model) viewProjectPicker() string {
	list := m.filteredProjects()
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Proyectos"))
	if len(list) > 0 {
		b.WriteString("  " + appPaneMeta.Render(listCounter(m.projectCursor, len(list))))
	}
	b.WriteString("\n\n")

	if m.projectFiltering || m.projectFilter != "" {
		b.WriteString(searchLine(m.projectFilter, m.projectFiltering))
		b.WriteString("\n\n")
	}

	if m.message != "" {
		b.WriteString(m.errorBlock(m.message))
		b.WriteString("\n\n")
	}
	switch {
	case m.projectsLoading:
		b.WriteString(statusMutedStyle.Render("Cargando proyectos…"))
		return b.String()
	case len(m.projects) == 0:
		b.WriteString(emptyState("◇", "No hay proyectos todavía.", "Creá uno con n para empezar."))
		return b.String()
	}

	if len(list) == 0 {
		b.WriteString(emptyState("⌕", "Sin coincidencias.", "Probá con otro nombre o borrá el filtro con Esc."))
		return b.String()
	}
	used := strings.Count(b.String(), "\n")
	available := m.listAreaHeight() - used
	if m.listAreaHeight() > 0 && available < 1 {
		available = 1
	}
	start, end := windowBounds(m.projectCursor, len(list), available)
	for i := start; i < end; i++ {
		b.WriteString(m.row(i == m.projectCursor, projectPickerRow(list[i], list[i].ID == m.activeProjectID)) + "\n")
	}
	return b.String()
}

func projectPickerRow(project ProjectSummary, active bool) string {
	// The session marker leads the row so a narrow terminal truncating the
	// right side never hides which project is active.
	marker := "  "
	if active {
		marker = statusOkStyle.Render("✓ ")
	}
	name := project.Name
	if project.Status != "" {
		name += "  " + projectStatusBadge(project.Status)
	}
	return marker + name + hintStyle.Render("   "+shortID(project.ID))
}
