package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model is the Bubble Tea model for the TUI.
type Model struct {
	APIURL  string
	Choices []string
	Cursor  int
	Screen  string
	APIErr  error

	width, height int
	showHelp      bool

	// Session context shared by every screen.
	activeProjectID   string
	activeProjectName string

	// Project picker.
	projects        []ProjectSummary
	projectsLoading bool
	projectFilter   string
	projectCursor   int
	pickerReturn    string
	pickerPurpose   string

	projectCursorLegacy int
	projectID           string
	projectAction       string
	formField           int
	form                ProjectInput
	message             string
	client              *ProjectClient

	backlogCursor    int
	backlogStories   []Story
	backlogLoading   bool
	backlogFilter    string
	backlogFiltering bool
	backlogForm      storyForm
	backlogMessage   string
	backlogReturn    string
	backlogClient    *BacklogClient
	sprintClient     *SprintClient
	sprintCursor     int
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
	sprintHistory    SprintHistory
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
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
		return m, nil
	}
	if result, ok := msg.(ProjectResultMsg); ok {
		return m.applyProjectResult(result)
	}
	if result, ok := msg.(ProjectsResultMsg); ok {
		return m.applyProjectsResult(result)
	}
	if result, ok := msg.(BacklogResultMsg); ok {
		return m.applyBacklogResult(result)
	}
	if result, ok := msg.(SprintResultMsg); ok {
		if result.Op == "close" || result.Op == "history" {
			return m.applyClosureHistoryResult(result)
		}
		return m.applySprintResult(result)
	}
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return m, nil
	}
	if key.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}
	if m.showHelp {
		if key.Type == tea.KeyEsc || key.Type == tea.KeyEnter || (key.Type == tea.KeyRunes && string(key.Runes) == "?") {
			m.showHelp = false
		}
		return m, nil
	}
	if key.Type == tea.KeyRunes && string(key.Runes) == "?" && !entersText(m.Screen) && !m.filtering() {
		m.showHelp = true
		return m, nil
	}
	if key.Type == tea.KeyRunes && string(key.Runes) == "p" && quitsOnQ(m.Screen) && !m.filtering() {
		return m.openProjectPicker("project")
	}
	if isQuitRune(key) && quitsOnQ(m.Screen) && !m.filtering() {
		return m, tea.Quit
	}
	switch m.Screen {
	case "menu":
		return m.updateMenu(key)
	case "projects":
		return m.updateProjects(key)
	case "project-picker":
		return m.updateProjectPicker(key)
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
	case "backlog-form":
		return m.updateBacklogForm(key)
	case "backlog-result":
		return m.updateBacklogResult(key)
	case "sprints":
		return m.updateClosureHistoryMenu(key)
	case "sprint-goal":
		return m.updateSprintForm(key)
	case "sprint-history":
		if key.Type == tea.KeyEsc || key.Type == tea.KeyEnter {
			m.Screen, m.sprintMessage = "sprints", ""
		}
		return m, nil
	case "sprint-open-list":
		if m.sprintAction == "close" {
			return m.updateClosureOpenSprints(key)
		}
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

// entersText reports whether the screen owns a focused text field, where "?" is a
// plain character rather than the help shortcut.
func entersText(screen string) bool {
	switch screen {
	case "create-project", "edit-project", "project-id", "backlog-form", "sprint-goal", "sprint-task-form", "project-picker":
		return true
	}
	return false
}

// quitsOnQ reports whether q/Q exits from screen. Text-entry screens are not
// listed, so q is a plain character there; Ctrl+C always quits.
func quitsOnQ(screen string) bool {
	switch screen {
	case "menu", "projects", "project-result", "backlog", "backlog-result", "sprints", "sprint-history", "sprint-result", "sprint-open-list", "sprint-assigned-list", "sprint-story-select":
		return true
	}
	return false
}

func isQuitRune(key tea.KeyMsg) bool {
	return key.Type == tea.KeyRunes && len(key.Runes) == 1 && (key.Runes[0] == 'q' || key.Runes[0] == 'Q')
}

// filtering reports whether the backlog search box owns the keyboard, so global
// shortcuts (?, p, q) must be typed into it instead of triggering.
func (m Model) filtering() bool {
	return m.Screen == "backlog" && m.backlogFiltering
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
			m.Screen, m.projectCursorLegacy = "projects", 0
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
	options := projectMenuOptions()
	switch key.Type {
	case tea.KeyUp:
		if m.projectCursorLegacy > 0 {
			m.projectCursorLegacy--
		}
	case tea.KeyDown:
		if m.projectCursorLegacy < len(options)-1 {
			m.projectCursorLegacy++
		}
	case tea.KeyEsc:
		m.Screen, m.Cursor = "menu", 0
	case tea.KeyEnter:
		switch options[m.projectCursorLegacy] {
		case "Crear proyecto":
			m.Screen, m.form, m.formField = "create-project", ProjectInput{}, 0
		case "Elegir proyecto":
			return m.openProjectPicker("projects")
		case "Modificar proyecto":
			m.Screen, m.projectID, m.form, m.formField, m.projectAction = "project-id", "", ProjectInput{}, 0, "edit"
			m.message = "Ingresá el ID del proyecto para editar."
		case "Consultar estado":
			m.Screen, m.projectID, m.projectAction = "project-id", "", "status"
			m.message = "Ingresá el ID del proyecto para consultar estado."
		default:
			m.Screen, m.Cursor = "menu", 0
		}
	}
	return m, nil
}

func projectMenuOptions() []string {
	return []string{"Crear proyecto", "Elegir proyecto", "Modificar proyecto", "Consultar estado", "Volver"}
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
	b.WriteString(m.chrome())
	b.WriteString("\n")
	if m.showHelp {
		b.WriteString(m.viewHelp())
		b.WriteString("\n\n")
		b.WriteString(m.footer())
		return b.String()
	}
	if m.APIErr != nil {
		b.WriteString(errorBannerStyle.Render("API no disponible: " + m.APIErr.Error()))
		b.WriteString("\n")
		b.WriteString(errorHintStyle.Render("Revisá API_URL o levantá la API; la navegación sigue disponible."))
		b.WriteString("\n\n")
	}
	b.WriteString(m.viewBody())
	b.WriteString("\n\n")
	b.WriteString(m.footer())
	return b.String()
}

func (m Model) viewBody() string {
	switch m.Screen {
	case "menu":
		return m.viewMenu()
	case "projects":
		return m.viewProjects()
	case "project-picker":
		return m.viewProjectPicker()
	case "create-project", "edit-project":
		return m.viewProjectForm()
	case "project-id":
		return m.viewProjectID()
	case "project-result":
		return m.viewProjectResult()
	case "backlog":
		return m.viewBacklog()
	case "backlog-form":
		return m.viewBacklogForm()
	case "backlog-result":
		return m.viewBacklogResult()
	case "sprints":
		return m.viewSprintsMenu()
	case "sprint-goal":
		return m.viewSprintGoal()
	case "sprint-history":
		return m.viewSprintHistory()
	case "sprint-open-list":
		return m.viewOpenSprints()
	case "sprint-story-select":
		return m.viewSprintStorySelect()
	case "sprint-assigned-list":
		return m.viewAssignedStories()
	case "sprint-task-form":
		return m.viewSprintTaskForm()
	case "sprint-result":
		return m.viewSprintResult()
	default:
		return screenTitleStyle.Render(m.Choices[m.Cursor])
	}
}

func (m Model) viewMenu() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Panel de control"))
	b.WriteString("\n")
	context := "Sin proyecto activo · elegí uno en Backlog o Sprints"
	if m.activeProjectID != "" {
		context = "Proyecto activo: " + m.activeProjectName
	}
	b.WriteString(hintStyle.Render(context))
	b.WriteString("\n\n")
	for i, choice := range m.Choices {
		b.WriteString(choiceRow(i == m.Cursor, choice) + "\n")
	}
	return b.String()
}

func (m Model) viewProjects() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Proyectos"))
	b.WriteString("\n\n")
	for i, option := range projectMenuOptions() {
		b.WriteString(choiceRow(i == m.projectCursorLegacy, option) + "\n")
	}
	return b.String()
}

func (m Model) viewProjectForm() string {
	title := "Crear proyecto"
	if m.Screen == "edit-project" {
		title = "Modificar proyecto " + m.projectID
	}
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render(title))
	b.WriteString("\n\n")
	fields := []struct{ label, value string }{{"Nombre", m.form.Name}, {"Fecha inicio (YYYY-MM-DD)", m.form.StartDate}, {"Fecha fin planificada (YYYY-MM-DD)", m.form.PlannedFinishDate}}
	for i, field := range fields {
		b.WriteString(formRow(i == m.formField, field.label, field.value) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Tab/↑/↓ para cambiar de campo; Enter en el último para enviar; Esc para volver."))
	return b.String()
}

func (m Model) viewProjectID() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Identificador de proyecto"))
	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render(m.message))
	b.WriteString("\n\n")
	b.WriteString(formRow(true, "ID", m.projectID))
	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render("Enter para continuar, Esc para volver."))
	return b.String()
}

func (m Model) viewProjectResult() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Resultado"))
	b.WriteString("\n\n")
	b.WriteString(resultBody(m.message, !strings.HasPrefix(m.message, "Error") && !strings.Contains(m.message, "no existe")))
	return b.String()
}

// chrome renders the persistent top bar: brand, API status, breadcrumb and rule.
func (m Model) chrome() string {
	width := m.contentWidth()
	left := brandStyle.Render("◆  Software Metrics & Estimation")
	var right string
	if m.APIErr != nil {
		right = badge("api caída", toneDanger)
	} else {
		right = statusOkStyle.Render("● API conectada")
	}
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	var b strings.Builder
	b.WriteString(left + strings.Repeat(" ", gap) + right)
	b.WriteString("\n")
	b.WriteString(breadcrumb(m.crumbParts()...))
	b.WriteString("\n")
	b.WriteString(ruleStyle.Render(strings.Repeat("─", width)))
	return b.String()
}

func (m Model) crumbParts() []string {
	var parts []string
	switch m.Screen {
	case "menu":
		return []string{"Panel de control"}
	case "projects", "create-project", "edit-project", "project-id", "project-result":
		parts = append(parts, "Proyectos")
		switch m.Screen {
		case "create-project":
			parts = append(parts, "Crear")
		case "edit-project":
			parts = append(parts, "Modificar")
		case "project-id":
			parts = append(parts, "Identificador")
		case "project-result":
			parts = append(parts, "Resultado")
		}
	case "project-picker":
		switch {
		case m.pickerReturn == "backlog":
			parts = append(parts, "Backlog")
		case m.pickerReturn == "sprints" || strings.HasPrefix(m.pickerPurpose, "sprint-"):
			parts = append(parts, "Sprints")
		default:
			parts = append(parts, "Proyectos")
		}
		parts = append(parts, "Elegir proyecto")
	case "backlog", "backlog-form", "backlog-result":
		parts = append(parts, "Backlog")
		switch m.Screen {
		case "backlog-form":
			parts = append(parts, "Historia")
		case "backlog-result":
			parts = append(parts, "Resultado")
		}
	case "sprints", "sprint-goal", "sprint-open-list", "sprint-story-select", "sprint-assigned-list", "sprint-task-form", "sprint-history", "sprint-result":
		parts = append(parts, "Sprints")
		switch m.Screen {
		case "sprint-goal":
			parts = append(parts, "Nuevo Sprint")
		case "sprint-open-list":
			parts = append(parts, "Elegir Sprint")
		case "sprint-story-select":
			parts = append(parts, "Asignar historias")
		case "sprint-assigned-list":
			parts = append(parts, "Historias")
		case "sprint-task-form":
			parts = append(parts, "Nueva tarea")
		case "sprint-history":
			parts = append(parts, "Historial")
		case "sprint-result":
			parts = append(parts, "Resultado")
		}
	default:
		parts = append(parts, "")
	}
	if m.activeProjectID != "" && !isProjectScreen(m.Screen) {
		name := m.activeProjectName
		if name == "" {
			name = shortID(m.activeProjectID)
		}
		parts = append(parts, "Proyecto: "+name)
	}
	return parts
}

func isProjectScreen(screen string) bool {
	switch screen {
	case "projects", "project-picker", "create-project", "edit-project", "project-id", "project-result":
		return true
	}
	return false
}

func (m Model) footer() string {
	if m.showHelp {
		return keyHints("Esc", "cerrar ayuda", "?", "cerrar", "Ctrl+C", "salir")
	}
	switch m.Screen {
	case "menu":
		return keyHints("↑/↓", "mover", "Enter", "elegir", "?", "ayuda", "q", "salir")
	case "projects":
		return keyHints("↑/↓", "mover", "Enter", "elegir", "Esc", "volver", "q", "salir")
	case "project-picker":
		return keyHints("escribí", "para filtrar", "↑/↓", "mover", "Enter", "elegir", "Esc", "volver")
	case "create-project", "edit-project", "project-id":
		return keyHints("Tab/↑/↓", "campo", "Enter", "continuar", "Esc", "volver", "Ctrl+C", "salir")
	case "project-result":
		return keyHints("Enter/Esc", "volver", "q", "salir")
	case "backlog":
		return keyHints("↑/↓", "mover", "Enter", "elegir", "/", "buscar", "r", "refrescar", "Esc", "menú", "q", "salir")
	case "backlog-form":
		return keyHints("Tab/↑/↓", "campo", "Enter", "siguiente/enviar", "Esc", "volver", "Ctrl+C", "salir")
	case "backlog-result", "sprint-result":
		return keyHints("Enter/Esc", "volver", "q", "salir")
	case "sprints":
		return keyHints("↑/↓", "mover", "Enter", "elegir", "Esc", "menú", "q", "salir")
	case "sprint-goal":
		return keyHints("Enter", "enviar", "Esc", "volver", "Ctrl+C", "salir")
	case "sprint-open-list", "sprint-assigned-list", "sprint-story-select":
		return keyHints("↑/↓", "mover", "Enter", "elegir", "Space", "marcar", "Esc", "volver", "q", "salir")
	case "sprint-task-form":
		return keyHints("Tab", "campo", "Enter", "continuar/enviar", "Esc", "volver", "Ctrl+C", "salir")
	case "sprint-history":
		return keyHints("Enter/Esc", "volver", "q", "salir")
	default:
		return keyHints("Esc", "volver", "?", "ayuda", "q", "salir")
	}
}

func (m Model) viewHelp() string {
	rows := [][2]string{
		{"↑ / ↓  o  j / k", "Mover la selección"},
		{"Enter", "Elegir / confirmar"},
		{"Space", "Marcar historia (multiselección)"},
		{"/", "Buscar o filtrar en una lista"},
		{"r", "Refrescar el listado actual"},
		{"p", "Cambiar el proyecto activo"},
		{"Esc", "Volver un nivel (nunca pierde el contexto)"},
		{"Tab / ↑ / ↓", "Cambiar de campo en un formulario"},
		{"?", "Mostrar u ocultar esta ayuda"},
		{"q", "Salir (no dentro de un campo de texto)"},
		{"Ctrl+C", "Salir desde cualquier pantalla"},
	}
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Atajos de teclado"))
	b.WriteString("\n\n")
	for _, row := range rows {
		b.WriteString("  " + helpKeyStyle.Render(padRight(row[0], 18)) + hintStyle.Render(row[1]) + "\n")
	}
	return b.String()
}

func padRight(s string, width int) string {
	if w := lipgloss.Width(s); w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-lipgloss.Width(s))
}

func (m Model) contentWidth() int {
	if m.width > 0 {
		return m.width
	}
	return 80
}
