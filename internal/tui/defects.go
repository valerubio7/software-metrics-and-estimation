package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// toneOfSeverity maps a severity to the tone of its cue, so the list keeps the
// same color language as the severity badge.
func toneOfSeverity(severity string) tone {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "alta", "high", "critical", "critica", "crítica":
		return toneDanger
	case "media", "medium", "moderate":
		return toneWarn
	case "baja", "low", "minor":
		return toneInfo
	default:
		return toneNeutral
	}
}

// toneOfDefectStatus maps a defect status to the tone of its cue.
func toneOfDefectStatus(status string) tone {
	switch strings.ToLower(strings.TrimSpace(strings.ReplaceAll(status, "_", " "))) {
	case "detectado", "open", "abierto":
		return toneWarn
	case "en correccion", "en_correccion", "in progress":
		return toneInfo
	case "resuelto", "resolved", "closed", "cerrado":
		return toneOk
	default:
		return toneNeutral
	}
}

// defectStatusLabel renders the API status without its enum underscores.
func defectStatusLabel(status string) string {
	label := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(status, "_", " ")))
	if label == "en correccion" {
		return "en corrección"
	}
	return label
}

// defectRow renders one defect as a two-line block: the description with its
// severity and status cues, then the abbreviated references in a quieter line.
// When the title line cannot hold the cues without clipping, they move to the
// front of the second line so they are never hidden.
func (m Model) defectRow(selected bool, defect Defect) string {
	display := func(id string) string {
		if strings.TrimSpace(id) == "" {
			return "—"
		}
		return shortID(id)
	}
	cues := ""
	if defect.Severity != "" {
		cues = statusCue(defect.Severity, toneOfSeverity(defect.Severity))
	}
	if defect.Status != "" {
		if cues != "" {
			cues += "  "
		}
		cues += statusCue(defectStatusLabel(defect.Status), toneOfDefectStatus(defect.Status))
	}
	title := titleStyle.Render(defect.Description)
	if m.innerWidth() < 60 {
		// The narrow list has no preview pane: the references live in the
		// detail, so the cues get the whole second line.
		return m.twoLineRow(selected, title, cues)
	}
	refs := fmt.Sprintf("historia %s  ·  detección %s  ·  resolución %s  ·  %s",
		display(defect.StoryID), display(defect.SprintID), display(stringValue(defect.ResolutionSprintID)), display(defect.ID))
	if lipgloss.Width(title+cues) <= m.innerWidth()-2 {
		return m.twoLineRow(selected, title+"  "+cues, refs)
	}
	return m.twoLineRow(selected, title, cues+"  ·  "+refs)
}

func (m Model) viewDefects() string {
	var b strings.Builder
	b.WriteString(screenHeader("", "Defectos del proyecto "+m.projectLabel(), ""))
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
	if m.listAreaHeight() > 0 && available < 2 {
		available = 2
	}
	// Every defect takes two lines: title with cues and the references.
	start, end := windowBounds(m.defectCursor, len(visible), available/2)
	b.WriteString(sectionTitle("Defectos", m.defectCursor, len(visible), start, end) + "\n")
	for i := start; i < end; i++ {
		b.WriteString(m.defectRow(i == m.defectCursor, visible[i]) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m Model) viewDefectDetail() string {
	d := m.defectDetail
	display := func(id string) string {
		if strings.TrimSpace(id) == "" {
			return "—"
		}
		return shortID(id)
	}
	var b strings.Builder
	b.WriteString(screenHeader("", "Detalle del defecto", "Datos del defecto seleccionado."))
	b.WriteString("\n\n")
	b.WriteString(sectionLabel("Referencias") + "\n")
	b.WriteString(detailField("ID", display(d.ID)) + "\n")
	description := d.Description
	if strings.TrimSpace(description) == "" {
		description = "—"
	}
	b.WriteString(m.detailFieldWrapped("Descripción", description) + "\n")
	rows := [][2]string{
		{"Severidad", d.Severity},
		{"Estado", defectStatusLabel(d.Status)}, {"Historia relacionada", display(d.StoryID)},
		{"Sprint de detección", display(d.SprintID)}, {"Sprint de resolución", display(stringValue(d.ResolutionSprintID))},
	}
	for _, row := range rows {
		b.WriteString(detailField(row[0], row[1]))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
