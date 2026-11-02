package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Defect is a project-scoped defect as serialized by the defect API.
type Defect struct {
	ID                 string  `json:"id"`
	Description        string  `json:"description"`
	Severity           string  `json:"severity"`
	Status             string  `json:"status"`
	StoryID            string  `json:"story_id"`
	SprintID           string  `json:"sprint_id"`
	ResolutionSprintID *string `json:"resolution_sprint_id"`
}

// DefectClient reads defects for a project without exposing mutation methods.
type DefectClient struct{ api apiClient }

// NewDefectClient creates a read-only defect API client.
func NewDefectClient(baseURL string) *DefectClient {
	return &DefectClient{api: newAPIClient(baseURL)}
}

// List fetches the defects belonging to a project.
func (c *DefectClient) List(ctx context.Context, projectID string) (APIResponse, error) {
	path := "/projects/" + url.PathEscape(projectID) + "/defects"
	return c.api.do(ctx, http.MethodGet, path, nil)
}

// DefectResultMsg is the asynchronous result of a project defect list request.
type DefectResultMsg struct {
	ProjectID string
	Response  APIResponse
	Err       error
}

func (m Model) openDefects() (Model, tea.Cmd) {
	m.defectCursor = 0
	if m.activeProjectID == "" {
		return m.openProjectPicker("defects")
	}
	m.Screen = "defects"
	return m.startDefectLoad()
}

func (m Model) startDefectLoad() (Model, tea.Cmd) {
	m.defects, m.defectError, m.defectLoading, m.defectCursor = nil, "", true, 0
	client, projectID := m.defectClient, m.activeProjectID
	return m, func() tea.Msg {
		response, err := client.List(context.Background(), projectID)
		return DefectResultMsg{ProjectID: projectID, Response: response, Err: err}
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (m Model) updateDefects(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyEsc:
		m.Screen, m.Cursor = "menu", 0
	case tea.KeyUp:
		if m.defectCursor > 0 {
			m.defectCursor--
		}
	case tea.KeyDown:
		if m.defectCursor < len(m.defects)-1 {
			m.defectCursor++
		}
	case tea.KeyRunes:
		if string(key.Runes) == "r" {
			return m.startDefectLoad()
		}
	}
	return m, nil
}

func (m Model) applyDefectResult(result DefectResultMsg) (Model, tea.Cmd) {
	if result.ProjectID != m.activeProjectID || m.Screen != "defects" || !m.defectLoading {
		return m, nil
	}
	m.defectLoading = false
	if result.Err != nil {
		m.defectError = "Error de conexión: " + result.Err.Error()
		return m, nil
	}
	if result.Response.StatusCode < 200 || result.Response.StatusCode >= 300 {
		m.defectError = fmt.Sprintf("Error HTTP %d: %s", result.Response.StatusCode, strings.TrimSpace(string(result.Response.Body)))
		return m, nil
	}
	var listing struct {
		ProjectID string   `json:"project_id"`
		Defects   []Defect `json:"defects"`
	}
	if err := json.Unmarshal(result.Response.Body, &listing); err != nil {
		m.defectError = unreadableResponse(err)
		return m, nil
	}
	m.defects = listing.Defects
	return m, nil
}

func (m Model) viewDefects() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Defectos del proyecto " + m.projectLabel()))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Listado de solo lectura."))
	b.WriteString("\n\n")
	switch {
	case m.defectLoading:
		b.WriteString(statusMutedStyle.Render("Cargando defectos…"))
		return strings.TrimRight(b.String(), "\n")
	case m.defectError != "":
		b.WriteString(m.errorBlock(m.defectError))
		b.WriteString("\n\n" + hintStyle.Render("r para reintentar."))
		return strings.TrimRight(b.String(), "\n")
	case len(m.defects) == 0:
		b.WriteString(emptyState("◇", "No hay defectos registrados.", "Los defectos se registran desde el flujo explícito de alta."))
		return strings.TrimRight(b.String(), "\n")
	}
	b.WriteString(sectionLabel("Defectos") + "  " + hintStyle.Render(fmt.Sprintf("(%d)", len(m.defects))) + "\n")
	for i, defect := range m.defects {
		resolution := stringValue(defect.ResolutionSprintID)
		if resolution == "" {
			resolution = "—"
		}
		row := fmt.Sprintf("%s  ·  %s  ·  %s  ·  historia %s  ·  detección %s  ·  resolución %s  ·  %s", defect.Description, defect.Severity, defect.Status, defect.StoryID, defect.SprintID, resolution, defect.ID)
		b.WriteString(m.row(i == m.defectCursor, row) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
