package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// Model is the Bubble Tea model for the TUI.
type Model struct {
	APIURL  string
	Choices []string
	Cursor  int
	Screen  string
	APIErr  error

	projectCursor int
	projectID     string
	projectAction string
	formField     int
	form          ProjectInput
	message       string
	client        *ProjectClient

	backlogProjectID string // project confirmed for the session
	backlogInput     string // text being edited in backlog-project
	backlogCursor    int
	backlogStories   []Story
	backlogLoading   bool
	backlogForm      storyForm
	backlogMessage   string // text of backlog-result
	backlogReturn    string // screen reached when leaving backlog-result
	backlogClient    *BacklogClient
	sprintClient     *SprintClient
	sprintCursor     int
	sprintProjectID  string
	sprintGoal       string
	sprintMessage    string
	sprintAction     string
	selectedSprintID string
	taskStoryID      string
	taskTitle        string
	taskHours        string
	taskField        int
	taskClient       *TaskClient
	openSprints      []Sprint
	sprintStories    []Story
	selectedStories  map[string]bool
}

// NewModel creates the base navigation model.
func NewModel(apiURL string) Model {
	return Model{
		APIURL:  apiURL,
		Choices: []string{"Proyectos", "Backlog", "Sprints", "Salir"},
		Screen:  "menu",
		client:  NewProjectClient(apiURL),

		backlogClient: NewBacklogClient(apiURL),
		sprintClient:  NewSprintClient(apiURL),
		taskClient:    NewTaskClient(apiURL),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if result, ok := msg.(ProjectResultMsg); ok {
		return m.applyProjectResult(result)
	}
	if result, ok := msg.(BacklogResultMsg); ok {
		return m.applyBacklogResult(result)
	}
	if result, ok := msg.(SprintResultMsg); ok {
		return m.applySprintResult(result)
	}
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return m, nil
	}
	if key.Type == tea.KeyCtrlC || (isQuitRune(key) && quitsOnQ(m.Screen)) {
		return m, tea.Quit
	}
	switch m.Screen {
	case "menu":
		return m.updateMenu(key)
	case "projects":
		return m.updateProjects(key)
	case "create-project", "edit-project":
		return m.updateProjectForm(key)
	case "project-id":
		return m.updateProjectID(key)
	case "project-result":
		if key.Type == tea.KeyEsc || key.Type == tea.KeyEnter {
			m.Screen = "projects"
			m.message = ""
		}
		return m, nil
	case "backlog":
		return m.updateBacklog(key)
	case "backlog-project":
		return m.updateBacklogProject(key)
	case "backlog-form":
		return m.updateBacklogForm(key)
	case "backlog-result":
		return m.updateBacklogResult(key)
	case "sprints":
		return m.updateSprints(key)
	case "sprint-project", "sprint-goal":
		return m.updateSprintForm(key)
	case "sprint-open-list":
		return m.updateOpenSprints(key)
	case "sprint-story-select":
		return m.updateSprintStorySelection(key)
	case "sprint-assigned-list":
		return m.updateAssignedStories(key)
	case "sprint-task-form":
		return m.updateSprintTaskForm(key)
	case "sprint-result":
		if key.Type == tea.KeyEsc || key.Type == tea.KeyEnter {
			m.Screen, m.sprintMessage = "sprints", ""
		}
		return m, nil
	default:
		if key.Type == tea.KeyEsc {
			m.Screen = "menu"
			m.Cursor = 0
		}
		return m, nil
	}
}

// quitsOnQ reports whether q/Q exits from screen. Text-entry screens are not
// listed, so q is a plain character there; Ctrl+C always quits.
func quitsOnQ(screen string) bool {
	switch screen {
	case "menu", "projects", "project-result", "backlog", "backlog-result", "sprints":
		return true
	}
	return false
}

func isQuitRune(key tea.KeyMsg) bool {
	return key.Type == tea.KeyRunes && len(key.Runes) == 1 && (key.Runes[0] == 'q' || key.Runes[0] == 'Q')
}

// editText applies a text-editing key to value: runes and the space bar are
// appended and backspace removes the last rune, so the value stays valid UTF-8.
// The boolean reports whether the key was consumed.
func editText(value string, key tea.KeyMsg) (string, bool) {
	switch key.Type {
	case tea.KeyRunes:
		return value + string(key.Runes), true
	case tea.KeySpace:
		return value + " ", true
	case tea.KeyBackspace:
		_, size := utf8.DecodeLastRuneInString(value)
		return value[:len(value)-size], true
	}
	return value, false
}

func (m Model) updateMenu(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyUp:
		if m.Cursor > 0 {
			m.Cursor--
		}
	case tea.KeyDown:
		if m.Cursor < len(m.Choices)-1 {
			m.Cursor++
		}
	case tea.KeyEnter:
		switch m.Cursor {
		case 0:
			m.Screen, m.projectCursor = "projects", 0
		case 1:
			return m.openBacklog()
		case 2:
			m.Screen = "sprints"
		default:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) updateProjects(key tea.KeyMsg) (Model, tea.Cmd) {
	options := []string{"Crear proyecto", "Modificar proyecto", "Consultar estado", "Volver"}
	switch key.Type {
	case tea.KeyUp:
		if m.projectCursor > 0 {
			m.projectCursor--
		}
	case tea.KeyDown:
		if m.projectCursor < len(options)-1 {
			m.projectCursor++
		}
	case tea.KeyEsc:
		m.Screen, m.Cursor = "menu", 0
	case tea.KeyEnter:
		switch m.projectCursor {
		case 0:
			m.Screen, m.form, m.formField = "create-project", ProjectInput{}, 0
		case 1:
			m.Screen, m.projectID, m.form, m.formField, m.projectAction = "project-id", "", ProjectInput{}, 0, "edit"
			m.message = "Ingresá el ID del proyecto para editar."
		case 2:
			m.Screen, m.projectID, m.projectAction = "project-id", "", "status"
			m.message = "Ingresá el ID del proyecto para consultar estado."
		default:
			m.Screen, m.Cursor = "menu", 0
		}
	}
	return m, nil
}

func (m Model) updateProjectID(key tea.KeyMsg) (Model, tea.Cmd) {
	if key.Type == tea.KeyEsc {
		m.Screen, m.message = "projects", ""
		return m, nil
	}
	if value, ok := editText(m.projectID, key); ok {
		m.projectID = value
		return m, nil
	}
	if key.Type == tea.KeyEnter && strings.TrimSpace(m.projectID) != "" {
		id := strings.TrimSpace(m.projectID)
		if m.projectAction == "edit" {
			m.formField = 0
			m.Screen = "edit-project"
			return m, nil
		}
		return m, m.statusCmd(id)
	}
	return m, nil
}

func (m Model) updateProjectForm(key tea.KeyMsg) (Model, tea.Cmd) {
	fields := []*string{&m.form.Name, &m.form.StartDate, &m.form.PlannedFinishDate}
	if key.Type == tea.KeyEsc {
		m.Screen = "projects"
		return m, nil
	}
	if value, ok := editText(*fields[m.formField], key); ok {
		*fields[m.formField] = value
		return m, nil
	}
	if key.Type == tea.KeyTab || key.Type == tea.KeyDown {
		m.formField = (m.formField + 1) % len(fields)
		return m, nil
	}
	if key.Type == tea.KeyShiftTab || key.Type == tea.KeyUp {
		m.formField = (m.formField + len(fields) - 1) % len(fields)
		return m, nil
	}
	if key.Type == tea.KeyEnter {
		if m.formField < len(fields)-1 {
			m.formField++
			return m, nil
		}
		if m.Screen == "create-project" {
			return m, m.createCmd(m.form)
		}
		return m, m.updateCmd(m.projectID, m.form)
	}
	return m, nil
}

// ProjectResultMsg is the asynchronous result of a project API request.
type ProjectResultMsg struct {
	Response APIResponse
	Err      error
}

func (m Model) createCmd(input ProjectInput) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		response, err := client.Create(context.Background(), input)
		return ProjectResultMsg{Response: response, Err: err}
	}
}

func (m Model) updateCmd(id string, input ProjectInput) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		response, err := client.Update(context.Background(), id, input)
		return ProjectResultMsg{Response: response, Err: err}
	}
}

func (m Model) statusCmd(id string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		response, err := client.Status(context.Background(), id)
		return ProjectResultMsg{Response: response, Err: err}
	}
}

func (m Model) applyProjectResult(result ProjectResultMsg) (Model, tea.Cmd) {
	m.Screen = "project-result"
	if result.Err != nil {
		m.message = "Error de conexión: " + result.Err.Error()
		return m, nil
	}
	if result.Response.StatusCode < 200 || result.Response.StatusCode >= 300 {
		detail := strings.TrimSpace(string(result.Response.Body))
		if result.Response.StatusCode == 404 {
			m.message = "El proyecto no existe (404): " + detail
		} else {
			m.message = fmt.Sprintf("Error HTTP %d: %s", result.Response.StatusCode, detail)
		}
		return m, nil
	}
	var status struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if json.Unmarshal(result.Response.Body, &status) == nil && status.Status != "" {
		m.message = fmt.Sprintf("Proyecto %s: %s (%s)", status.ID, status.Name, status.Status)
	} else {
		m.message = strings.TrimSpace(string(result.Response.Body))
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	var b strings.Builder
	b.WriteString(header())
	b.WriteString("\n\n")
	if m.APIErr != nil {
		b.WriteString(errorBannerStyle.Render("API no disponible: " + m.APIErr.Error()))
		b.WriteString("\n")
		b.WriteString(errorHintStyle.Render("Revisá API_URL o levantá la API; la navegación sigue disponible."))
		b.WriteString("\n\n")
	} else {
		b.WriteString(statusOkStyle.Render("● API conectada · " + m.APIURL))
		b.WriteString("\n\n")
	}
	switch m.Screen {
	case "menu":
		for i, choice := range m.Choices {
			if m.Cursor == i {
				b.WriteString(cursorStyle.Render("▸") + " " + selectedStyle.Render(choice))
			} else {
				b.WriteString("  " + itemStyle.Render(choice))
			}
			b.WriteString("\n")
		}
	case "projects":
		b.WriteString("Gestión de proyectos\n\n")
		for i, option := range []string{"Crear proyecto", "Modificar proyecto", "Consultar estado", "Volver"} {
			prefix := "  "
			if i == m.projectCursor {
				prefix = "▸ "
			}
			b.WriteString(prefix + option + "\n")
		}
	case "create-project", "edit-project":
		title := "Crear proyecto"
		if m.Screen == "edit-project" {
			title = "Modificar proyecto " + m.projectID
		}
		b.WriteString(title + "\n\n")
		fields := []struct{ label, value string }{{"Nombre", m.form.Name}, {"Fecha inicio (YYYY-MM-DD)", m.form.StartDate}, {"Fecha fin planificada (YYYY-MM-DD)", m.form.PlannedFinishDate}}
		for i, field := range fields {
			prefix := "  "
			if i == m.formField {
				prefix = "▸ "
			}
			b.WriteString(prefix + field.label + ": " + field.value + "\n")
		}
		b.WriteString("\nTab/↑/↓ para cambiar campo; Enter en el último para enviar.\n")
	case "project-id":
		b.WriteString(m.message + "\n\nID: " + m.projectID + "_\nEnter para continuar, Esc para volver.\n")
	case "project-result":
		b.WriteString("Resultado\n\n" + m.message + "\n\nEnter/Esc para volver.\n")
	case "backlog":
		b.WriteString(m.viewBacklog())
	case "backlog-project":
		b.WriteString(m.viewBacklogProject())
	case "backlog-form":
		b.WriteString(m.viewBacklogForm())
	case "backlog-result":
		b.WriteString(m.viewBacklogResult())
	case "sprints":
		b.WriteString("Operaciones de Sprint\n\n")
		for i, option := range []string{"Crear Sprint", "Asignar historias", "Crear tareas", "Volver"} {
			b.WriteString(choiceRow(i == m.sprintCursor, option) + "\n")
		}
	case "sprint-project":
		b.WriteString("Proyecto del Sprint\n\nID del proyecto: " + m.sprintProjectID + "_\nEnter para continuar, Esc para volver.\n")
	case "sprint-goal":
		b.WriteString("Crear Sprint\n\nObjetivo: " + m.sprintGoal + "_\nEnter para enviar, Esc para volver.\n")
	case "sprint-result":
		b.WriteString("Resultado\n\n" + m.sprintMessage + "\n\nEnter/Esc para volver.\n")
	case "sprint-open-list":
		b.WriteString("Elegir Sprint abierto\n\n")
		for i, sprint := range m.openSprints {
			b.WriteString(choiceRow(i == m.sprintCursor, sprint.ID+" · "+sprint.SprintGoal) + "\n")
		}
		if len(m.openSprints) == 0 {
			b.WriteString("No hay Sprints abiertos.\n")
		}
	case "sprint-story-select":
		b.WriteString("Elegir historias para asignar\n\n")
		for _, story := range m.sprintStories {
			mark := "[ ]"
			if m.selectedStories[story.ID] {
				mark = "[x]"
			}
			b.WriteString(fmt.Sprintf("%s %s · %s\n", mark, story.ID, story.Title))
		}
		b.WriteString("\nSpace para alternar; Enter para asignar.\n")
	case "sprint-assigned-list":
		b.WriteString("Historias asignadas al Sprint\n\n")
		for i, story := range m.sprintStories {
			b.WriteString(choiceRow(i == m.sprintCursor, story.ID+" · "+story.Title) + "\n")
		}
	case "sprint-task-form":
		b.WriteString("Crear tarea para " + m.taskStoryID + "\n\n")
		b.WriteString(choiceRow(m.taskField == 0, "Título: "+m.taskTitle) + "\n")
		b.WriteString(choiceRow(m.taskField == 1, "Horas estimadas (opcional): "+m.taskHours) + "\n")
		if m.sprintMessage != "" {
			b.WriteString("\n" + errorBannerStyle.Render(m.sprintMessage) + "\n")
		}
		b.WriteString("\nTab para cambiar campo; Enter para continuar/enviar.\n")
	default:
		b.WriteString(screenTitleStyle.Render(m.Choices[m.Cursor]))
		b.WriteString("  ")
		b.WriteString(badgeStyle.Render("Disponible en " + placeholderRelease(m.Cursor)))
		b.WriteString("\n\nPulsa esc para volver, q para salir.\n")
	}
	return b.String()
}

func header() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Software Metrics & Estimation"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Sprint 2 · La Interfaz · TUI base"))
	b.WriteString("\n")
	b.WriteString(ruleStyle.Render("────────────────────────────────"))
	return b.String()
}

func placeholderRelease(cursor int) string {
	switch cursor {
	case 0:
		return "TUI-02"
	default:
		return "TUI-04/TUI-05"
	}
}
