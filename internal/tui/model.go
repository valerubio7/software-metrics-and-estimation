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
	helpOffset    int    // first visible line of the help screen
	toast         string // last success notice, cleared by the next key

	// Pane geometry for the master-detail workspace. Zero means the body uses
	// the whole content area.
	paneWidth  int
	paneHeight int
	// quietSelection draws the secondary list selection with a softer marker,
	// so only the pane the user is acting on owns the bright focus bar.
	quietSelection bool

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
	defectSprintSelector  selector
	defectCreateLoading   bool
	defectCreateMessage   string
	defectRequestSequence uint64
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
	if result, ok := msg.(storyTasksResult); ok {
		return m.applyStoryTasks(result)
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
		switch {
		case key.Type == tea.KeyEsc || key.Type == tea.KeyEnter || (key.Type == tea.KeyRunes && string(key.Runes) == "?"):
			m.showHelp, m.helpOffset = false, 0
		case key.Type == tea.KeyUp:
			if m.helpOffset > 0 {
				m.helpOffset--
			}
		case key.Type == tea.KeyDown:
			m.helpOffset++
		case key.Type == tea.KeyPgUp:
			m.helpOffset -= 5
			if m.helpOffset < 0 {
				m.helpOffset = 0
			}
		case key.Type == tea.KeyPgDown:
			m.helpOffset += 5
		}
		m.clampHelpOffset()
		return m, nil
	}
	if key.Type == tea.KeyRunes && string(key.Runes) == "?" && m.helpAvailable() {
		m.showHelp, m.helpOffset = true, 0
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
	if m.sprintSaving && (isSprintScreen(m.Screen) || (m.Screen == "story-detail" && m.storyDetail.origin == storyDetailFromSprint)) {
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
	case "project-picker", "backlog", "story-detail", "sprints", "sprint-history", "sprint-confirm", "sprint-assigned-list", "sprint-story-select", "section", "dashboard", "dashboard-sprint-select", "planning-poker", "defects", "defect-detail", "defect-sprint-select", "defect-create-confirm", "defect-status-select", "defect-resolution-select":
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

// View implements tea.Model. The shell is fixed: the app bar, the project and
// tab header, a live status line, the body and a wrapped footer. The body
// always fills the terminal so the layout never jumps between screens.
func (m Model) View() string {
	if m.showHelp {
		return m.helpView()
	}
	m.paneWidth, m.paneHeight = 0, m.bodyInnerHeight()
	body := ""
	if m.masterDetail() {
		body = m.masterDetailView()
	} else {
		body = m.card(m.viewBody())
	}
	view := m.headerView() + "\n" + m.statusView() + "\n" + body + "\n" + m.footerView()
	if m.height > 0 {
		view = fitHeight(view, m.height)
	}
	return view
}

// viewBody dispatches the screen body. It never draws the outer frame; the
// shell owns it, so narrow and wide layouts share every screen.
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

// headerLines is how many lines the shell spends above the body and status.
// Wide workspaces merge the breadcrumb and the tabs on one line.
func (m Model) headerLines() int {
	if m.inWorkspace() {
		if m.compactHeader() {
			return 2
		}
		return 3
	}
	if m.Screen == "create-project" || m.Screen == "edit-project" {
		return 2
	}
	return 1
}

// compactHeader reports whether the breadcrumb and tabs fit side by side.
func (m Model) compactHeader() bool {
	return m.contentWidth() >= 100
}

// footerHeight is how many lines the wrapped footer needs.
func (m Model) footerHeight() int {
	lines := strings.Count(m.footer(), "\n") + 1
	if lines < 1 {
		return 1
	}
	if lines > 3 {
		return 3
	}
	return lines
}

// bodyOuterHeight is the height reserved for the framed body, borders included.
func (m Model) bodyOuterHeight() int {
	if m.height <= 0 {
		return 0
	}
	h := m.height - m.headerLines() - 1 - m.footerHeight() // status + footer
	if h < 3 {
		return 3
	}
	return h
}

// bodyInnerHeight is the usable height inside the body frame.
func (m Model) bodyInnerHeight() int {
	outer := m.bodyOuterHeight()
	if outer <= 0 {
		return 0
	}
	if outer < 3 {
		return 1
	}
	return outer - 2
}

// headerView renders the app bar, the project context line (inside a workspace)
// and the tab row. The project form keeps its breadcrumb on the second line.
func (m Model) headerView() string {
	width := m.contentWidth()
	lines := []string{m.appBar()}
	if m.inWorkspace() {
		if m.compactHeader() {
			lines = append(lines, m.breadcrumbTabsLine())
		} else {
			lines = append(lines, m.contextLine(), m.tabBar())
		}
	} else if m.Screen == "create-project" || m.Screen == "edit-project" {
		lines = append(lines, ansi.Truncate(breadcrumb(m.crumbParts()...), width, "…"))
	}
	return strings.Join(lines, "\n")
}

// breadcrumbTabsLine is the wide workspace header: the path on the left and
// the tabs on the right.
func (m Model) breadcrumbTabsLine() string {
	width := m.contentWidth()
	crumbs := breadcrumb(m.crumbParts()...)
	tabs := m.tabsView(true)
	if lipgloss.Width(crumbs)+lipgloss.Width(tabs)+1 > width {
		tabs = m.tabsView(false)
	}
	if lipgloss.Width(crumbs)+lipgloss.Width(tabs)+1 > width {
		parts := m.crumbParts()
		if len(parts) > 1 {
			crumbs = breadcrumb(parts[len(parts)-1])
		}
	}
	available := width - lipgloss.Width(tabs) - 1
	if available < 0 {
		available = 0
	}
	crumbs = ansi.Truncate(crumbs, available, "…")
	gap := width - lipgloss.Width(crumbs) - lipgloss.Width(tabs)
	if gap < 1 {
		gap = 1
	}
	return crumbs + strings.Repeat(" ", gap) + tabs
}

// contextLine shows where the user is and which project is open. The
// breadcrumb is truncated before the project, so the project never disappears.
func (m Model) contextLine() string {
	width := m.contentWidth()
	project := titleStyle.Render("◆ " + m.projectLabel())
	if m.activeProjectStatus != "" {
		project += "  " + projectStatusBadge(m.activeProjectStatus)
	}
	projectWidth := lipgloss.Width(project)
	if projectWidth >= width {
		return ansi.Truncate(project, width, "…")
	}
	crumbs := breadcrumb(m.crumbParts()...)
	available := width - projectWidth - 2
	if available < 8 {
		return project
	}
	crumbs = ansi.Truncate(crumbs, available, "…")
	gap := width - lipgloss.Width(crumbs) - projectWidth
	if gap < 1 {
		gap = 1
	}
	return crumbs + strings.Repeat(" ", gap) + project
}

// appBar is the single top line: brand and connection state.
func (m Model) appBar() string {
	width := m.contentWidth()
	status, statusStyle := "● API conectada", appBarOkStyle
	if m.APIErr != nil {
		status, statusStyle = "● API caída", appBarDangerStyle
	}
	right := statusStyle.Render(" " + status + " ")
	rw := lipgloss.Width(right)
	if width <= rw {
		return statusStyle.Render(ansi.Truncate(" "+status+" ", width, "…"))
	}
	brand := appBarBrandStyle.Render(" ◆ Software Metrics & Estimation ")
	if lipgloss.Width(brand)+rw+1 > width {
		brand = appBarBrandStyle.Render(" ◆ ")
	}
	gap := width - lipgloss.Width(brand) - rw
	if gap < 0 {
		gap = 0
	}
	return brand + appBarFillStyle.Render(strings.Repeat(" ", gap)) + right
}

// tabBar renders the workspace tabs on their own line for narrow terminals,
// dropping the digits only when they cannot fit.
func (m Model) tabBar() string {
	width := m.contentWidth()
	tabs := m.tabsView(true)
	if lipgloss.Width(tabs) > width {
		tabs = m.tabsView(false)
	}
	return ansi.Truncate(tabs, width, "")
}

// tabsView renders the tab chips. The active tab is a filled accent chip.
func (m Model) tabsView(numbered bool) string {
	current := m.currentTab()
	parts := make([]string, 0, len(workspaceTabs))
	for i, t := range workspaceTabs {
		label := t.title()
		if numbered {
			label = fmt.Sprintf("%d %s", i+1, label)
		}
		if t == current {
			parts = append(parts, tabActiveStyle.Render(" "+label+" "))
		} else {
			parts = append(parts, tabIdleStyle.Render(" "+label+" "))
		}
	}
	return strings.Join(parts, "")
}

// statusView is the live line above the footer: the last success notice, an
// API failure or a short summary of the current collection.
func (m Model) statusView() string {
	width := m.contentWidth()
	left := ""
	switch {
	case m.toast != "":
		left = appOK.Render("✓ " + m.toast)
	case m.APIErr != nil:
		left = appErr.Render("✗ API no disponible: " + m.APIErr.Error())
	case m.message != "":
		left = appErr.Render("✗ " + m.message)
	default:
		left = appHint.Render(m.contextSummary())
	}
	return ansi.Truncate(left, width, "…")
}

// contextSummary is the calm state of the status line: what the user is looking
// at and how much of it there is. It never replaces an inline error.
func (m Model) contextSummary() string {
	switch m.Screen {
	case "project-picker":
		if m.projectsLoading {
			return "Cargando proyectos…"
		}
		return countLabel(len(m.projects), "proyecto", "proyectos")
	case "backlog":
		stories := m.visibleStories()
		done := 0
		for _, story := range stories {
			if normalizeStoryStatus(story.Status) == "completada" {
				done++
			}
		}
		if m.backlogLoading {
			return "Cargando backlog…"
		}
		return fmt.Sprintf("%s · %s", countLabel(len(stories), "historia", "historias"), countLabel(done, "completada", "completadas"))
	case "story-detail":
		if story, ok := m.detailStory(); ok {
			return "Historia " + shortID(story.ID) + " · " + story.Title
		}
		return "Detalle de la historia"
	case "sprints":
		return countLabel(len(m.openSprints), "Sprint abierto", "Sprints abiertos")
	case "sprint-assigned-list":
		return fmt.Sprintf("%s en el Sprint", countLabel(len(m.sprintStories), "historia", "historias"))
	case "defects":
		open := 0
		for _, defect := range m.visibleDefects() {
			if defect.Status != "resuelto" {
				open++
			}
		}
		return fmt.Sprintf("%s · %d sin resolver", countLabel(len(m.visibleDefects()), "defecto", "defectos"), open)
	case "defect-detail":
		return "Defecto " + shortID(m.defectDetail.ID) + " · " + m.defectDetail.Status
	case "dashboard":
		if m.dashboard.sprintID != "" {
			return "Estado y métricas · Sprint " + m.dashboard.sprintLabel
		}
		return "Estado y métricas del proyecto"
	}
	parts := m.crumbParts()
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " › ")
}

// footerView is the shortcut line. It wraps whole key pairs instead of
// clipping them, and the shell reserves exactly the lines it needs.
func (m Model) footerView() string {
	lines := strings.Split(m.footer(), "\n")
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, m.contentWidth(), "…")
	}
	return strings.Join(lines, "\n")
}

// helpView is the full-screen reference for concepts and shortcuts. It is the
// only screen that replaces the workspace: it is a lookup, not a step in a
// flow. It windows the content to the body height so a short terminal scrolls
// instead of dropping the end of the list.
func (m Model) helpView() string {
	m.paneWidth, m.paneHeight = 0, 0
	lines := m.helpContentLines()
	height := m.bodyInnerHeight()
	offset := m.helpOffset
	if height > 0 && len(lines) > height {
		maxOffset := len(lines) - height
		if offset > maxOffset {
			offset = maxOffset
		}
		if offset < 0 {
			offset = 0
		}
		lines = lines[offset : offset+height]
	}
	body := m.cardAt(m.contentWidth(), height, strings.Join(lines, "\n"))
	return m.headerView() + "\n" + m.statusView() + "\n" + body + "\n" + m.footerView()
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
		if m.storyDetail.origin == storyDetailFromSprint {
			parts = append(parts, "Sprints", "Historias", "Detalle")
		} else {
			parts = append(parts, "Backlog", "Detalle")
		}
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
	case "defects", "defect-detail", "defect-create-form", "defect-sprint-select", "defect-create-confirm", "defect-status-select", "defect-resolution-select":
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
	if !m.inStoryFlow() {
		return nil
	}
	if m.storyDetail.origin == storyDetailFromSprint {
		return []string{"Sprints", "Historias", "Detalle"}
	}
	return []string{"Backlog", "Detalle"}
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

// countLabel renders a count with the right singular or plural noun.
func countLabel(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %s", n, plural)
}

func (m Model) contentWidth() int {
	if m.width > 0 {
		return m.width
	}
	return 80
}

// innerWidth is the usable width inside the active frame: the whole body in
// narrow layouts, or one master-detail pane when paneWidth is set.
func (m Model) innerWidth() int {
	base := m.contentWidth()
	if m.paneWidth > 0 {
		base = m.paneWidth
	}
	if w := base - 4; w > 10 {
		return w
	}
	return 10
}

// listAreaHeight is the number of lines a list may use. Zero means unbounded,
// which keeps tests and headless renders from clipping.
func (m Model) listAreaHeight() int {
	if m.paneHeight > 0 {
		return m.paneHeight
	}
	return m.bodyInnerHeight()
}

// card frames body in a rounded panel that fills the available body height.
// Plain text wraps to the inner width; styled lines are truncated as a safety
// net so a stray long row cannot widen the panel.
func (m Model) card(body string) string {
	return m.cardAt(m.contentWidth(), m.bodyInnerHeight(), body)
}

func (m Model) cardAt(width, innerHeight int, body string) string {
	inner := width - 4
	if inner < 10 {
		inner = 10
	}
	lines := m.renderLines(body, inner)
	if innerHeight > 0 {
		lines = clipPad(lines, innerHeight)
	}
	style := panelStyle.Width(width - 2)
	if innerHeight > 0 {
		style = style.Height(innerHeight)
	}
	return style.Render(strings.Join(lines, "\n"))
}

// renderLines wraps unstyled text and truncates styled lines to width, padding
// every line to an exact display width so panes and overlays align.
func (m Model) renderLines(body string, width int) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		if ansiPattern.MatchString(line) {
			out = append(out, padToWidth(ansi.Truncate(line, width, "…"), width))
			continue
		}
		for _, wrapped := range wrapPlain(line, width) {
			out = append(out, padToWidth(wrapped, width))
		}
	}
	return out
}

// shortName clips a name to width columns, adding an ellipsis.
func shortName(name string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(name) <= width {
		return name
	}
	return ansi.Truncate(name, width, "…")
}

// fitHeight clips or pads an already-styled view to exactly height lines.
func fitHeight(view string, height int) string {
	lines := strings.Split(view, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// clipPad trims or pads lines to exactly height.
func clipPad(lines []string, height int) []string {
	if height <= 0 {
		return lines
	}
	if len(lines) > height {
		return lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

// padToWidth right-pads s with spaces to width using its display width.
func padToWidth(s string, width int) string {
	if pad := width - lipgloss.Width(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}
