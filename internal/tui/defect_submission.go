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
	if key.Type == tea.KeyEsc {
		m.Screen = "defect-create-form"
		return m, nil
	}
	if key.Type == tea.KeyRunes {
		switch strings.ToLower(string(key.Runes)) {
		case "n":
			m.Screen = "defects"
		case "s":
			return m.submitDefectCreate()
		}
	}
	return m, nil
}

func (m Model) submitDefectCreate() (Model, tea.Cmd) {
	m.Screen, m.defectCreateLoading, m.defectCreateOK = "defect-create-result", true, false
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

func (m Model) applyDefectCreateResult(result defectCreateResult) (Model, tea.Cmd) {
	if result.ProjectID != m.activeProjectID || m.Screen != "defect-create-result" || !m.defectCreateLoading || result.RequestID != m.defectCreateRequestID {
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
	m.defectCreateOK = true
	m.defectCreateMessage = fmt.Sprintf("Defecto registrado (HTTP %d):\n%s", result.Response.StatusCode, string(result.Response.Body))
	return m, nil
}

func (m Model) updateDefectCreateResult(key tea.KeyMsg) (Model, tea.Cmd) {
	if m.defectCreateLoading || (key.Type != tea.KeyEnter && key.Type != tea.KeyEsc) {
		return m, nil
	}
	if m.defectCreateOK {
		return m.openDefects()
	}
	m.Screen = "defect-create-form"
	return m, nil
}

func (m Model) viewDefectCreateConfirm() string {
	f := m.defectCreateForm
	return screenTitleStyle.Render("Confirmar defecto") + "\n\n" +
		fmt.Sprintf("Descripción: %s\nSeveridad: %s\nEstado: %s\nHistoria: %s\nSprint de detección: %s\n\n", f.description, f.severity(), f.status(), f.storyLabel, f.sprintLabel) +
		hintStyle.Render("s confirma y registra · n cancela · Esc vuelve al borrador.")
}

func (m Model) viewDefectCreateResult() string {
	if m.defectCreateLoading {
		return screenTitleStyle.Render("Registrar defecto") + "\n\n" + statusMutedStyle.Render("Registrando defecto…")
	}
	if !m.defectCreateOK {
		return screenTitleStyle.Render("Error al registrar el defecto") + "\n\n" + m.errorBlock(m.defectCreateMessage)
	}
	return screenTitleStyle.Render("Resultado") + "\n\n" + m.resultBlock(m.defectCreateMessage, true)
}
