package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var defectStatusOptions = []struct {
	label string
	api   string
}{
	{label: "Open", api: "detectado"},
	{label: "In Progress", api: "en_correccion"},
	{label: "Resolved", api: "resuelto"},
}

type defectStatusOptionsResult struct {
	ProjectID  string
	RequestID  uint64
	Open       APIResponse
	History    APIResponse
	OpenErr    error
	HistoryErr error
}

type defectStatusResult struct {
	ProjectID string
	RequestID uint64
	Response  APIResponse
	Err       error
}

func (m Model) beginDefectStatusUpdate() (Model, tea.Cmd) {
	m.defectStatusIndex = 0
	m.defectStatusMessage = ""
	m.Screen = "defect-status-select"
	return m, nil
}

func (m Model) updateDefectStatusSelect(key tea.KeyMsg) (Model, tea.Cmd) {
	if m.defectStatusLoading {
		return m, nil
	}
	switch key.Type {
	case tea.KeyEsc:
		m.Screen = "defect-detail"
	case tea.KeyUp:
		if m.defectStatusIndex > 0 {
			m.defectStatusIndex--
		}
	case tea.KeyDown:
		if m.defectStatusIndex < len(defectStatusOptions)-1 {
			m.defectStatusIndex++
		}
	case tea.KeyEnter:
		status := defectStatusOptions[m.defectStatusIndex].api
		if status == "resuelto" {
			return m.loadDefectResolutionSprints()
		}
		return m.submitDefectStatus(status, "")
	}
	return m, nil
}

func (m Model) loadDefectResolutionSprints() (Model, tea.Cmd) {
	m.Screen = "defect-resolution-select"
	m.defectStatusOptionsLoading = true
	m.defectResolutionSelector = selector{}
	m.defectResolutionSelector.startLoading()
	m.defectRequestSequence++
	requestID := m.defectRequestSequence
	m.defectStatusOptionsRequestID = requestID
	client, projectID := m.sprintClient, m.activeProjectID
	return m, func() tea.Msg {
		open, openErr := client.ListOpen(context.Background(), projectID)
		history, historyErr := client.History(context.Background(), projectID)
		return defectStatusOptionsResult{ProjectID: projectID, RequestID: requestID, Open: open, History: history, OpenErr: openErr, HistoryErr: historyErr}
	}
}

func (m Model) applyDefectStatusOptionsResult(result defectStatusOptionsResult) (Model, tea.Cmd) {
	if result.ProjectID != m.activeProjectID || m.Screen != "defect-resolution-select" || !m.defectStatusOptionsLoading || result.RequestID != m.defectStatusOptionsRequestID {
		return m, nil
	}
	m.defectStatusOptionsLoading = false
	if message := readFailure(result.Open, result.OpenErr); message != "" {
		m.defectResolutionSelector.fail(message)
		return m, nil
	}
	if message := readFailure(result.History, result.HistoryErr); message != "" {
		m.defectResolutionSelector.fail(message)
		return m, nil
	}
	var open struct {
		Sprints []Sprint `json:"sprints"`
	}
	if err := json.Unmarshal(result.Open.Body, &open); err != nil {
		m.defectResolutionSelector.fail(unreadableResponse(err))
		return m, nil
	}
	var history SprintHistory
	if err := json.Unmarshal(result.History.Body, &history); err != nil {
		m.defectResolutionSelector.fail(unreadableResponse(err))
		return m, nil
	}
	items := make([]selectorItem, 0, len(open.Sprints)+len(history.Sprints))
	for _, sprint := range open.Sprints {
		items = append(items, resolutionSprintItem(sprint.ID, sprint.SprintGoal, false))
	}
	for _, sprint := range history.Sprints {
		items = append(items, resolutionSprintItem(sprint.ID, sprint.SprintGoal, true))
	}
	m.defectResolutionSelector.setItems(items)
	return m, nil
}

func resolutionSprintItem(id, goal string, closed bool) selectorItem {
	label := strings.TrimSpace(goal)
	if label == "" {
		label = "(sin objetivo)"
	}
	state := "abierto"
	if closed {
		state = "cerrado"
	}
	return selectorItem{ID: id, Label: label, Meta: shortID(id) + " · " + state}
}

func (m Model) updateDefectResolutionSelect(key tea.KeyMsg) (Model, tea.Cmd) {
	if m.defectStatusLoading {
		return m, nil
	}
	switch m.defectResolutionSelector.update(key) {
	case selectorBack:
		m.defectStatusOptionsLoading = false
		m.Screen = "defect-status-select"
	case selectorRefresh:
		return m.loadDefectResolutionSprints()
	case selectorChosen:
		item, ok := m.defectResolutionSelector.selected()
		if ok {
			return m.submitDefectStatus("resuelto", item.ID)
		}
	}
	return m, nil
}

// submitDefectStatus sends the chosen status and stays on the screen that chose
// it until the API answers.
func (m Model) submitDefectStatus(status, resolutionSprintID string) (Model, tea.Cmd) {
	if m.defectStatusLoading {
		return m, nil
	}
	m.defectStatusLoading, m.defectStatusMessage, m.defectStatusPending = true, "", status
	m.defectRequestSequence++
	requestID := m.defectRequestSequence
	m.defectStatusRequestID = requestID
	client, projectID, defectID := m.defectClient, m.activeProjectID, m.defectDetail.ID
	return m, func() tea.Msg {
		response, err := client.UpdateStatus(context.Background(), projectID, defectID, status, resolutionSprintID)
		return defectStatusResult{ProjectID: projectID, RequestID: requestID, Response: response, Err: err}
	}
}

// applyDefectStatusResult returns to the detail of the defect with a toast on
// success. The detail and its row in the list are refreshed from the PATCH
// body (or with the sent status when the body is not the defect), so no extra
// request is needed. A rejection stays on the selector with the API reason.
func (m Model) applyDefectStatusResult(result defectStatusResult) (Model, tea.Cmd) {
	onSelector := m.Screen == "defect-status-select" || m.Screen == "defect-resolution-select"
	if result.ProjectID != m.activeProjectID || !onSelector || !m.defectStatusLoading || result.RequestID != m.defectStatusRequestID {
		return m, nil
	}
	m.defectStatusLoading = false
	if result.Err != nil {
		m.defectStatusMessage = "Error de conexión: " + result.Err.Error()
		return m, nil
	}
	if result.Response.StatusCode < 200 || result.Response.StatusCode >= 300 {
		m.defectStatusMessage = fmt.Sprintf("Error HTTP %d: %s", result.Response.StatusCode, strings.TrimSpace(string(result.Response.Body)))
		return m, nil
	}
	var updated Defect
	if json.Unmarshal(result.Response.Body, &updated) == nil && updated.ID == m.defectDetail.ID {
		m.defectDetail = updated
	} else {
		m.defectDetail.Status = m.defectStatusPending
	}
	defects := make([]Defect, len(m.defects))
	copy(defects, m.defects)
	for i := range defects {
		if defects[i].ID == m.defectDetail.ID {
			defects[i] = m.defectDetail
		}
	}
	m.defects = defects
	m.toast = "Estado del defecto actualizado: " + m.defectDetail.Status
	m.Screen = "defect-detail"
	return m, nil
}

// defectStatusLines renders the in-flight indicator and the API rejection of a
// status change below its selector.
func (m Model) defectStatusLines() string {
	if m.defectStatusLoading {
		return "\n\n" + statusMutedStyle.Render("Actualizando estado…")
	}
	if m.defectStatusMessage != "" {
		return "\n\n" + strings.TrimRight(m.errorBlock(m.defectStatusMessage), "\n")
	}
	return ""
}

func (m Model) viewDefectStatusSelect() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Cambiar estado del defecto"))
	b.WriteString("\n\n")
	for i, option := range defectStatusOptions {
		b.WriteString(m.row(i == m.defectStatusIndex, option.label) + "\n")
	}
	b.WriteString("\n" + hintStyle.Render("Elegí uno de los estados admitidos por la API."))
	return strings.TrimRight(b.String(), "\n") + m.defectStatusLines()
}
