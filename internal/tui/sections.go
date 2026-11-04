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
)

// title returns the user-facing section name.
func (k sectionKind) title() string {
	switch k {
	case sectionPlanningPoker:
		return "Estimación"
	case sectionEffort:
		return "Esfuerzo"
	case sectionDefects:
		return "Defectos"
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

// firstStep is the selector a flow starts at when it is launched from the story
// detail, where the story is already chosen: the participants for Planning
// Poker and the tasks for Esfuerzo.
func (k sectionKind) firstStep() sectionStep {
	if k == sectionEffort {
		return sectionStepTasks
	}
	return sectionStepMembers
}

// storyNextStep is the selector that follows choosing a story; an unknown
// kind keeps the primary step, so choosing a story is a no-op.
func (k sectionKind) storyNextStep() sectionStep {
	switch k {
	case sectionPlanningPoker:
		return sectionStepMembers
	case sectionEffort:
		return sectionStepTasks
	case sectionDefects:
		return sectionStepSprints
	}
	return sectionStepPrimary
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
)

// sectionState is the navigation state of a Sprint 3 section. The primary
// selector is preserved while a secondary selector is displayed, so Esc can
// restore the previous list without another request.
type sectionState struct {
	kind     sectionKind
	step     sectionStep
	selector selector
	primary  selector
	tasks    selector // task list kept while Esfuerzo shows its members
	selected selectorItem
	detail   selectorItem
	// preselected marks a section opened from the backlog with P/E: the story
	// came chosen, so Esc on its first selector returns to the backlog.
	preselected bool
}

// SectionResultMsg is the asynchronous result of a section API read.
type SectionResultMsg struct {
	Kind     sectionKind
	Step     sectionStep
	StoryID  string
	Response APIResponse
	Err      error
}

// openSection enters a section directly, or asks for a project first when the
// session has none. The section kind is retained across the picker so the
// breadcrumb and the resume path can use it.
func (m Model) openSection(kind sectionKind) (Model, tea.Cmd) {
	m.section.kind = kind
	m.returnTo = returnDefault
	if m.activeProjectID == "" {
		return m.leaveWorkspace()
	}
	m.Screen = "section"
	m.section.step = sectionStepPrimary
	m.section.selected = selectorItem{}
	m.section.detail = selectorItem{}
	m.section.preselected = false
	m.section.selector = selector{}
	m.section.tasks = selector{}
	m.effort = effortState{seq: m.effort.seq}
	m.section.selector.startLoading()
	m.section.primary = m.section.selector
	return m, m.loadSectionStepCmd()
}

// loadSectionStepCmd dispatches the read for the active step. All commands are
// GETs; the section never mutates business data.
func (m Model) loadSectionStepCmd() tea.Cmd {
	kind, step, projectID := m.section.kind, m.section.step, m.activeProjectID
	storyID := m.section.selected.ID

	switch step {
	case sectionStepPrimary:
		if kind.primaryUsesStories() {
			client := m.backlogClient
			return func() tea.Msg {
				response, err := client.List(context.Background(), projectID)
				return SectionResultMsg{Kind: kind, Step: step, StoryID: storyID, Response: response, Err: err}
			}
		}
		client := m.sprintClient
		return func() tea.Msg {
			response, err := client.ListOpen(context.Background(), projectID)
			return SectionResultMsg{Kind: kind, Step: step, StoryID: storyID, Response: response, Err: err}
		}
	case sectionStepMembers:
		client := m.memberClient
		return func() tea.Msg {
			response, err := client.List(context.Background(), projectID)
			return SectionResultMsg{Kind: kind, Step: step, StoryID: storyID, Response: response, Err: err}
		}
	case sectionStepTasks:
		client := m.taskClient
		return func() tea.Msg {
			response, err := client.ListByStory(context.Background(), projectID, storyID)
			return SectionResultMsg{Kind: kind, Step: step, StoryID: storyID, Response: response, Err: err}
		}
	case sectionStepSprints:
		client := m.sprintClient
		return func() tea.Msg {
			response, err := client.ListOpen(context.Background(), projectID)
			return SectionResultMsg{Kind: kind, Step: step, StoryID: storyID, Response: response, Err: err}
		}
	}
	return nil
}

// applySectionResult stores a response only when it still belongs to the active
// section, step, story and in-flight request, discarding late responses after a
// navigation change. The story matters because a flow launched from the detail
// of another story reuses the same section, step and loading state.
func (m Model) applySectionResult(result SectionResultMsg) (Model, tea.Cmd) {
	if m.Screen != "section" || m.section.kind != result.Kind ||
		m.section.step != result.Step || m.section.selected.ID != result.StoryID ||
		!m.section.selector.loading {
		return m, nil
	}

	if message := sectionFailure(result.Err, result.Response); message != "" {
		m.section.selector.fail(message)
		if result.Step == sectionStepPrimary {
			m.section.primary = m.section.selector
		}
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
		if m.section.step == sectionStepPrimary {
			return m.leaveWorkspace()
		}
		if m.returnTo == returnStoryDetail && m.section.step == m.section.kind.firstStep() {
			return m.backToStoryDetail()
		}
		if m.section.preselected && m.section.step == m.section.kind.storyNextStep() {
			return m.openBacklog()
		}
		if m.section.kind == sectionEffort && m.section.step == sectionStepMembers {
			m.section.step = sectionStepTasks
			m.section.selector = m.section.tasks
			return m, nil
		}
		m.section.step = sectionStepPrimary
		m.section.selector = m.section.primary
	case selectorRefresh:
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

// chooseSectionItem advances to the next selector or opens the section flow;
// a step without a next one keeps the current list.
func (m Model) chooseSectionItem() (Model, tea.Cmd) {
	item, ok := m.section.selector.selected()
	if !ok {
		return m, nil
	}
	if m.section.step == sectionStepPrimary {
		m.section.selected = item
		m.section.primary = m.section.selector
		if next := m.section.kind.storyNextStep(); next != sectionStepPrimary {
			return m.beginSectionStep(next)
		}
		return m, nil
	}

	if m.section.kind == sectionEffort && m.section.step == sectionStepTasks {
		// Keep the task list as it is (rows, filter, cursor) so Esc from the
		// members can restore it without another request.
		m.section.detail = item
		m.section.tasks = m.section.selector
		return m.beginSectionStep(sectionStepMembers)
	}

	if m.section.kind == sectionEffort && m.section.step == sectionStepMembers {
		return m.openEffortForm(item), nil
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
	return m, nil
}

func (m Model) beginSectionStep(step sectionStep) (Model, tea.Cmd) {
	m.section.step = step
	m.section.selector = selector{}
	m.section.selector.startLoading()
	return m, m.loadSectionStepCmd()
}

func (m Model) viewSection() string {
	switch m.section.step {
	case sectionStepPrimary:
		title := m.section.kind.title()
		hint := m.section.kind.primaryHint()
		emptyTitle, emptyHint := m.section.kind.primaryEmptyTitle(), m.section.kind.primaryEmptyHint()
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
	default:
		return ""
	}
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
