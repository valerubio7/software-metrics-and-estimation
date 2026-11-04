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
	APIURL string
	Screen string
	APIErr error

	width, height int
	showHelp      bool
	toast         string // last success notice, cleared by the next key

	// Session context shared by every screen.
	activeProjectID     string
	activeProjectName   string
	activeProjectStatus string

	// Project browser (the searchable project list).
	projects         []ProjectSummary
	projectsLoading  bool
	projectFilter    string
	projectFiltering bool
	projectCursor    int
	autoOpen         bool // open the only project on the first load

	projectID     string
	formField     int
	formSegment   int
	form          ProjectInput
	formStart     dateField
	formFinish    dateField
	formErr       string
	formReturn    string
	projectSaving bool
	message       string
	client        *ProjectClient

	backlogCursor        int
	backlogStories       []Story
	backlogLoading       bool
	backlogFilter        string
	backlogFiltering     bool
	backlogForm          storyForm
	backlogMessage       string
	storyDetail          storyDetailState
	returnTo             returnTarget // where the open story flow goes back to
	backlogClient        *BacklogClient
	storyPointsClient    *StoryPointsClient
	sprintClient         *SprintClient
	metricsClient        *MetricsClient
	reportClient         *ReportClient
	report               reportState
	sprintCursor         int
	historyCursor        int
	historyExpanded      string
	sprintGoal           string
	sprintMessage        string
	sprintSaving         bool
	sprintSavingLabel    string
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

	dashboard             dashboardState
	effortClient          *EffortClient
	effort                effortState
	defectClient          *DefectClient
	defects               []Defect
	defectCursor          int
	defectFilter          string
	defectFiltering       bool
	defectLoading         bool
	defectError           string
	defectDetail          Defect
	defectCreateForm      defectCreateForm
	defectStorySelector   selector
	defectSprintSelector  selector
	defectCreateLoading   bool
	defectCreateMessage   string
	defectRequestSequence uint64
	defectStoryRequestID  uint64
	defectSprintRequestID uint64
	defectCreateRequestID uint64

	defectStatusIndex            int
	defectResolutionSelector     selector
	defectStatusOptionsLoading   bool
	defectStatusOptionsRequestID uint64
	defectStatusLoading          bool
	defectStatusRequestID        uint64
	defectStatusMessage          string
	defectStatusPending          string
	now                          func() time.Time
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
		APIURL:          apiURL,
		Screen:          "project-picker",
		projectsLoading: true,
		autoOpen:        true,
		client:          NewProjectClient(apiURL),

		backlogClient:     NewBacklogClient(apiURL),
		storyPointsClient: NewStoryPointsClient(apiURL), sprintClient: NewSprintClient(apiURL),
		metricsClient: NewMetricsClient(apiURL),
		reportClient:  NewReportClient(apiURL),
		taskClient:    NewTaskClient(apiURL),
		memberClient:  NewMemberClient(apiURL),
		effortClient:  NewEffortClient(apiURL),
		defectClient:  NewDefectClient(apiURL),
	}
}

// Init implements tea.Model: the TUI starts loading the project browser.
func (m Model) Init() tea.Cmd { return m.listProjectsCmd() }

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
	if result, ok := msg.(effortSaveResult); ok {
		return m.applyEffortSave(result)
	}
	if result, ok := msg.(dashboardResultMsg); ok {
		return m.applyDashboardResult(result)
	}
	if result, ok := msg.(dashboardScopesMsg); ok {
		return m.applyDashboardScopes(result)
	}
	if result, ok := msg.(ReportResultMsg); ok {
		return m.applyReportResult(result)
	}
	if result, ok := msg.(DefectResultMsg); ok {
		return m.applyDefectResult(result)
	}
	if result, ok := msg.(defectOptionsResult); ok {
		return m.applyDefectOptionsResult(result)
	}
	if result, ok := msg.(defectCreateResult); ok {
		return m.applyDefectCreateResult(result)
	}
	if result, ok := msg.(defectStatusOptionsResult); ok {
		return m.applyDefectStatusOptionsResult(result)
	}
	if result, ok := msg.(defectStatusResult); ok {
		return m.applyDefectStatusResult(result)
	}
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return m, nil
	}
	m.toast = ""
	if key.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}
	if m.showHelp {
		if key.Type == tea.KeyEsc || key.Type == tea.KeyEnter || (key.Type == tea.KeyRunes && string(key.Runes) == "?") {
			m.showHelp = false
		}
		return m, nil
	}
	if key.Type == tea.KeyRunes && string(key.Runes) == "?" && m.helpAvailable() {
		m.showHelp = true
		return m, nil
	}
	if m.tabRoot() && !m.filtering() {
		if t, ok := tabForKey(key, m.currentTab()); ok {
			return m.openTab(t)
		}
	}
	if isQuitRune(key) && quitsOnQ(m.Screen) && !m.filtering() && !m.defectSubmissionInFlight() {
		return m, tea.Quit
	}
	if m.sprintSaving && isSprintScreen(m.Screen) {
		// A Sprint write is in flight: no key may send it again or leave.
		return m, nil
	}
	switch m.Screen {
	case "project-picker":
		return m.updateProjectPicker(key)
	case "create-project", "edit-project":
		return m.updateProjectForm(key)
	case "backlog":
		return m.updateBacklog(key)
	case "backlog-form":
		return m.updateBacklogForm(key)
	case "story-detail":
		return m.updateStoryDetail(key)
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
	case "section":
		return m.updateSection(key)
	case "dashboard":
		return m.updateDashboard(key)
	case "dashboard-sprint-select":
		return m.updateDashboardScopes(key)
	case "planning-poker":
		return m.updatePlanningPoker(key)
	case "effort-form":
		return m.updateEffortForm(key)
	case "defects":
		return m.updateDefects(key)
	case "defect-detail":
		if key.Type == tea.KeyEsc {
			m.Screen = "defects"
		} else if key.Type == tea.KeyRunes && string(key.Runes) == "u" {
			return m.beginDefectStatusUpdate()
		}
		return m, nil
	case "defect-status-select":
		return m.updateDefectStatusSelect(key)
	case "defect-resolution-select":
		return m.updateDefectResolutionSelect(key)
	case "defect-create-form":
		return m.updateDefectCreateForm(key)
	case "defect-story-select":
		return m.updateDefectStorySelect(key)
	case "defect-sprint-select":
		return m.updateDefectSprintSelect(key)
	case "defect-create-confirm":
		return m.updateDefectCreateConfirm(key)
	default:
		if key.Type == tea.KeyEsc {
			return m.leaveWorkspace()
		}
		return m, nil
	}
}

// entersText reports whether the screen owns a focused text field, where "?" is a
// plain character rather than the help shortcut.
func entersText(screen string) bool {
	switch screen {
	case "create-project", "edit-project", "backlog-form", "sprint-goal", "sprint-task-form", "effort-form", "defect-create-form":
		return true
	}
	return false
}

// quitsOnQ reports whether q/Q exits from screen. Text-entry screens are not
// listed, so q is a plain character there; Ctrl+C always quits.
func quitsOnQ(screen string) bool {
	switch screen {
	case "project-picker", "backlog", "story-detail", "sprints", "sprint-history", "sprint-confirm", "sprint-assigned-list", "sprint-story-select", "section", "dashboard", "dashboard-sprint-select", "planning-poker", "defects", "defect-detail", "defect-story-select", "defect-sprint-select", "defect-create-confirm", "defect-status-select", "defect-resolution-select":
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
		(m.Screen == "defects" && m.defectFiltering) ||
		(m.Screen == "defect-story-select" && m.defectStorySelector.filtering) ||
		(m.Screen == "defect-sprint-select" && m.defectSprintSelector.filtering) ||
		(m.Screen == "defect-resolution-select" && m.defectResolutionSelector.filtering) ||
		(m.Screen == "project-picker" && m.projectFiltering) ||
		(m.Screen == "section" && m.section.selector.filtering) ||
		(m.Screen == "dashboard-sprint-select" && m.dashboard.scopes.filtering) ||
		(isSprintScreen(m.Screen) && m.sprintFiltering)
}

func (m Model) defectSubmissionInFlight() bool {
	return ((m.Screen == "defect-status-select" || m.Screen == "defect-resolution-select") && m.defectStatusLoading) ||
		(m.Screen == "defect-create-confirm" && m.defectCreateLoading)
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

// reopenProjectPicker reloads the project list, used after a create or edit so
// the browser shows the change. It never auto-opens the only project.
func (m Model) reopenProjectPicker() (Model, tea.Cmd) {
	m.Screen = "project-picker"
	m.projectFilter, m.projectFiltering, m.projectCursor = "", false, 0
	m.projectsLoading, m.autoOpen, m.message = true, false, ""
	return m, m.listProjectsCmd()
}

func (m Model) updateProjectForm(key tea.KeyMsg) (Model, tea.Cmd) {
	if m.projectSaving {
		// The request is in flight: no key may send it again or leave the form.
		return m, nil
	}
	if key.Type == tea.KeyEsc {
		m.formErr = ""
		m.Screen = m.formReturn
		if m.Screen == "" {
			m.Screen = "project-picker"
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
	m.formErr, m.projectSaving = "", true
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

// applyProjectResult returns to the reloaded browser with a toast on success.
// A rejection stays on the form with the typed values and the API reason.
func (m Model) applyProjectResult(result ProjectResultMsg) (Model, tea.Cmd) {
	if !m.projectSaving || (m.Screen != "create-project" && m.Screen != "edit-project") {
		return m, nil // an answer nobody is waiting for
	}
	m.projectSaving = false
	if result.Err != nil {
		m.formErr = "Error de conexión: " + result.Err.Error()
		return m, nil
	}
	if result.Response.StatusCode < 200 || result.Response.StatusCode >= 300 {
		detail := strings.TrimSpace(string(result.Response.Body))
		if result.Response.StatusCode == 404 {
			m.formErr = "El proyecto no existe (404): " + detail
		} else {
			m.formErr = fmt.Sprintf("Error HTTP %d: %s", result.Response.StatusCode, detail)
		}
		return m, nil
	}
	name := m.form.Name
	var project struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(result.Response.Body, &project) == nil && project.Name != "" {
		name = project.Name
	}
	verb := "creado"
	if m.Screen == "edit-project" {
		verb = "modificado"
	}
	m.toast = fmt.Sprintf("Proyecto %s: %s", verb, name)
	return m.reopenProjectPicker()
}

// View implements tea.Model.
func (m Model) View() string {
	var content strings.Builder
	if m.showHelp {
		content.WriteString(m.viewHelp())
	} else {
		if m.toast != "" {
			content.WriteString(m.okBlock("✓ " + m.toast))
			content.WriteString("\n\n")
		}
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
	case "project-picker":
		return m.viewProjectPicker()
	case "create-project", "edit-project":
		return m.viewProjectForm()
	case "backlog":
		return m.viewBacklog()
	case "backlog-form":
		return m.viewBacklogForm()
	case "story-detail":
		return m.viewStoryDetail()
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
	case "section":
		return m.viewSection()
	case "dashboard":
		return m.viewDashboard()
	case "dashboard-sprint-select":
		return m.viewDashboardScopes()
	case "planning-poker":
		return m.viewPlanningPoker()
	case "effort-form":
		return m.viewEffortForm()
	case "defects":
		return m.viewDefects()
	case "defect-detail":
		return m.viewDefectDetail()
	case "defect-status-select":
		return m.viewDefectStatusSelect()
	case "defect-resolution-select":
		return strings.TrimRight(m.viewSelector(&m.defectResolutionSelector, "Defecto · Sprint de resolución", "Elegí un Sprint abierto o cerrado del proyecto.", "No hay Sprints disponibles.", "Creá un Sprint desde Sprints y volvé a intentar."), "\n") + m.defectStatusLines()
	case "defect-create-form":
		return m.viewDefectCreateForm()
	case "defect-story-select":
		return m.viewSelector(&m.defectStorySelector, "Defecto · Historia", "Elegí la historia relacionada.", "No hay historias en el backlog.", "Creá una historia desde Backlog y volvé a intentar.")
	case "defect-sprint-select":
		return m.viewSelector(&m.defectSprintSelector, "Defecto · Sprint de detección", "Elegí un Sprint abierto del proyecto.", "No hay Sprints abiertos.", "Creá un Sprint desde Sprints y volvé a intentar.")
	case "defect-create-confirm":
		return m.viewDefectCreateConfirm()
	default:
		return ""
	}
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
	if m.projectSaving {
		b.WriteString("\n")
		b.WriteString(statusMutedStyle.Render("Guardando…"))
		b.WriteString("\n")
	}
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
	if m.inWorkspace() {
		b.WriteString(m.workspaceHeader())
		b.WriteString("\n")
	}
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
	case "create-project", "edit-project":
		parts = append(parts, "Proyectos")
		if m.Screen == "create-project" {
			parts = append(parts, "Crear")
		} else {
			parts = append(parts, "Modificar")
		}
	case "project-picker":
		return []string{"Proyectos"}
	case "backlog", "backlog-form":
		parts = append(parts, "Backlog")
		if m.Screen == "backlog-form" {
			parts = append(parts, "Historia")
		}
	case "story-detail":
		parts = append(parts, "Backlog", "Detalle")
	case "sprints", "sprint-goal", "sprint-story-select", "sprint-assigned-list", "sprint-task-form", "sprint-history", "sprint-confirm":
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
		}
	case "section", "planning-poker":
		parts = append(parts, m.storyFlowCrumb()...)
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
		}
	case "effort-form":
		parts = append(parts, m.storyFlowCrumb()...)
		parts = append(parts, sectionEffort.title(), "Registrar")
	case "dashboard":
		parts = append(parts, "Dashboard")
	case "dashboard-sprint-select":
		parts = append(parts, "Dashboard", "Elegir ámbito")
	case "defects", "defect-detail", "defect-create-form", "defect-story-select", "defect-sprint-select", "defect-create-confirm", "defect-status-select", "defect-resolution-select":
		if prefix := m.storyFlowCrumb(); prefix != nil {
			parts = append(parts, prefix...) // reported from the story detail
		} else {
			parts = append(parts, tabDefects.title())
		}
		switch m.Screen {
		case "defect-detail":
			parts = append(parts, "Detalle")
		case "defect-create-form":
			parts = append(parts, "Registrar defecto")
		case "defect-story-select":
			parts = append(parts, "Elegir historia")
		case "defect-sprint-select":
			parts = append(parts, "Elegir Sprint de detección")
		case "defect-create-confirm":
			parts = append(parts, "Confirmar registro")
		case "defect-status-select":
			parts = append(parts, "Cambiar estado")
		case "defect-resolution-select":
			parts = append(parts, "Elegir Sprint de resolución")
		}
	default:
		parts = append(parts, "")
	}
	if m.inWorkspace() {
		parts = append([]string{"Proyectos", m.projectLabel()}, parts...)
	}
	return parts
}

// storyFlowCrumb is the prefix of the breadcrumb of an action launched from the
// story detail; the defect screens reached from the Defectos tab have none.
func (m Model) storyFlowCrumb() []string {
	if m.inStoryFlow() {
		return []string{"Backlog", "Detalle"}
	}
	return nil
}

func isProjectScreen(screen string) bool {
	switch screen {
	case "project-picker", "create-project", "edit-project":
		return true
	}
	return false
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
