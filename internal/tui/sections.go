package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// sectionKind identifies one of the flows launched from the story detail that
// run in the section shell.
type sectionKind string

const (
	sectionPlanningPoker sectionKind = "planning-poker"
	sectionEffort        sectionKind = "effort"
)

// title returns the user-facing section name.
func (k sectionKind) title() string {
	switch k {
	case sectionPlanningPoker:
		return "Estimación"
	case sectionEffort:
		return "Esfuerzo"
	default:
		return string(k)
	}
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

// wholeStoryLabel is the synthetic task row that registers effort over the
// whole story; its empty ID means "no task".
const wholeStoryLabel = "Toda la historia (sin tarea)"

// sectionStep is the selector currently active inside a section.
type sectionStep int

const (
	// sectionStepNone is the zero value: no selector is active, so a section that
	// was never started cannot be mistaken for one at its first step.
	sectionStepNone sectionStep = iota
	sectionStepMembers
	sectionStepTasks
)

// sectionState is the navigation state of a flow launched from the story detail.
// The story comes already chosen (selected), so the flow starts at its first
// selector (see firstStep).
type sectionState struct {
	kind     sectionKind
	step     sectionStep
	selector selector
	tasks    selector // task list kept while Esfuerzo shows its members
	selected selectorItem
	detail   selectorItem
}

// SectionResultMsg is the asynchronous result of a section API read.
type SectionResultMsg struct {
	Kind     sectionKind
	Step     sectionStep
	StoryID  string
	Response APIResponse
	Err      error
}

// loadSectionStepCmd dispatches the read for the active step. All commands are
// GETs; the section never mutates business data.
func (m Model) loadSectionStepCmd() tea.Cmd {
	kind, step, projectID := m.section.kind, m.section.step, m.activeProjectID
	storyID := m.section.selected.ID

	switch step {
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
		return m, nil
	}
	items, ok := sectionItems(result)
	if !ok {
		m.section.selector.fail("No se pudo interpretar la respuesta de la API.")
		return m, nil
	}
	m.section.selector.setItems(items)
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
		if m.section.step == m.section.kind.firstStep() {
			return m.backToStoryDetail()
		}
		// The members of Esfuerzo are its only second selector: Esc restores the
		// task list as it was (rows, filter, cursor) without another request.
		m.section.step = sectionStepTasks
		m.section.selector = m.section.tasks
	case selectorRefresh:
		m.section.selector.startLoading()
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
	used := strings.Count(b.String(), "\n")
	available := m.listAreaHeight() - used - 1
	if m.listAreaHeight() > 0 && available < 1 {
		available = 1
	}
	start, end := windowBounds(s.cursor, len(visible), available)
	b.WriteString(sectionTitle("Opciones", s.cursor, len(visible), start, end))
	b.WriteString("\n")
	for i := start; i < end; i++ {
		item := visible[i]
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
