package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type defectCreateResult struct {
	ProjectID string
	RequestID uint64
	Response  APIResponse
	Err       error
}

func (m Model) openDefectCreateConfirm() (Model, tea.Cmd) {
	m.Screen = "defect-create-confirm"
	return m, nil
}

func (m Model) updateDefectCreateConfirm(key tea.KeyMsg) (Model, tea.Cmd) {
	if m.defectCreateLoading {
		// The POST is not idempotent: no key may send it again or leave before
		// the answer is known.
		return m, nil
	}
	if key.Type == tea.KeyEsc {
		m.Screen = "defect-create-form"
		return m, nil
	}
	if key.Type == tea.KeyRunes {
		switch strings.ToLower(string(key.Runes)) {
		case "n":
			return m.cancelDefectCreation()
		case "s":
			return m.submitDefectCreate()
		}
	}
	return m, nil
}

// submitDefectCreate sends the confirmed draft and stays on the confirmation
// until the API answers.
func (m Model) submitDefectCreate() (Model, tea.Cmd) {
	m.Screen, m.defectCreateLoading, m.defectCreateMessage = "defect-create-confirm", true, ""
	m.defectRequestSequence++
	requestID := m.defectRequestSequence
	m.defectCreateRequestID = requestID
	client, projectID := m.defectClient, m.activeProjectID
	form := m.defectCreateForm
	input := DefectCreateInput{
		StoryID: form.storyID, SprintID: form.sprintID,
		Description: form.description, Severity: form.severity(), Status: form.status(),
	}
	return m, func() tea.Msg {
		response, err := client.Create(context.Background(), projectID, input)
		return defectCreateResult{ProjectID: projectID, RequestID: requestID, Response: response, Err: err}
	}
}

// applyDefectCreateResult returns to where the flow started with a toast on
// success: the reloaded defect list, or the story detail without reloading
// anything when the defect was reported from it. A rejection stays on the
// confirmation with the API reason.
func (m Model) applyDefectCreateResult(result defectCreateResult) (Model, tea.Cmd) {
	if result.ProjectID != m.activeProjectID || m.Screen != "defect-create-confirm" || !m.defectCreateLoading || result.RequestID != m.defectCreateRequestID {
		return m, nil
	}
	m.defectCreateLoading = false
	if result.Err != nil {
		m.defectCreateMessage = "Error de conexión: " + result.Err.Error()
		return m, nil
	}
	if result.Response.StatusCode < 200 || result.Response.StatusCode >= 300 {
		m.defectCreateMessage = fmt.Sprintf("Error HTTP %d: %s", result.Response.StatusCode, string(result.Response.Body))
		return m, nil
	}
	m.toast = "Defecto registrado: " + m.defectCreateForm.description
	return m.backToStoryDetail()
}

func (m Model) viewDefectCreateConfirm() string {
	f := m.defectCreateForm
	view := screenTitleStyle.Render("Confirmar defecto") + "\n\n" +
		fmt.Sprintf("Descripción: %s\nSeveridad: %s\nEstado: %s\nHistoria: %s\nSprint de detección: %s\n\n", f.description, f.severity(), f.status(), f.storyLabel, f.sprintLabel) +
		hintStyle.Render("s confirma y registra · n cancela · Esc vuelve al borrador.")
	if m.defectCreateLoading {
		view += "\n\n" + statusMutedStyle.Render("Registrando defecto…")
	}
	if m.defectCreateMessage != "" {
		view += "\n\n" + strings.TrimRight(m.errorBlock(m.defectCreateMessage), "\n")
	}
	return view
}
