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

// DefectClient reads the project-scoped defect collection.
type DefectClient struct{ api apiClient }

// DefectCreateInput contains the exact fields accepted by defect registration.
type DefectCreateInput struct {
	StoryID     string `json:"story_id"`
	SprintID    string `json:"sprint_id"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Status      string `json:"status"`
}

// NewDefectClient creates a read-only defect API client.
func NewDefectClient(baseURL string) *DefectClient {
	return &DefectClient{api: newAPIClient(baseURL)}
}

// List fetches the defects belonging to a project.
func (c *DefectClient) List(ctx context.Context, projectID string) (APIResponse, error) {
	path := "/projects/" + url.PathEscape(projectID) + "/defects"
	return c.api.do(ctx, http.MethodGet, path, nil)
}

// Create registers a defect after explicit confirmation in the TUI.
func (c *DefectClient) Create(ctx context.Context, projectID string, input DefectCreateInput) (APIResponse, error) {
	path := "/projects/" + url.PathEscape(projectID) + "/defects"
	return c.api.do(ctx, http.MethodPost, path, input)
}

// UpdateStatus updates a defect after explicit confirmation in the TUI.
func (c *DefectClient) UpdateStatus(ctx context.Context, projectID, defectID, status, resolutionSprintID string) (APIResponse, error) {
	input := struct {
		Status             string  `json:"status"`
		ResolutionSprintID *string `json:"resolution_sprint_id,omitempty"`
	}{Status: status}
	if status == "resuelto" {
		input.ResolutionSprintID = &resolutionSprintID
	}
	path := "/projects/" + url.PathEscape(projectID) + "/defects/" + url.PathEscape(defectID)
	return c.api.do(ctx, http.MethodPatch, path, input)
}

// DefectResultMsg is the asynchronous result of a project defect list request.
type DefectResultMsg struct {
	ProjectID string
	Response  APIResponse
	Err       error
}

func (m Model) openDefects() (Model, tea.Cmd) {
	m.defectCursor, m.defectFilter, m.defectFiltering = 0, "", false
	if m.activeProjectID == "" {
		return m.leaveWorkspace()
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

func (m Model) visibleDefects() []Defect {
	filter := strings.ToLower(strings.TrimSpace(m.defectFilter))
	if filter == "" {
		return m.defects
	}
	visible := make([]Defect, 0, len(m.defects))
	for _, defect := range m.defects {
		searchable := strings.Join([]string{
			defect.ID, defect.Description, defect.Severity, defect.Status,
			defect.StoryID, defect.SprintID, stringValue(defect.ResolutionSprintID),
		}, " ")
		if strings.Contains(strings.ToLower(searchable), filter) {
			visible = append(visible, defect)
		}
	}
	return visible
}

func (m Model) updateDefects(key tea.KeyMsg) (Model, tea.Cmd) {
	visible := m.visibleDefects()
	if m.defectFiltering {
		switch key.Type {
		case tea.KeyEsc:
			m.defectFiltering, m.defectFilter, m.defectCursor = false, "", 0
			return m, nil
		case tea.KeyEnter:
			m.defectFiltering = false
			return m, nil
		case tea.KeyUp:
			if m.defectCursor > 0 {
				m.defectCursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.defectCursor < len(visible)-1 {
				m.defectCursor++
			}
			return m, nil
		}
		if value, ok := editText(m.defectFilter, key); ok {
			m.defectFilter, m.defectCursor = value, 0
		}
		return m, nil
	}
	switch key.Type {
	case tea.KeyEsc:
		if m.defectFilter != "" {
			m.defectFilter, m.defectCursor = "", 0
			return m, nil
		}
		return m.leaveWorkspace()
	case tea.KeyUp:
		if m.defectCursor > 0 {
			m.defectCursor--
		}
	case tea.KeyDown:
		if m.defectCursor < len(visible)-1 {
			m.defectCursor++
		}
	case tea.KeyEnter:
		if m.defectCursor >= 0 && m.defectCursor < len(visible) {
			m.defectDetail = visible[m.defectCursor]
			m.Screen = "defect-detail"
		}
	case tea.KeyRunes:
		switch string(key.Runes) {
		case "u":
			visible := m.visibleDefects()
			if m.defectCursor >= 0 && m.defectCursor < len(visible) {
				m.defectDetail = visible[m.defectCursor]
				return m.beginDefectStatusUpdate()
			}
		case "/":
			m.defectFiltering = true
		case "r":
			return m.startDefectLoad()
		default:
			m.defectFiltering = true
			if value, ok := editText(m.defectFilter, key); ok {
				m.defectFilter, m.defectCursor = value, 0
			}
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
	b.WriteString(m.sectionIntroView(m.innerWidth()))
	b.WriteString("\n\n")
	if m.defectFiltering || m.defectFilter != "" {
		b.WriteString(searchLine(m.defectFilter, m.defectFiltering) + "\n\n")
	}
	switch {
	case m.defectLoading:
		b.WriteString(statusMutedStyle.Render("Cargando defectos…"))
		return strings.TrimRight(b.String(), "\n")
	case m.defectError != "":
		b.WriteString(m.errorBlock(m.defectError))
		return strings.TrimRight(b.String(), "\n")
	case len(m.defects) == 0:
		b.WriteString(emptyState("◇", "No hay defectos registrados.", "Reportá uno desde el detalle de una historia en Backlog o Sprints."))
		return strings.TrimRight(b.String(), "\n")
	}
	visible := m.visibleDefects()
	if len(visible) == 0 {
		b.WriteString(emptyState("⌕", "Sin coincidencias.", "Probá con otro texto o borrá el filtro con Esc."))
		return strings.TrimRight(b.String(), "\n")
	}
	used := strings.Count(b.String(), "\n")
	available := m.listAreaHeight() - used - 1
	if m.listAreaHeight() > 0 && available < 1 {
		available = 1
	}
	start, end := windowBounds(m.defectCursor, len(visible), available)
	b.WriteString(sectionTitle("Defectos", m.defectCursor, len(visible), start, end) + "\n")
	for i := start; i < end; i++ {
		defect := visible[i]
		resolution := stringValue(defect.ResolutionSprintID)
		if resolution == "" {
			resolution = "—"
		}
		row := fmt.Sprintf("%s  ·  %s  ·  %s  ·  historia %s  ·  detección %s  ·  resolución %s  ·  %s", defect.Description, defect.Severity, defect.Status, defect.StoryID, defect.SprintID, resolution, defect.ID)
		b.WriteString(m.row(i == m.defectCursor, row) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m Model) viewDefectDetail() string {
	d := m.defectDetail
	resolution := stringValue(d.ResolutionSprintID)
	if resolution == "" {
		resolution = "—"
	}
	rows := [][2]string{
		{"ID", d.ID}, {"Descripción", d.Description}, {"Severidad", d.Severity},
		{"Estado", d.Status}, {"Historia relacionada", d.StoryID},
		{"Sprint de detección", d.SprintID}, {"Sprint de resolución", resolution},
	}
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Detalle del defecto"))
	b.WriteString("\n\n")
	for _, row := range rows {
		b.WriteString(row[0] + ":\n")
		for _, line := range wrapPlain(row[1], m.innerWidth()) {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
