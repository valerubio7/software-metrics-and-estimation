package tui

import (
	"context"
	"encoding/json"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// dashboardState is the read-only state of the project Dashboard. Both reads
// belong to the same load sequence so a late answer from a previous project or
// refresh can be discarded.
type dashboardState struct {
	seq        uint64
	loading    bool
	status     string
	statusErr  string
	metrics    metricsPayload
	metricsOK  bool
	metricsErr string
}

// dashboardResultMsg is the asynchronous result of the project status and
// metrics reads issued by openDashboard and its refresh.
type dashboardResultMsg struct {
	seq        uint64
	status     string
	statusErr  string
	metrics    metricsPayload
	metricsOK  bool
	metricsErr string
}

// openDashboard enters the Dashboard for the active project, or asks for a
// project first when the session has none.
func (m Model) openDashboard() (Model, tea.Cmd) {
	if m.activeProjectID == "" {
		return m.openProjectPicker("dashboard")
	}
	m.Screen = "dashboard"
	m.dashboard.seq++
	m.dashboard.loading = true
	m.dashboard.status, m.dashboard.statusErr = "", ""
	m.dashboard.metrics = metricsPayload{}
	m.dashboard.metricsOK, m.dashboard.metricsErr = false, ""
	return m, m.loadDashboardCmd()
}

// loadDashboardCmd reads the project status and the project metrics. It returns
// one message so the screen leaves its loading state only when both reads end.
func (m Model) loadDashboardCmd() tea.Cmd {
	seq, projectID := m.dashboard.seq, m.activeProjectID
	projectClient, metricsClient := m.client, m.metricsClient
	return func() tea.Msg {
		result := dashboardResultMsg{seq: seq}

		response, err := projectClient.Status(context.Background(), projectID)
		if message := sectionFailure(err, response); message != "" {
			result.statusErr = message
		} else {
			var status struct {
				Status string `json:"status"`
			}
			if json.Unmarshal(response.Body, &status) != nil {
				result.statusErr = "No se pudo interpretar la respuesta de la API."
			} else {
				result.status = status.Status
			}
		}

		response, err = metricsClient.Project(context.Background(), projectID)
		if message := sectionFailure(err, response); message != "" {
			result.metricsErr = message
		} else if metrics, ok := parseMetricsPayload(response.Body); !ok {
			result.metricsErr = "No se pudo interpretar la respuesta de la API."
		} else {
			result.metrics, result.metricsOK = metrics, true
		}
		return result
	}
}

// applyDashboardResult stores the answer only when it still belongs to the
// active Dashboard load.
func (m Model) applyDashboardResult(result dashboardResultMsg) (Model, tea.Cmd) {
	if m.Screen != "dashboard" || !m.dashboard.loading || result.seq != m.dashboard.seq {
		return m, nil
	}
	m.dashboard.loading = false
	m.dashboard.status, m.dashboard.statusErr = result.status, result.statusErr
	m.dashboard.metrics, m.dashboard.metricsOK, m.dashboard.metricsErr = result.metrics, result.metricsOK, result.metricsErr
	return m, nil
}

// updateDashboard handles the read-only Dashboard keys. The global Update
// already handles help, project switching, quitting and Ctrl+C.
func (m Model) updateDashboard(key tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Type == tea.KeyEsc:
		m.Screen, m.Cursor = "menu", 0
		return m, nil
	case key.Type == tea.KeyRunes && string(key.Runes) == "r" && !m.dashboard.loading:
		m.dashboard.seq++
		m.dashboard.loading = true
		m.dashboard.statusErr, m.dashboard.metricsErr = "", ""
		return m, m.loadDashboardCmd()
	}
	return m, nil
}

// viewDashboard renders the project status and the metrics returned by the API.
// The values are never calculated here; formatMetric keeps nil as «sin datos».
func (m Model) viewDashboard() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Dashboard"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Proyecto activo: " + m.dashboardProjectName()))
	b.WriteString("\n\n")
	b.WriteString("Estado: ")
	switch {
	case m.dashboard.loading && m.dashboard.status == "" && m.dashboard.statusErr == "":
		b.WriteString(statusMutedStyle.Render("Cargando…"))
	case m.dashboard.statusErr != "" || strings.TrimSpace(m.dashboard.status) == "":
		b.WriteString("sin datos")
	default:
		b.WriteString(projectStatusBadge(m.dashboard.status))
	}
	b.WriteString("\n")
	if m.dashboard.statusErr != "" {
		b.WriteString(m.errorBlock(m.dashboard.statusErr))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(panelTitleStyle.Render("Métricas del proyecto"))
	b.WriteString("\n")
	switch {
	case m.dashboard.loading:
		b.WriteString(statusMutedStyle.Render("Cargando métricas…"))
	case m.dashboard.metricsErr != "":
		b.WriteString(m.errorBlock(m.dashboard.metricsErr))
		b.WriteString("\n")
		b.WriteString(hintStyle.Render("r para reintentar · Esc para volver."))
	case !m.dashboard.metricsOK:
		b.WriteString(statusMutedStyle.Render("Cargando…"))
	default:
		b.WriteString(m.viewDashboardMetrics())
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m Model) dashboardProjectName() string {
	if strings.TrimSpace(m.activeProjectName) != "" {
		return m.activeProjectName
	}
	if m.activeProjectID != "" {
		return shortID(m.activeProjectID)
	}
	return "sin datos"
}

// viewDashboardMetrics renders one line per metric. The chart rows arrive with
// TUI-11's graphics unit; text values already satisfy the consistency rule.
func (m Model) viewDashboardMetrics() string {
	rows := [][2]string{
		{"Story Points", dashboardRatio(m.dashboard.metrics.StoryPointsCompleted, m.dashboard.metrics.StoryPointsPlanned, "")},
		{"Horas", dashboardRatio(m.dashboard.metrics.ActualHours, m.dashboard.metrics.EstimatedHours, "h")},
		{"Historias completadas", formatMetric(m.dashboard.metrics.CompletedStoriesPercentage, "%")},
		{"Defectos", dashboardRatio(m.dashboard.metrics.DefectsResolved, m.dashboard.metrics.DefectsDetected, "")},
		{"Velocidad", formatMetric(m.dashboard.metrics.Velocity, "")},
		{"Desviación", formatMetric(m.dashboard.metrics.Deviation, "")},
	}
	var b strings.Builder
	width := m.innerWidth()
	for _, row := range rows {
		line := row[0] + ": " + row[1]
		for _, wrapped := range wrapPlain(line, width) {
			b.WriteString(wrapped)
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// dashboardRatio displays a pair of API values without calculating a new
// business metric. It is the textual base of the Dashboard chart rows.
func dashboardRatio(value, total *float64, unit string) string {
	if value == nil || total == nil {
		return "sin datos"
	}
	text := formatMetric(value, "") + " / " + formatMetric(total, "")
	if unit != "" {
		text += " " + unit
	}
	return text
}

func (m Model) dashboardFooter() string {
	return m.keyHints("r", "refrescar", "p", "proyecto", "Esc", "menú", "?", "ayuda", "q", "salir")
}

func dashboardHelpRows() [][2]string {
	return [][2]string{
		{"r", "Refrescar el estado y las métricas del proyecto"},
		{"p", "Cambiar el proyecto activo"},
		{"Esc", "Volver al menú"},
		{"?", "Mostrar u ocultar esta ayuda"},
		{"q", "Salir"},
		{"Ctrl+C", "Salir desde cualquier pantalla"},
	}
}
