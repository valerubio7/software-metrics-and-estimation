package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// sectionKind identifies one of the Sprint 3 navigation sections.
type sectionKind string

const (
	sectionPlanningPoker sectionKind = "planning-poker"
	sectionEffort        sectionKind = "effort"
	sectionDefects       sectionKind = "defects"
	sectionMetrics       sectionKind = "metrics"
	sectionDashboard     sectionKind = "dashboard"
)

// title returns the user-facing section name.
func (k sectionKind) title() string {
	switch k {
	case sectionPlanningPoker:
		return "Planning Poker"
	case sectionEffort:
		return "Esfuerzo"
	case sectionDefects:
		return "Defectos"
	case sectionMetrics:
		return "Métricas"
	case sectionDashboard:
		return "Dashboard"
	default:
		return string(k)
	}
}

// primaryUsesStories reports whether the section starts from the backlog.
func (k sectionKind) primaryUsesStories() bool {
	switch k {
	case sectionPlanningPoker, sectionEffort, sectionDefects:
		return true
	}
	return false
}

func (k sectionKind) primaryHint() string {
	switch k {
	case sectionPlanningPoker:
		return "Elegí una historia del backlog para estimarla."
	case sectionEffort:
		return "Elegí una historia para registrar su esfuerzo."
	case sectionDefects:
		return "Elegí la historia relacionada con el defecto."
	default:
		return "Elegí un Sprint abierto para continuar."
	}
}

func (k sectionKind) primaryEmptyTitle() string {
	if k.primaryUsesStories() {
		return "No hay historias en el backlog."
	}
	return "No hay Sprints abiertos."
}

func (k sectionKind) primaryEmptyHint() string {
	if k.primaryUsesStories() {
		return "Creá una historia desde Backlog y volvé a intentar."
	}
	return "Creá un Sprint desde Sprints y volvé a intentar."
}

// wholeStoryLabel is the synthetic task row that registers effort over the
// whole story; its empty ID means "no task".
const wholeStoryLabel = "Toda la historia (sin tarea)"

// sectionStep is the selector currently active inside a section.
type sectionStep int

const (
	sectionStepPrimary sectionStep = iota
	sectionStepMembers
	sectionStepTasks
	sectionStepSprints
	sectionStepScope
	sectionStepMetrics
	sectionStepNotice
)

type metricsScope string

const (
	metricsScopeProject metricsScope = "project"
	metricsScopeSprint  metricsScope = "sprint"
)

// sectionState is the navigation state of a Sprint 3 section. The primary
// selector is preserved while a secondary selector is displayed, so Esc can
// restore the previous list without another request.
type sectionState struct {
	kind         sectionKind
	step         sectionStep
	metricsScope metricsScope
	metrics      metricsPayload
	selector     selector
	primary      selector
	tasks        selector // task list kept while Esfuerzo shows its members
	selected     selectorItem
	detail       selectorItem
	notice       string
	noticeReturn sectionStep
}

// SectionResultMsg is the asynchronous result of a section API read.
type SectionResultMsg struct {
	Kind     sectionKind
	Step     sectionStep
	Response APIResponse
	Err      error
}

// openSection enters a section directly, or asks for a project first when the
// session has none. The section kind is retained across the picker so the
// breadcrumb and the resume path can use it.
func (m Model) openSection(kind sectionKind) (Model, tea.Cmd) {
	m.section.kind = kind
	if m.activeProjectID == "" {
		return m.openProjectPicker("section:" + string(kind))
	}
	m.Screen = "section"
	m.section.step = sectionStepPrimary
	m.section.metricsScope = ""
	m.section.metrics = metricsPayload{}
	m.section.selected = selectorItem{}
	m.section.detail = selectorItem{}
	m.section.notice = ""
	m.section.selector = selector{}
	m.section.tasks = selector{}
	if kind == sectionMetrics {
		m.section.step = sectionStepScope
		m.section.selector.setItems(sectionMetricsScopeItems())
		return m, nil
	}
	m.section.selector.startLoading()
	m.section.primary = m.section.selector
	return m, m.loadSectionStepCmd()
}

// loadSectionStepCmd dispatches the read for the active step. All commands are
// GETs; the section never mutates business data.
func (m Model) loadSectionStepCmd() tea.Cmd {
	kind, step, projectID := m.section.kind, m.section.step, m.activeProjectID
	storyID := m.section.selected.ID
	sprintID := m.section.selected.ID
	scope := m.section.metricsScope

	switch step {
	case sectionStepPrimary:
		if kind.primaryUsesStories() {
			client := m.backlogClient
			return func() tea.Msg {
				response, err := client.List(context.Background(), projectID)
				return SectionResultMsg{Kind: kind, Step: step, Response: response, Err: err}
			}
		}
		client := m.sprintClient
		return func() tea.Msg {
			response, err := client.ListOpen(context.Background(), projectID)
			return SectionResultMsg{Kind: kind, Step: step, Response: response, Err: err}
		}
	case sectionStepMembers:
		client := m.memberClient
		return func() tea.Msg {
			response, err := client.List(context.Background(), projectID)
			return SectionResultMsg{Kind: kind, Step: step, Response: response, Err: err}
		}
	case sectionStepTasks:
		client := m.taskClient
		return func() tea.Msg {
			response, err := client.ListByStory(context.Background(), projectID, storyID)
			return SectionResultMsg{Kind: kind, Step: step, Response: response, Err: err}
		}
	case sectionStepSprints:
		client := m.sprintClient
		return func() tea.Msg {
			response, err := client.ListOpen(context.Background(), projectID)
			return SectionResultMsg{Kind: kind, Step: step, Response: response, Err: err}
		}
	case sectionStepMetrics:
		client := m.metricsClient
		return func() tea.Msg {
			if scope == metricsScopeProject {
				response, err := client.Project(context.Background(), projectID)
				return SectionResultMsg{Kind: kind, Step: step, Response: response, Err: err}
			}
			response, err := client.Sprint(context.Background(), projectID, sprintID)
			return SectionResultMsg{Kind: kind, Step: step, Response: response, Err: err}
		}
	}
	return nil
}

// applySectionResult stores a response only when it still belongs to the active
// section, step and in-flight request, discarding late responses after a
// navigation change.
func (m Model) applySectionResult(result SectionResultMsg) (Model, tea.Cmd) {
	if m.Screen != "section" || m.section.kind != result.Kind ||
		m.section.step != result.Step || !m.section.selector.loading {
		return m, nil
	}

	if message := sectionFailure(result.Err, result.Response); message != "" {
		m.section.selector.fail(message)
		if result.Step == sectionStepPrimary {
			m.section.primary = m.section.selector
		}
		return m, nil
	}
	if result.Step == sectionStepMetrics {
		metrics, ok := parseMetricsPayload(result.Response.Body)
		if !ok {
			m.section.selector.fail("No se pudo interpretar la respuesta de la API.")
			return m, nil
		}
		m.section.metrics = metrics
		m.section.selector.loading = false
		m.section.selector.err = ""
		return m, nil
	}

	items, ok := sectionItems(result)
	if !ok {
		m.section.selector.fail("No se pudo interpretar la respuesta de la API.")
		if result.Step == sectionStepPrimary {
			m.section.primary = m.section.selector
		}
		return m, nil
	}
	m.section.selector.setItems(items)
	if result.Step == sectionStepPrimary {
		m.section.primary = m.section.selector
	}
	return m, nil
}

// updateSection drives the shared selector and the step transitions.
func (m Model) updateSection(key tea.KeyMsg) (Model, tea.Cmd) {
	if m.section.step == sectionStepNotice {
		if key.Type == tea.KeyEsc || key.Type == tea.KeyEnter {
			// The selector still holds the list that produced the notice, so
			// returning restores the previous step without another request.
			m.section.step = m.section.noticeReturn
			m.section.notice = ""
		}
		return m, nil
	}
	if m.section.step == sectionStepMetrics {
		if key.Type == tea.KeyEsc {
			if m.section.metricsScope == metricsScopeSprint {
				m.section.step = sectionStepPrimary
				m.section.selector = m.section.primary
			} else {
				m.section.step = sectionStepScope
				m.section.selector = selector{}
				m.section.selector.setItems(sectionMetricsScopeItems())
			}
			return m, nil
		}
		if key.Type == tea.KeyRunes && string(key.Runes) == "r" {
			m.section.selector.startLoading()
			return m, m.loadSectionStepCmd()
		}
		return m, nil
	}

	if m.section.kind == sectionPlanningPoker && m.section.step == sectionStepMembers && key.Type == tea.KeySpace {
		item, ok := m.section.selector.selected()
		if ok {
			if m.pokerVotes == nil {
				m.pokerVotes = make(map[string]string)
			}
			if m.pokerVotes[item.ID] == "selected" {
				delete(m.pokerVotes, item.ID)
			} else {
				m.pokerVotes[item.ID] = "selected"
			}
		}
		return m, nil
	}
	action := m.section.selector.update(key)
	switch action {
	case selectorBack:
		if m.section.step == sectionStepScope {
			m.Screen, m.Cursor = "menu", 0
			return m, nil
		}
		if m.section.step == sectionStepPrimary {
			if m.section.kind == sectionMetrics && m.section.metricsScope == metricsScopeSprint {
				m.section.step = sectionStepScope
				m.section.selector = selector{}
				m.section.selector.setItems(sectionMetricsScopeItems())
				return m, nil
			}
			m.Screen, m.Cursor = "menu", 0
			return m, nil
		}
		if m.section.kind == sectionEffort && m.section.step == sectionStepMembers {
			m.section.step = sectionStepTasks
			m.section.selector = m.section.tasks
			return m, nil
		}
		m.section.step = sectionStepPrimary
		m.section.selector = m.section.primary
	case selectorRefresh:
		if m.section.step == sectionStepScope {
			return m, nil
		}
		m.section.selector.startLoading()
		if m.section.step == sectionStepPrimary {
			m.section.primary = m.section.selector
		}
		return m, m.loadSectionStepCmd()
	case selectorChosen:
		return m.chooseSectionItem()
	}
	return m, nil
}

// chooseSectionItem advances to the secondary selector or finishes the
// navigation shell with the future-issue notice.
func (m Model) chooseSectionItem() (Model, tea.Cmd) {
	item, ok := m.section.selector.selected()
	if !ok {
		return m, nil
	}
	if m.section.step == sectionStepScope && m.section.kind == sectionMetrics {
		if item.ID == string(metricsScopeProject) {
			m.section.metricsScope = metricsScopeProject
			m.section.selected = selectorItem{}
			return m.beginSectionStep(sectionStepMetrics)
		}
		m.section.metricsScope = metricsScopeSprint
		m.section.selected = selectorItem{}
		return m.beginSectionStep(sectionStepPrimary)
	}

	if m.section.step == sectionStepPrimary {
		m.section.selected = item
		m.section.primary = m.section.selector
		switch m.section.kind {
		case sectionPlanningPoker:
			return m.beginSectionStep(sectionStepMembers)
		case sectionEffort:
			return m.beginSectionStep(sectionStepTasks)
		case sectionDefects:
			return m.beginSectionStep(sectionStepSprints)
		case sectionMetrics:
			m.section.metricsScope = metricsScopeSprint
			return m.beginSectionStep(sectionStepMetrics)
		default:
			m.section.noticeReturn = sectionStepPrimary
			m.section.step = sectionStepNotice
			m.section.notice = sectionNotice(m.section.kind, item, selectorItem{})
			return m, nil
		}
	}

	if m.section.kind == sectionEffort && m.section.step == sectionStepTasks {
		// Keep the task list as it is (rows, filter, cursor) so Esc from the
		// members can restore it without another request.
		m.section.detail = item
		m.section.tasks = m.section.selector
		return m.beginSectionStep(sectionStepMembers)
	}

	if m.section.kind == sectionPlanningPoker && m.section.step == sectionStepMembers {
		m.pokerParticipants = []selectorItem{item}
		for _, member := range m.section.selector.items {
			if m.pokerVotes[member.ID] == "selected" && member.ID != item.ID {
				m.pokerParticipants = append(m.pokerParticipants, member)
			}
		}
		m.pokerVotes = map[string]string{}
		m.pokerPhase, m.pokerCursor, m.Screen = "vote", 0, "planning-poker"
		return m, nil
	}
	m.section.detail = item
	m.section.noticeReturn = m.section.step
	m.section.step = sectionStepNotice
	m.section.notice = sectionNotice(m.section.kind, m.section.selected, item)
	return m, nil
}

func (m Model) beginSectionStep(step sectionStep) (Model, tea.Cmd) {
	m.section.step = step
	m.section.selector = selector{}
	if step == sectionStepScope {
		m.section.selector.setItems(sectionMetricsScopeItems())
		return m, nil
	}
	m.section.selector.startLoading()
	return m, m.loadSectionStepCmd()
}

// sectionNotice explains the limit of the navigation shell and names the
// future issue that will implement the business flow.
func sectionNotice(kind sectionKind, primary, detail selectorItem) string {
	switch kind {
	case sectionPlanningPoker:
		return fmt.Sprintf("Historia: %s\nParticipante: %s\n\nLa votación de Planning Poker estará disponible en TUI-07.\nNo se guardó ningún dato.", primary.Label, detail.Label)
	case sectionEffort:
		return fmt.Sprintf("Historia: %s\nTarea: %s\n\nEl registro de esfuerzo estará disponible en TUI-08.\nNo se guardó ningún dato.", primary.Label, detail.Label)
	case sectionDefects:
		return fmt.Sprintf("Historia: %s\nSprint de detección: %s\n\nEl alta de defectos estará disponible en TUI-09.\nNo se guardó ningún dato.", primary.Label, detail.Label)
	case sectionMetrics:
		return fmt.Sprintf("Sprint: %s\n\nLas métricas estarán disponibles en TUI-10.\nNo se guardó ningún dato.", primary.Label)
	case sectionDashboard:
		return fmt.Sprintf("Sprint: %s\n\nEl Dashboard estará disponible en TUI-11.\nNo se guardó ningún dato.", primary.Label)
	default:
		return "El flujo de negocio todavía no está disponible.\nNo se guardó ningún dato."
	}
}

func sectionMetricsScopeItems() []selectorItem {
	return []selectorItem{
		{ID: string(metricsScopeProject), Label: "Proyecto", Meta: "Métricas consolidadas del proyecto activo"},
		{ID: string(metricsScopeSprint), Label: "Sprint", Meta: "Métricas del Sprint abierto seleccionado"},
	}
}

func (m Model) viewSection() string {
	switch m.section.step {
	case sectionStepScope:
		return m.viewSelector(&m.section.selector, m.section.kind.title()+" · Ámbito",
			"Elegí si querés ver métricas del proyecto o de un Sprint.",
			"No hay ámbitos disponibles.", "")
	case sectionStepPrimary:
		title := m.section.kind.title()
		hint := m.section.kind.primaryHint()
		emptyTitle, emptyHint := m.section.kind.primaryEmptyTitle(), m.section.kind.primaryEmptyHint()
		if m.section.kind == sectionMetrics {
			title += " · Sprint"
			hint = "Elegí un Sprint abierto para consultar métricas."
			emptyTitle = "No hay Sprints abiertos."
			emptyHint = "Creá un Sprint desde Sprints y volvé a intentar."
		}
		return m.viewSelector(&m.section.selector, title, hint, emptyTitle, emptyHint)
	case sectionStepMembers:
		if m.section.kind == sectionPlanningPoker {
			return m.viewPokerParticipants()
		}
		return m.viewSelector(&m.section.selector, m.section.kind.title()+" · Integrante",
			"Integrante que realizó el trabajo.", "El proyecto no tiene integrantes registrados.",
			"Los integrantes se dan de alta mediante la API; r vuelve a consultar.")
	case sectionStepTasks:
		hint := "Elegí una tarea de " + m.section.selected.Label + "."
		if m.section.kind == sectionEffort {
			hint = "Tarea de " + m.section.selected.Label + " o la historia completa."
		}
		view := m.viewSelector(&m.section.selector, m.section.kind.title()+" · Tareas",
			hint, "La historia no tiene tareas.", "Creá tareas desde Sprints.")
		if items := m.section.selector.items; m.section.kind == sectionEffort &&
			!m.section.selector.loading && m.section.selector.err == "" &&
			len(items) == 1 && items[0].ID == "" {
			view += "\n\n" + hintStyle.Render("La historia no tiene tareas.")
		}
		return view
	case sectionStepSprints:
		return m.viewSelector(&m.section.selector, m.section.kind.title()+" · Sprint",
			"Elegí el Sprint relacionado.", "No hay Sprints abiertos.",
			"Creá un Sprint desde Sprints.")
	case sectionStepMetrics:
		return m.viewMetrics()
	case sectionStepNotice:
		return m.viewSectionNotice()
	default:
		return ""
	}
}

func (m Model) viewMetrics() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Métricas"))
	b.WriteString("\n")
	if m.section.metricsScope == metricsScopeSprint {
		b.WriteString(hintStyle.Render("Ámbito: Sprint · " + m.section.selected.Label))
	} else {
		b.WriteString(hintStyle.Render("Ámbito: Proyecto"))
	}
	b.WriteString("\n\n")

	if m.section.selector.err != "" {
		b.WriteString(m.errorBlock(m.section.selector.err))
		b.WriteString("\n\n")
		b.WriteString(hintStyle.Render("r para reintentar · Esc para volver."))
		return strings.TrimRight(b.String(), "\n")
	}
	if m.section.selector.loading {
		b.WriteString(statusMutedStyle.Render("Cargando…"))
		return strings.TrimRight(b.String(), "\n")
	}

	rows := [][2]string{
		{"Story Points planificados", formatMetric(m.section.metrics.StoryPointsPlanned, "")},
		{"Story Points completados", formatMetric(m.section.metrics.StoryPointsCompleted, "")},
		{"Velocidad", formatMetric(m.section.metrics.Velocity, "")},
		{"Horas estimadas", formatMetric(m.section.metrics.EstimatedHours, " h")},
		{"Horas reales", formatMetric(m.section.metrics.ActualHours, " h")},
		{"Desviación", formatMetric(m.section.metrics.Deviation, "")},
		{"Historias completadas", formatMetric(m.section.metrics.CompletedStoriesPercentage, "%")},
		{"Defectos detectados", formatMetric(m.section.metrics.DefectsDetected, "")},
		{"Defectos resueltos", formatMetric(m.section.metrics.DefectsResolved, "")},
	}
	width := m.innerWidth()
	for _, row := range rows {
		line := row[0] + ": " + row[1]
		for _, wrapped := range wrapPlain(line, width) {
			b.WriteString(wrapped)
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// viewSelector renders the shared navigable list with search, loading, error
// and empty states.
func (m Model) viewSelector(s *selector, title, hint, emptyTitle, emptyHint string) string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render(hint))
	b.WriteString("\n\n")

	if s.filtering || s.filter != "" {
		b.WriteString(searchLine(s.filter, s.filtering))
		b.WriteString("\n\n")
	}

	if s.err != "" {
		b.WriteString(m.errorBlock(s.err))
		b.WriteString("\n\n")
		b.WriteString(hintStyle.Render("r para reintentar · Esc para volver."))
		return strings.TrimRight(b.String(), "\n")
	}
	if s.loading {
		b.WriteString(statusMutedStyle.Render("Cargando…"))
		return strings.TrimRight(b.String(), "\n")
	}
	if len(s.items) == 0 {
		b.WriteString(emptyState("◇", emptyTitle, emptyHint))
		return strings.TrimRight(b.String(), "\n")
	}

	visible := s.visible()
	if len(visible) == 0 {
		b.WriteString(emptyState("⌕", "Sin coincidencias.", "Probá con otro texto o borrá el filtro con Esc."))
		return strings.TrimRight(b.String(), "\n")
	}
	for i, item := range visible {
		text := item.Label
		if item.Meta != "" {
			text += "  ·  " + item.Meta
		}
		b.WriteString(m.row(i == s.cursor, text))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m Model) viewSectionNotice() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render(m.section.kind.title() + " · Próximamente"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Este flujo todavía no está implementado; la navegación ya está disponible."))
	b.WriteString("\n\n")
	b.WriteString(m.section.notice)
	return strings.TrimRight(b.String(), "\n")
}

// sectionItems projects each response to the shared selector shape.
func sectionItems(result SectionResultMsg) ([]selectorItem, bool) {
	switch result.Step {
	case sectionStepPrimary:
		if result.Kind.primaryUsesStories() {
			var listing struct {
				Stories []Story `json:"stories"`
			}
			if err := json.Unmarshal(result.Response.Body, &listing); err != nil {
				return nil, false
			}
			items := make([]selectorItem, 0, len(listing.Stories))
			for _, story := range listing.Stories {
				items = append(items, storySelectorItem(story))
			}
			return items, true
		}
		return sprintSelectorItems(result.Response.Body)
	case sectionStepMembers:
		var listing struct {
			Members []Member `json:"members"`
		}
		if err := json.Unmarshal(result.Response.Body, &listing); err != nil {
			return nil, false
		}
		items := make([]selectorItem, 0, len(listing.Members))
		for _, member := range listing.Members {
			meta := "sin email"
			if member.Email != nil && *member.Email != "" {
				meta = *member.Email
			}
			items = append(items, selectorItem{ID: member.ID, Label: member.FullName, Meta: meta})
		}
		return items, true
	case sectionStepTasks:
		var listing struct {
			Tasks []Task `json:"tasks"`
		}
		if err := json.Unmarshal(result.Response.Body, &listing); err != nil {
			return nil, false
		}
		items := make([]selectorItem, 0, len(listing.Tasks)+1)
		if result.Kind == sectionEffort {
			// Effort can be registered over the whole story, so the choice is
			// always available, even when the story has no tasks.
			items = append(items, selectorItem{Label: wholeStoryLabel, Meta: "esfuerzo sobre la historia"})
		}
		for _, task := range listing.Tasks {
			meta := "sin estimar"
			if task.EstimatedHours != nil {
				meta = fmt.Sprintf("%g h", *task.EstimatedHours)
			}
			items = append(items, selectorItem{ID: task.ID, Label: task.Title, Meta: meta})
		}
		return items, true
	case sectionStepSprints:
		return sprintSelectorItems(result.Response.Body)
	}
	return nil, false
}

func storySelectorItem(story Story) selectorItem {
	meta := make([]string, 0, 3)
	if value := strings.TrimSpace(story.Priority); value != "" {
		meta = append(meta, value)
	}
	if story.StoryPoints != nil {
		meta = append(meta, fmt.Sprintf("%d pts", *story.StoryPoints))
	}
	if value := strings.TrimSpace(story.Status); value != "" {
		meta = append(meta, value)
	}
	return selectorItem{ID: story.ID, Label: story.Title, Meta: strings.Join(meta, " · ")}
}

func sprintSelectorItems(body []byte) ([]selectorItem, bool) {
	var listing struct {
		Sprints []Sprint `json:"sprints"`
	}
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, false
	}
	items := make([]selectorItem, 0, len(listing.Sprints))
	for _, sprint := range listing.Sprints {
		label := strings.TrimSpace(sprint.SprintGoal)
		if label == "" {
			label = "(sin objetivo)"
		}
		state := "abierto"
		if sprint.IsClosed {
			state = "cerrado"
		}
		items = append(items, selectorItem{ID: sprint.ID, Label: label, Meta: shortID(sprint.ID) + " · " + state})
	}
	return items, true
}

func sectionFailure(err error, response APIResponse) string {
	if err != nil {
		return "Error de conexión: " + err.Error()
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return describeBacklogFailure(response)
	}
	return ""
}

func (m Model) sectionFooter() string {
	if m.section.step == sectionStepNotice {
		return m.keyHints("Enter/Esc", "volver", "?", "ayuda", "q", "salir")
	}
	if m.section.step == sectionStepMetrics {
		return m.keyHints("r", "refrescar", "p", "proyecto", "Esc", "volver", "?", "ayuda", "q", "salir")
	}
	if m.section.selector.filtering {
		return m.keyHints("escribí", "para filtrar", "↑/↓", "mover", "Enter", "dejar de filtrar", "Esc", "limpiar")
	}

	action := "elegir"
	switch m.section.step {
	case sectionStepScope:
		action = "elegir ámbito"
	case sectionStepPrimary:
		if m.section.kind.primaryUsesStories() {
			action = "elegir historia"
		} else {
			action = "elegir Sprint"
		}
	case sectionStepMembers:
		action = "elegir integrante"
	case sectionStepTasks:
		action = "elegir tarea"
	case sectionStepSprints:
		action = "elegir Sprint"
	}
	if m.section.kind == sectionPlanningPoker && m.section.step == sectionStepMembers {
		return m.keyHints("↑/↓", "mover", "Space", "seleccionar", "Enter", "confirmar participantes", "/", "buscar", "Esc", "volver", "?", "ayuda", "q", "salir")
	}
	return m.keyHints("↑/↓", "mover", "Enter", action, "/", "buscar", "r", "refrescar",
		"p", "proyecto", "Esc", "volver", "?", "ayuda", "q", "salir")
}

func (m Model) sectionHelpRows() [][2]string {
	if m.section.step == sectionStepNotice {
		return [][2]string{
			{"Enter / Esc", "Volver a la lista de la sección"},
			{"?", "Mostrar u ocultar esta ayuda"},
			{"q", "Salir (no dentro de un filtro)"},
			{"Ctrl+C", "Salir desde cualquier pantalla"},
		}
	}
	if m.section.step == sectionStepMetrics {
		return [][2]string{
			{"r", "Refrescar las métricas del ámbito activo"},
			{"p", "Cambiar el proyecto activo"},
			{"Esc", "Volver al selector anterior"},
			{"?", "Mostrar u ocultar esta ayuda"},
			{"q", "Salir (no dentro de un filtro)"},
			{"Ctrl+C", "Salir desde cualquier pantalla"},
		}
	}

	action := "Elegir / confirmar la fila resaltada"
	switch m.section.step {
	case sectionStepScope:
		action = "Elegir el ámbito de métricas"
	case sectionStepPrimary:
		if m.section.kind.primaryUsesStories() {
			action = "Elegir la historia resaltada"
		} else {
			action = "Elegir el Sprint resaltado"
		}
	case sectionStepMembers:
		action = "Elegir el integrante resaltado"
	case sectionStepTasks:
		action = "Elegir la tarea resaltada"
	case sectionStepSprints:
		action = "Elegir el Sprint resaltado"
	}
	return [][2]string{
		{"↑ / ↓", "Mover la selección"},
		{"Enter", action},
		{"/", "Buscar o filtrar en la lista"},
		{"r", "Refrescar el listado actual"},
		{"p", "Cambiar el proyecto activo"},
		{"Esc", "Limpiar el filtro y volver un nivel"},
		{"?", "Mostrar u ocultar esta ayuda"},
		{"q", "Salir (no dentro de un filtro)"},
		{"Ctrl+C", "Salir desde cualquier pantalla"},
	}
}
