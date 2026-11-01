package tui

import (
	"context"
	"encoding/json"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// openProjectPicker shows the searchable project list (the "browser").
// purpose selects what Enter does with the confirmed project, so the same
// screen serves the Projects menu, Backlog and Sprints.
func (m Model) openProjectPicker(purpose string) (Model, tea.Cmd) {
	m.pickerReturn = m.Screen
	m.pickerPurpose = purpose
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
		m.message = ""
		m.Screen = m.pickerReturn
		if m.Screen == "" {
			m.Screen = "menu"
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
		case "p":
			// p is the global «change project» shortcut; the browser already is
			// that changer, so keep it inert instead of seeding a search.
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

// selectProject stores the session context and resumes the flow that opened the
// picker. Sprint flows always land on the Sprints browser, which is the single
// entry point for every Sprint operation.
func (m Model) selectProject(project ProjectSummary) (Model, tea.Cmd) {
	m.activeProjectID = project.ID
	m.activeProjectName = project.Name
	returnTo, purpose := m.pickerReturn, m.pickerPurpose
	m.message = ""

	switch {
	case purpose == "backlog":
		m.Screen = "backlog"
		m.backlogFilter, m.backlogFiltering = "", false
		return m.startBacklogLoad()
	case purpose == "defects" || returnTo == "defects" || returnTo == "defect-detail":
		return m.openDefects()
	case purpose == "sprints" || isSprintScreen(returnTo):
		return m.enterSprints()
	case strings.HasPrefix(purpose, "section:"):
		return m.openSection(sectionKind(strings.TrimPrefix(purpose, "section:")))
	case returnTo == "section" && m.section.kind != "":
		return m.openSection(m.section.kind)
	case returnTo == "effort-result":
		// The result belongs to the project that was just left: drop the flow and
		// start Esfuerzo again on the stories of the new one.
		return m.openSection(sectionEffort)
	default:
		m.Screen = returnTo
		if m.Screen == "" {
			m.Screen = "menu"
		}
		if m.Screen == "menu" {
			m.Cursor = 0
		}
		return m, nil
	}
}

// isSprintScreen reports whether screen belongs to the Sprints flow.
func isSprintScreen(screen string) bool {
	switch screen {
	case "sprints", "sprint-goal", "sprint-history", "sprint-confirm", "sprint-story-select", "sprint-assigned-list", "sprint-task-form", "sprint-result":
		return true
	}
	return false
}

func (m Model) viewProjectPicker() string {
	title, hint := "Elegí un proyecto", "Enter usa el proyecto para Backlog y Sprints."
	if m.pickerPurpose == "projects" {
		title = "Proyectos"
		hint = "✓ = proyecto activo · Enter usa · e edita · n crea · / busca por nombre o ID."
	}
	if strings.HasPrefix(m.pickerPurpose, "section:") {
		hint = "Enter continúa en " + m.section.kind.title() + " con el proyecto elegido."
	}
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render(hint))
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

	list := m.filteredProjects()
	if len(list) == 0 {
		b.WriteString(emptyState("⌕", "Sin coincidencias.", "Probá con otro nombre o borrá el filtro con Esc."))
		return b.String()
	}
	for i, project := range list {
		b.WriteString(m.row(i == m.projectCursor, projectPickerRow(project, project.ID == m.activeProjectID)) + "\n")
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
