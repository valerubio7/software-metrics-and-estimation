package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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

	// Project browser (the searchable project list).
	projects         []ProjectSummary
	projectsLoading  bool
	projectFilter    string
	projectFiltering bool
	projectCursor    int
	pickerReturn     string
	pickerPurpose    string

	projectID        string
	formField        int
	formSegment      int
	form             ProjectInput
	formStart        dateField
	formFinish       dateField
	formErr          string
	formReturn       string
	projectResultOK  bool
	resultFormScreen string
	message          string
	client           *ProjectClient

	backlogCursor        int
	backlogStories       []Story
	backlogLoading       bool
	backlogFilter        string
	backlogFiltering     bool
	backlogForm          storyForm
	backlogMessage       string
	backlogReturn        string
	backlogClient        *BacklogClient
	storyPointsClient    *StoryPointsClient
	sprintClient         *SprintClient
	metricsClient        *MetricsClient
	sprintCursor         int
	historyCursor        int
	historyExpanded      string
	sprintGoal           string
	sprintMessage        string
	sprintOK             bool
	sprintReturn         string
	sprintFilter         string
	sprintFiltering      bool
	sprintLoading        bool
	sprintHistoryLoading bool
	sprintStoriesLoading bool

	selectedSprintID  string
	taskStoryID       string
	taskTitle         string
	taskHours         string
	taskField         int
	taskClient        *TaskClient
	memberClient      *MemberClient
	section           sectionState
	openSprints       []Sprint
	sprintHistory     SprintHistory
	sprintStories     []Story
	selectedStories   map[string]bool
	pokerParticipants []selectorItem
	pokerVotes        map[string]string
	pokerCursor       int
	pokerPhase        string
	pokerEstimate     int
	pokerMessage      string
	pokerStory        Story

	effort effortState
	now    func() time.Time
}

// WithClock returns a copy of the model that reads the current time from now.
// It is the test seam for the default date of the effort form.
func (m Model) WithClock(now func() time.Time) Model {
	m.now = now
	return m
}

// today returns the current UTC date as YYYY-MM-DD.
func (m Model) today() string {
	now := m.now
	if now == nil {
		now = time.Now
	}
	return now().UTC().Format("2006-01-02")
}

// NewModel creates the base navigation model.
func NewModel(apiURL string) Model {
	return Model{
		APIURL: apiURL,
		Choices: []string{
			"Proyectos", "Backlog", "Sprints", "Planning Poker",
			"Esfuerzo", "Defectos", "Métricas", "Dashboard", "Salir",
		},
		Screen: "menu",
		client: NewProjectClient(apiURL),

		backlogClient:     NewBacklogClient(apiURL),
		storyPointsClient: NewStoryPointsClient(apiURL), sprintClient: NewSprintClient(apiURL),
		metricsClient: NewMetricsClient(apiURL),
		taskClient:    NewTaskClient(apiURL),
		memberClient:  NewMemberClient(apiURL),
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
		return m.applySprintResult(result)
	}
	if result, ok := msg.(SectionResultMsg); ok {
		return m.applySectionResult(result)
	}
	if result, ok := msg.(pokerSaveResult); ok {
		return m.applyPokerSave(result)
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
	if key.Type == tea.KeyRunes && string(key.Runes) == "?" && !entersText(m.Screen) && !m.filtering() && !(m.Screen == "planning-poker" && m.pokerPhase == "vote") {
		m.showHelp = true
		return m, nil
	}
	if key.Type == tea.KeyRunes && string(key.Runes) == "p" && quitsOnQ(m.Screen) && m.Screen != "project-picker" && m.Screen != "sprint-confirm" && !m.filtering() {
		return m.openProjectPicker("project")
	}
	if isQuitRune(key) && quitsOnQ(m.Screen) && !m.filtering() {
		return m, tea.Quit
	}
	switch m.Screen {
	case "menu":
		return m.updateMenu(key)
	case "project-picker":
		return m.updateProjectPicker(key)
	case "create-project", "edit-project":
		return m.updateProjectForm(key)
	case "project-result":
		if key.Type == tea.KeyEsc || key.Type == tea.KeyEnter {
			return m.leaveProjectResult()
		}
		return m, nil
	case "backlog":
		return m.updateBacklog(key)
	case "backlog-form":
		return m.updateBacklogForm(key)
	case "backlog-result":
		return m.updateBacklogResult(key)
	case "sprints":
		return m.updateSprints(key)
	case "sprint-goal":
		return m.updateSprintForm(key)
	case "sprint-history":
		return m.updateSprintHistory(key)
	case "sprint-confirm":
		return m.updateSprintConfirm(key)
	case "sprint-story-select":
		return m.updateSprintStorySelection(key)
	case "sprint-assigned-list":
		return m.updateAssignedStories(key)
	case "sprint-task-form":
		return m.updateSprintTaskForm(key)
	case "sprint-result":
		if key.Type == tea.KeyEsc || key.Type == tea.KeyEnter {
			m.sprintMessage = ""
			if m.sprintReturn == "sprint-assigned-list" && m.selectedSprintID != "" {
				return m.openAssignedStories(m.selectedSprintID)
			}
			return m.enterSprints()
		}
		return m, nil
	case "section":
		return m.updateSection(key)
	case "planning-poker":
		return m.updatePlanningPoker(key)
	case "effort-form":
		return m.updateEffortForm(key)
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
	case "create-project", "edit-project", "backlog-form", "sprint-goal", "sprint-task-form", "effort-form":
		return true
	}
	return false
}

// quitsOnQ reports whether q/Q exits from screen. Text-entry screens are not
// listed, so q is a plain character there; Ctrl+C always quits.
func quitsOnQ(screen string) bool {
	switch screen {
	case "menu", "project-picker", "project-result", "backlog", "backlog-result", "sprints", "sprint-history", "sprint-result", "sprint-confirm", "sprint-assigned-list", "sprint-story-select", "section", "planning-poker":
		return true
	}
	return false
}

func isQuitRune(key tea.KeyMsg) bool {
	return key.Type == tea.KeyRunes && len(key.Runes) == 1 && (key.Runes[0] == 'q' || key.Runes[0] == 'Q')
}

// filtering reports whether a search box owns the keyboard, so global
// shortcuts (?, p, q) must be typed into it instead of triggering.
func (m Model) filtering() bool {
	return (m.Screen == "backlog" && m.backlogFiltering) ||
		(m.Screen == "project-picker" && m.projectFiltering) ||
		(m.Screen == "section" && m.section.selector.filtering) ||
		(isSprintScreen(m.Screen) && m.sprintFiltering)
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
			return m.openProjectPicker("projects")
		case 1:
			return m.openBacklog()
		case 2:
			return m.enterSprints()
		case 3:
			return m.openSection(sectionPlanningPoker)
		case 4:
			return m.openSection(sectionEffort)
		case 5:
			return m.openSection(sectionDefects)
		case 6:
			return m.openSection(sectionMetrics)
		case 7:
			return m.openSection(sectionDashboard)
		default:
			return m, tea.Quit
		}
	}
	return m, nil
}

// projectFormFields is the number of rows in the create/edit project form:
// name, start date and planned finish date.
const projectFormFields = 3

// resetProjectForm clears every field of the create/edit project form.
func (m *Model) resetProjectForm() {
	m.form = ProjectInput{}
	m.formStart = dateField{}
	m.formFinish = dateField{}
	m.formField, m.formSegment, m.formErr = 0, 0, ""
}

// openProjectForm enters the project form. "edit" prefills it from the summary
// chosen in the browser; "create" starts empty. Esc and a finished request
// return to the browser.
func (m Model) openProjectForm(mode string, summary ProjectSummary) (Model, tea.Cmd) {
	m.resetProjectForm()
	m.formReturn = "project-picker"
	m.projectID = summary.ID
	if mode == "edit" {
		m.form.Name = summary.Name
		m.formStart = dateFieldFromISO(summary.StartDate)
		m.formFinish = dateFieldFromISO(summary.PlannedFinishDate)
		m.Screen = "edit-project"
	} else {
		m.Screen = "create-project"
	}
	return m, nil
}

// editSelectedProject opens the form for the project highlighted in the browser.
func (m Model) editSelectedProject() (Model, tea.Cmd) {
	list := m.filteredProjects()
	if m.projectCursor < 0 || m.projectCursor >= len(list) {
		return m, nil
	}
	return m.openProjectForm("edit", list[m.projectCursor])
}

// leaveProjectResult returns to the form on a failed request so it can be
// corrected, or to the freshly reloaded browser on success.
func (m Model) leaveProjectResult() (Model, tea.Cmd) {
	if m.projectResultOK {
		m.message = ""
		return m.reopenProjectPicker()
	}
	m.formErr = strings.TrimSpace(m.message)
	m.message = ""
	m.Screen = m.resultFormScreen
	if m.Screen == "" {
		m.Screen = "project-picker"
	}
	return m, nil
}

// reopenProjectPicker reloads the project list, used after a create or edit so
// the browser shows the change.
func (m Model) reopenProjectPicker() (Model, tea.Cmd) {
	m.Screen = "project-picker"
	m.projectFilter, m.projectFiltering, m.projectCursor = "", false, 0
	m.projectsLoading = true
	return m, m.listProjectsCmd()
}

func (m Model) updateProjectForm(key tea.KeyMsg) (Model, tea.Cmd) {
	if key.Type == tea.KeyEsc {
		m.formErr = ""
		m.Screen = m.formReturn
		if m.Screen == "" {
			m.Screen = "menu"
		}
		return m, nil
	}
	if m.formField == 0 {
		if value, ok := editText(m.form.Name, key); ok {
			m.form.Name, m.formErr = value, ""
			return m, nil
		}
		return m.navigateProjectForm(key)
	}
	return m.updateProjectDate(key)
}

// updateProjectDate drives the segmented date fields: digits fill the active
// segment (advancing when full), backspace erases, and left/right move between
// year, month and day.
func (m Model) updateProjectDate(key tea.KeyMsg) (Model, tea.Cmd) {
	date := m.formFinish
	if m.formField == 1 {
		date = m.formStart
	}
	date, segment, handled := date.applyKey(m.formSegment, key)
	if !handled {
		return m.navigateProjectForm(key)
	}
	m.formSegment = segment
	if key.Type == tea.KeyBackspace || key.Type == tea.KeyRunes {
		m.formErr = ""
	}
	if m.formField == 1 {
		m.formStart = date
	} else {
		m.formFinish = date
	}
	return m, nil
}

func (m Model) navigateProjectForm(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyTab, tea.KeyDown:
		m.formField = (m.formField + 1) % projectFormFields
		m.formSegment = 0
	case tea.KeyShiftTab, tea.KeyUp:
		m.formField = (m.formField + projectFormFields - 1) % projectFormFields
		m.formSegment = 0
	case tea.KeyEnter:
		if m.formField < projectFormFields-1 {
			m.formField++
			m.formSegment = 0
			return m, nil
		}
		return m.submitProjectForm()
	}
	return m, nil
}

// submitProjectForm validates the segmented dates locally before sending, so an
// incomplete or impossible date is corrected in place instead of round-tripping
// to the API.
func (m Model) submitProjectForm() (Model, tea.Cmd) {
	start, startErr := m.formStart.validate()
	if startErr != "" {
		m.formErr = "La fecha de inicio " + startErr + "."
		return m, nil
	}
	finish, finishErr := m.formFinish.validate()
	if finishErr != "" {
		m.formErr = "La fecha fin planificada " + finishErr + "."
		return m, nil
	}
	if finish < start {
		m.formErr = "La fecha fin planificada debe ser igual o posterior a la de inicio."
		return m, nil
	}
	m.formErr = ""
	input := ProjectInput{Name: m.form.Name, StartDate: start, PlannedFinishDate: finish}
	if m.Screen == "create-project" {
		return m, m.createCmd(input)
	}
	return m, m.updateCmd(m.projectID, input)
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

func (m Model) applyProjectResult(result ProjectResultMsg) (Model, tea.Cmd) {
	m.resultFormScreen = m.Screen
	m.Screen = "project-result"
	m.projectResultOK = false
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
	m.projectResultOK = true
	var status struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if json.Unmarshal(result.Response.Body, &status) == nil && status.Name != "" {
		m.message = fmt.Sprintf("Proyecto %s: %s", status.Name, status.ID)
	} else {
		m.message = strings.TrimSpace(string(result.Response.Body))
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	var content strings.Builder
	if m.showHelp {
		content.WriteString(m.viewHelp())
	} else {
		if m.APIErr != nil {
			content.WriteString(m.errorBlock("API no disponible: " + m.APIErr.Error()))
			content.WriteString("\n")
			content.WriteString(errorHintStyle.Render("Revisá API_URL o levantá la API; la navegación sigue disponible."))
			content.WriteString("\n\n")
		}
		content.WriteString(m.viewBody())
	}
	var b strings.Builder
	b.WriteString(m.chrome())
	b.WriteString("\n")
	b.WriteString(m.card(content.String()))
	b.WriteString("\n")
	b.WriteString(m.footer())
	return b.String()
}
func (m Model) viewBody() string {
	switch m.Screen {
	case "menu":
		return m.viewMenu()
	case "project-picker":
		return m.viewProjectPicker()
	case "create-project", "edit-project":
		return m.viewProjectForm()
	case "project-result":
		return m.viewProjectResult()
	case "backlog":
		return m.viewBacklog()
	case "backlog-form":
		return m.viewBacklogForm()
	case "backlog-result":
		return m.viewBacklogResult()
	case "sprints":
		return m.viewSprints()
	case "sprint-goal":
		return m.viewSprintGoal()
	case "sprint-history":
		return m.viewSprintHistory()
	case "sprint-confirm":
		return m.viewSprintConfirm()
	case "sprint-story-select":
		return m.viewSprintStorySelect()
	case "sprint-assigned-list":
		return m.viewAssignedStories()
	case "sprint-task-form":
		return m.viewSprintTaskForm()
	case "sprint-result":
		return m.viewSprintResult()
	case "section":
		return m.viewSection()
	case "planning-poker":
		return m.viewPlanningPoker()
	case "effort-form":
		return m.viewEffortForm()
	default:
		return screenTitleStyle.Render(m.Choices[m.Cursor])
	}
}

func (m Model) viewMenu() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Panel de control"))
	b.WriteString("\n")
	context := "Sin proyecto activo · elegí uno en Proyectos"
	if m.activeProjectID != "" {
		context = "Proyecto activo: " + m.activeProjectName
	}
	b.WriteString(hintStyle.Render(context))
	b.WriteString("\n\n")
	for i, choice := range m.Choices {
		b.WriteString(m.row(i == m.Cursor, choice) + "\n")
	}
	return b.String()
}

func (m Model) viewProjectForm() string {
	title := "Crear proyecto"
	if m.Screen == "edit-project" {
		title = "Modificar proyecto"
	}
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render(title))
	b.WriteString("\n")
	if m.Screen == "edit-project" {
		b.WriteString(hintStyle.Render("ID: " + shortID(m.projectID)))
	} else {
		b.WriteString(hintStyle.Render("Completá los datos del nuevo proyecto."))
	}
	b.WriteString("\n\n")
	b.WriteString(m.projectTextRow(0, "Nombre", m.form.Name) + "\n")
	startLabel, finishLabel := "Fecha inicio", "Fecha fin planificada"
	if m.projectFieldWidth() < projectLabelWidth {
		startLabel, finishLabel = "Inicio", "Fin"
	}
	b.WriteString(m.projectDateRow(1, startLabel, m.formStart) + "\n")
	b.WriteString(m.projectDateRow(2, finishLabel, m.formFinish) + "\n")
	if m.formErr != "" {
		b.WriteString("\n")
		b.WriteString(m.errorBlock(m.formErr))
	}
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Tab/↑/↓ cambia de campo · ←/→ mueve entre año, mes y día."))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Enter en el último campo envía · Esc vuelve."))
	return b.String()
}

func (m Model) viewProjectResult() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Resultado"))
	b.WriteString("\n\n")
	b.WriteString(m.resultBlock(m.message, m.projectResultOK))
	return b.String()
}

// chrome renders the persistent top bar: a solid app bar with the brand and
// the connection state, followed by the breadcrumb trail and a rule.
func (m Model) chrome() string {
	width := m.contentWidth()
	status, statusStyle := "● API conectada", appBarOkStyle
	if m.APIErr != nil {
		status, statusStyle = "● API caída", appBarDangerStyle
	}
	var b strings.Builder
	b.WriteString(m.appBar(statusStyle, status))
	b.WriteString("\n")
	b.WriteString(ansi.Truncate(breadcrumb(m.crumbParts()...), width, "…"))
	b.WriteString("\n")
	b.WriteString(ruleStyle.Render(strings.Repeat("─", width)))
	return b.String()
}

// appBar renders the solid top band. On narrow terminals it drops the brand
// text (keeping the marker) so the connection badge always stays visible.
func (m Model) appBar(statusStyle lipgloss.Style, status string) string {
	width := m.contentWidth()
	right := statusStyle.Render(" " + status + " ")
	rw := lipgloss.Width(right)
	if width <= rw {
		return statusStyle.Render(ansi.Truncate(" "+status+" ", width, "…"))
	}
	brand := " ◆ Software Metrics & Estimation "
	left := appBarBrandStyle.Render(brand)
	if lipgloss.Width(left)+rw+1 > width {
		left = appBarBrandStyle.Render(" ◆ ")
	}
	gap := width - lipgloss.Width(left) - rw
	if gap < 0 {
		gap = 0
	}
	return left + appBarFillStyle.Render(strings.Repeat(" ", gap)) + right
}

func (m Model) crumbParts() []string {
	var parts []string
	switch m.Screen {
	case "menu":
		return []string{"Panel de control"}
	case "create-project", "edit-project", "project-result":
		parts = append(parts, "Proyectos")
		switch m.Screen {
		case "create-project":
			parts = append(parts, "Crear")
		case "edit-project":
			parts = append(parts, "Modificar")
		case "project-result":
			parts = append(parts, "Resultado")
		}
	case "project-picker":
		switch {
		case m.pickerReturn == "backlog":
			parts = append(parts, "Backlog")
		case m.pickerReturn == "sprints" || isSprintScreen(m.pickerReturn) || m.pickerPurpose == "sprints":
			parts = append(parts, "Sprints")
		case m.pickerReturn == "section" || strings.HasPrefix(m.pickerPurpose, "section:"):
			parts = append(parts, m.section.kind.title())
		default:
			parts = append(parts, "Proyectos")
		}
		if m.pickerPurpose != "projects" {
			parts = append(parts, "Elegir proyecto")
		}
	case "backlog", "backlog-form", "backlog-result":
		parts = append(parts, "Backlog")
		switch m.Screen {
		case "backlog-form":
			parts = append(parts, "Historia")
		case "backlog-result":
			parts = append(parts, "Resultado")
		}
	case "sprints", "sprint-goal", "sprint-story-select", "sprint-assigned-list", "sprint-task-form", "sprint-history", "sprint-confirm", "sprint-result":
		parts = append(parts, "Sprints")
		switch m.Screen {
		case "sprint-goal":
			parts = append(parts, "Nuevo Sprint")
		case "sprint-story-select":
			parts = append(parts, "Asignar historias")
		case "sprint-assigned-list":
			parts = append(parts, "Historias")
		case "sprint-task-form":
			parts = append(parts, "Nueva tarea")
		case "sprint-history":
			parts = append(parts, "Historial")
		case "sprint-confirm":
			parts = append(parts, "Cerrar")
		case "sprint-result":
			parts = append(parts, "Resultado")
		}
	case "section", "planning-poker":
		parts = append(parts, m.section.kind.title())
		switch m.section.step {
		case sectionStepMembers:
			if m.section.kind == sectionEffort {
				parts = append(parts, "Integrante")
			} else {
				parts = append(parts, "Participantes")
			}
		case sectionStepTasks:
			parts = append(parts, "Tareas")
		case sectionStepSprints:
			parts = append(parts, "Sprint")
		case sectionStepNotice:
			parts = append(parts, "Próximamente")
		}
	case "effort-form":
		parts = append(parts, sectionEffort.title(), "Registrar")
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
	case "project-picker", "create-project", "edit-project", "project-result":
		return true
	}
	return false
}

func (m Model) footer() string {
	if m.showHelp {
		return m.keyHints("Esc", "cerrar ayuda", "?", "cerrar", "Ctrl+C", "salir")
	}
	switch m.Screen {
	case "menu":
		return m.keyHints("↑/↓", "mover", "Enter", "elegir", "?", "ayuda", "q", "salir")
	case "project-picker":
		if m.projectFiltering {
			return m.keyHints("escribí", "para filtrar", "↑/↓", "mover", "Enter", "elegir", "Esc", "limpiar")
		}
		return m.keyHints("↑/↓", "mover", "Enter", "usar", "e", "editar", "n", "crear", "r", "refrescar", "/", "buscar", "Esc", "volver")
	case "create-project", "edit-project":
		return m.keyHints("Tab/↑/↓", "campo", "Enter", "continuar", "Esc", "volver", "Ctrl+C", "salir")
	case "project-result":
		return m.keyHints("Enter/Esc", "volver", "q", "salir")
	case "backlog":
		if m.backlogFiltering {
			return m.keyHints("escribí", "para filtrar", "↑/↓", "mover", "Enter", "dejar de filtrar", "Esc", "limpiar")
		}
		return m.keyHints("↑/↓", "mover", "Enter", "editar", "n", "crear", "c", "proyecto", "r", "refrescar", "/", "buscar", "Esc", "menú", "q", "salir")
	case "backlog-form":
		return m.keyHints("Tab/↑/↓", "campo", "Enter", "siguiente/enviar", "Esc", "volver", "Ctrl+C", "salir")
	case "backlog-result", "sprint-result":
		return m.keyHints("Enter/Esc", "volver", "q", "salir")
	case "sprints":
		if m.sprintFiltering {
			return m.keyHints("escribí", "para filtrar", "↑/↓", "mover", "Enter", "dejar de filtrar", "Esc", "limpiar")
		}
		return m.keyHints("↑/↓", "mover", "Enter", "historias", "a", "asignar", "x", "cerrar", "h", "cerrados", "n", "nuevo", "r", "refrescar", "/", "buscar", "Esc", "menú", "q", "salir")
	case "sprint-goal":
		return m.keyHints("Enter", "enviar", "Esc", "volver", "Ctrl+C", "salir")
	case "sprint-confirm":
		return m.keyHints("s", "confirmar", "n/Esc", "cancelar", "q", "salir")
	case "sprint-assigned-list":
		if m.sprintFiltering {
			return m.keyHints("escribí", "para filtrar", "↑/↓", "mover", "Enter", "dejar de filtrar", "Esc", "limpiar")
		}
		return m.keyHints("↑/↓", "mover", "Enter", "tareas", "c", "completar", "r", "refrescar", "/", "buscar", "Esc", "volver", "q", "salir")
	case "sprint-story-select":
		if m.sprintFiltering {
			return m.keyHints("escribí", "para filtrar", "↑/↓", "mover", "Enter", "dejar de filtrar", "Esc", "limpiar")
		}
		return m.keyHints("↑/↓", "mover", "Space", "marcar", "Enter", "asignar", "/", "buscar", "Esc", "volver", "q", "salir")
	case "sprint-task-form":
		return m.keyHints("Tab", "campo", "Enter", "continuar/enviar", "Esc", "volver", "Ctrl+C", "salir")
	case "sprint-history":
		if m.sprintFiltering {
			return m.keyHints("escribí", "para filtrar", "↑/↓", "mover", "Enter", "dejar de filtrar", "Esc", "limpiar")
		}
		return m.keyHints("↑/↓", "mover", "Enter", "desplegar", "r", "refrescar", "/", "buscar", "Esc", "volver", "q", "salir")
	case "section":
		return m.sectionFooter()
	case "planning-poker":
		return m.planningPokerFooter()
	case "effort-form":
		return m.effortFormFooter()
	default:
		return m.keyHints("Esc", "volver", "?", "ayuda", "q", "salir")
	}
}

func (m Model) viewHelp() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Atajos de teclado"))
	b.WriteString("\n\n")
	for _, row := range m.helpRows() {
		b.WriteString("  " + helpKeyStyle.Render(padRight(row[0], 18)) + hintStyle.Render(row[1]) + "\n")
	}
	return b.String()
}

// helpRows returns the keys that are actually available on the active screen.
// Sections get their own list so the overlay never advertises actions that do
// not exist there.
func (m Model) helpRows() [][2]string {
	if m.Screen == "planning-poker" && m.pokerPhase == "vote" {
		return [][2]string{{"0–100 / ?", "Registrar el voto privado"}, {"Enter", "Confirmar voto y pasar al siguiente participante"}, {"Esc", "Volver"}, {"Ctrl+C", "Salir desde cualquier pantalla"}}
	}
	if m.Screen == "section" {
		return m.sectionHelpRows()
	}
	return [][2]string{
		{"↑ / ↓", "Mover la selección"},
		{"Enter", "Elegir / confirmar la fila resaltada"},
		{"n", "Crear un elemento nuevo"},
		{"e", "Editar el elemento resaltado"},
		{"Space", "Marcar historias (multiselección)"},
		{"/", "Buscar o filtrar en una lista"},
		{"r", "Refrescar el listado actual"},
		{"x", "Cerrar el Sprint resaltado (pide confirmación)"},
		{"h", "Ver los Sprints cerrados"},
		{"p", "Cambiar el proyecto activo"},
		{"Esc", "Limpiar el filtro y volver un nivel (nunca pierde el contexto)"},
		{"Tab / ↑ / ↓", "Cambiar de campo en un formulario"},
		{"?", "Mostrar u ocultar esta ayuda"},
		{"q", "Salir (no dentro de un campo de texto)"},
		{"Ctrl+C", "Salir desde cualquier pantalla"},
	}
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

// innerWidth is the usable width inside the content panel (borders and
// horizontal padding subtracted), used by rows and form fields so their
// selection bar fills the panel.
func (m Model) innerWidth() int {
	if w := m.contentWidth() - 4; w > 10 {
		return w
	}
	return 10
}

// card frames the body in a rounded panel sized to the content width. Plain
// text wraps to the inner width; already-styled lines are truncated as a
// safety net so a stray long row cannot widen the panel.
func (m Model) card(body string) string {
	inner := m.innerWidth()
	var lines []string
	for _, line := range strings.Split(body, "\n") {
		if ansiPattern.MatchString(line) {
			lines = append(lines, padToWidth(ansi.Truncate(line, inner, "…"), inner))
			continue
		}
		for _, wrapped := range wrapPlain(line, inner) {
			lines = append(lines, padToWidth(wrapped, inner))
		}
	}
	return panelStyle.Width(m.contentWidth() - 2).Render(strings.Join(lines, "\n"))
}

// padToWidth right-pads s with spaces to width using its display width.
func padToWidth(s string, width int) string {
	if pad := width - lipgloss.Width(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}
