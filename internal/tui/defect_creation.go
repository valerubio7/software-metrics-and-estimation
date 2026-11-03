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

func (m Model) beginDefectCreation() (Model, tea.Cmd) {
	m.defectCreateForm = defectCreateForm{severityIndex: 0}
	m.defectStorySelector, m.defectSprintSelector = selector{}, selector{}
	m.Screen = "defect-create-form"
	return m, nil
}

func (m Model) updateDefectCreateForm(key tea.KeyMsg) (Model, tea.Cmd) {
	form := &m.defectCreateForm
	if key.Type == tea.KeyEsc {
		m.Screen = "defects"
		return m, nil
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
		if form.storyID == "" {
			return m.loadDefectStories()
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

func (m Model) loadDefectStories() (Model, tea.Cmd) {
	m.Screen = "defect-story-select"
	m.defectStorySelector = selector{}
	m.defectStorySelector.startLoading()
	m.defectRequestSequence++
	requestID := m.defectRequestSequence
	m.defectStoryRequestID = requestID
	client, projectID := m.backlogClient, m.activeProjectID
	return m, func() tea.Msg {
		response, err := client.List(context.Background(), projectID)
		return defectOptionsResult{step: "stories", projectID: projectID, requestID: requestID, response: response, err: err}
	}
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

func (m Model) updateDefectStorySelect(key tea.KeyMsg) (Model, tea.Cmd) {
	switch m.defectStorySelector.update(key) {
	case selectorBack:
		m.Screen = "defect-create-form"
	case selectorRefresh:
		return m.loadDefectStories()
	case selectorChosen:
		item, ok := m.defectStorySelector.selected()
		if ok {
			m.defectCreateForm.storyID, m.defectCreateForm.storyLabel = item.ID, item.Label
			return m.loadDefectOpenSprints()
		}
	}
	return m, nil
}

func (m Model) updateDefectSprintSelect(key tea.KeyMsg) (Model, tea.Cmd) {
	switch m.defectSprintSelector.update(key) {
	case selectorBack:
		m.Screen = "defect-story-select"
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
	case "stories":
		if m.Screen != "defect-story-select" || !m.defectStorySelector.loading || result.requestID != m.defectStoryRequestID {
			return m, nil
		}
		if message := defectReadFailure(result.response, result.err); message != "" {
			m.defectStorySelector.fail(message)
			return m, nil
		}
		var listing struct {
			Stories []Story `json:"stories"`
		}
		if err := json.Unmarshal(result.response.Body, &listing); err != nil {
			m.defectStorySelector.fail(unreadableResponse(err))
			return m, nil
		}
		items := make([]selectorItem, 0, len(listing.Stories))
		for _, story := range listing.Stories {
			items = append(items, storySelectorItem(story))
		}
		m.defectStorySelector.setItems(items)
	case "sprints":
		if m.Screen != "defect-sprint-select" || !m.defectSprintSelector.loading || result.requestID != m.defectSprintRequestID {
			return m, nil
		}
		if message := defectReadFailure(result.response, result.err); message != "" {
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

func defectReadFailure(response APIResponse, err error) string {
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
	if f.storyID != "" && f.sprintID != "" {
		b.WriteString(hintStyle.Render("Borrador sin registrar · Esc cancela."))
	} else {
		b.WriteString(hintStyle.Render("Usá ←/→ para cambiar severidad o estado; Enter en el último campo elige la historia."))
	}
	return b.String()
}
