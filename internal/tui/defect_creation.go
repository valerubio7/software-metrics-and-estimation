package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var defectSeverities = []string{"alta", "media", "baja"}
var defectInitialStatuses = []string{"detectado", "en_correccion"}

type defectCreateForm struct {
	description   string
	severityIndex int
	statusIndex   int
	field         int
	storyID       string
	storyLabel    string
	sprintID      string
	sprintLabel   string
}

func (f defectCreateForm) severity() string { return defectSeverities[f.severityIndex] }
func (f defectCreateForm) status() string   { return defectInitialStatuses[f.statusIndex] }

type defectOptionsResult struct {
	step      string
	projectID string
	requestID uint64
	response  APIResponse
	err       error
}

// beginDefectCreationForStory opens the defect form with the story of the
// detail already chosen. The story is fixed for the whole flow, so the story
// selector is never shown. When the detail belongs to a Sprint, that Sprint is
// also preloaded as the detection Sprint.
func (m Model) beginDefectCreationForStory(story Story) (Model, tea.Cmd) {
	m.defectCreateForm = defectCreateForm{severityIndex: 0}
	m.defectSprintSelector = selector{}
	item := storySelectorItem(story)
	m.defectCreateForm.storyID, m.defectCreateForm.storyLabel = item.ID, item.Label
	if m.storyDetail.origin == storyDetailFromSprint {
		m.defectCreateForm.sprintID = m.selectedSprintID
		m.defectCreateForm.sprintLabel = m.sprintGoalByID(m.selectedSprintID)
	}
	m.Screen = "defect-create-form"
	return m, nil
}

// cancelDefectCreation abandons the draft and goes back to the story detail.
// It never issues a request; the next draft always starts empty.
func (m Model) cancelDefectCreation() (Model, tea.Cmd) {
	return m.backToStoryDetail()
}

func (m Model) updateDefectCreateForm(key tea.KeyMsg) (Model, tea.Cmd) {
	form := &m.defectCreateForm
	if key.Type == tea.KeyEsc {
		return m.cancelDefectCreation()
	}
	switch key.Type {
	case tea.KeyTab, tea.KeyDown:
		form.field = (form.field + 1) % 3
		return m, nil
	case tea.KeyShiftTab, tea.KeyUp:
		form.field = (form.field + 2) % 3
		return m, nil
	case tea.KeyEnter:
		if form.field < 2 {
			form.field++
			return m, nil
		}
		if form.sprintID == "" {
			return m.loadDefectOpenSprints()
		}
		return m.openDefectCreateConfirm()
	case tea.KeyLeft, tea.KeyRight:
		direction := 1
		if key.Type == tea.KeyLeft {
			direction = -1
		}
		if form.field == 1 {
			form.severityIndex = cycleOption(form.severityIndex, len(defectSeverities), direction)
		} else if form.field == 2 {
			form.statusIndex = cycleOption(form.statusIndex, len(defectInitialStatuses), direction)
		}
		return m, nil
	}
	if form.field == 0 {
		if value, ok := editText(form.description, key); ok {
			form.description = value
		}
	}
	return m, nil
}

func cycleOption(index, count, direction int) int {
	return (index + direction + count) % count
}

func (m Model) loadDefectOpenSprints() (Model, tea.Cmd) {
	m.Screen = "defect-sprint-select"
	m.defectSprintSelector = selector{}
	m.defectSprintSelector.startLoading()
	m.defectRequestSequence++
	requestID := m.defectRequestSequence
	m.defectSprintRequestID = requestID
	client, projectID := m.sprintClient, m.activeProjectID
	return m, func() tea.Msg {
		response, err := client.ListOpen(context.Background(), projectID)
		return defectOptionsResult{step: "sprints", projectID: projectID, requestID: requestID, response: response, err: err}
	}
}

func (m Model) updateDefectSprintSelect(key tea.KeyMsg) (Model, tea.Cmd) {
	switch m.defectSprintSelector.update(key) {
	case selectorBack:
		m.Screen = "defect-create-form"
	case selectorRefresh:
		return m.loadDefectOpenSprints()
	case selectorChosen:
		item, ok := m.defectSprintSelector.selected()
		if ok {
			m.defectCreateForm.sprintID, m.defectCreateForm.sprintLabel = item.ID, item.Label
			m.Screen = "defect-create-form"
		}
	}
	return m, nil
}

func (m Model) applyDefectOptionsResult(result defectOptionsResult) (Model, tea.Cmd) {
	if result.projectID != m.activeProjectID {
		return m, nil
	}
	switch result.step {
	case "sprints":
		if m.Screen != "defect-sprint-select" || !m.defectSprintSelector.loading || result.requestID != m.defectSprintRequestID {
			return m, nil
		}
		if message := readFailure(result.response, result.err); message != "" {
			m.defectSprintSelector.fail(message)
			return m, nil
		}
		var listing struct {
			Sprints []Sprint `json:"sprints"`
		}
		if err := json.Unmarshal(result.response.Body, &listing); err != nil {
			m.defectSprintSelector.fail(unreadableResponse(err))
			return m, nil
		}
		items := make([]selectorItem, 0, len(listing.Sprints))
		for _, sprint := range listing.Sprints {
			if sprint.IsClosed {
				continue
			}
			label := strings.TrimSpace(sprint.SprintGoal)
			if label == "" {
				label = "(sin objetivo)"
			}
			items = append(items, selectorItem{ID: sprint.ID, Label: label, Meta: shortID(sprint.ID) + " · abierto"})
		}
		m.defectSprintSelector.setItems(items)
	}
	return m, nil
}

func readFailure(response APIResponse, err error) string {
	if err != nil {
		return "Error de conexión: " + err.Error()
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Sprintf("Error HTTP %d: %s", response.StatusCode, string(response.Body))
	}
	return ""
}

func (m Model) viewDefectCreateForm() string {
	f := m.defectCreateForm
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Registrar defecto"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Proyecto: " + m.projectLabel()))
	b.WriteString("\n\n")
	b.WriteString(m.formRow(f.field == 0, "Descripción", f.description) + "\n")
	b.WriteString(m.formRow(f.field == 1, "Severidad", f.severity()) + "\n")
	b.WriteString(m.formRow(f.field == 2, "Estado inicial", f.status()) + "\n")
	if f.storyLabel != "" {
		b.WriteString("Historia relacionada: " + f.storyLabel + "\n")
	}
	if f.sprintLabel != "" {
		b.WriteString("Sprint de detección: " + f.sprintLabel + "\n")
	}
	b.WriteString("\n")
	if f.sprintID != "" {
		b.WriteString(hintStyle.Render("Borrador sin registrar."))
	} else {
		// The Sprint came from the Sprint detail or is chosen in the next step.
		b.WriteString(hintStyle.Render("Usá ←/→ para cambiar severidad o estado; Enter en el último campo elige el Sprint de detección."))
	}
	return b.String()
}
