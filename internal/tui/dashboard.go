package tui

import (
	"context"
	"encoding/json"
	"math"
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

// viewDashboardMetrics renders one line per metric plus a Unicode bar for each
// relation available. Values are never calculated here; formatMetric keeps nil
// as «sin datos».
func (m Model) viewDashboardMetrics() string {
	rows := dashboardMetricRows(m.dashboard.metrics)
	width := m.innerWidth()
	barWidth := dashboardBarWidth(width)
	var b strings.Builder
	for _, row := range rows {
		line := row.label + ": " + row.value
		for _, wrapped := range wrapPlain(line, width) {
			b.WriteString(wrapped)
			b.WriteString("\n")
		}
		if row.fraction != nil {
			b.WriteString(dashboardBar(*row.fraction, barWidth))
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// dashboardRow is one metric line. fraction is nil when the API cannot express
// a relation (missing value or zero denominator), so no bar is drawn.
type dashboardRow struct {
	label    string
	value    string
	fraction *float64
}

func dashboardMetricRows(metrics metricsPayload) []dashboardRow {
	storyPoints, storyPointsFraction := dashboardPair(metrics.StoryPointsCompleted, metrics.StoryPointsPlanned, "")
	hours, hoursFraction := dashboardPair(metrics.ActualHours, metrics.EstimatedHours, "h")
	stories, storiesFraction := dashboardPercentage(metrics.CompletedStoriesPercentage)
	defects, defectsFraction := dashboardPair(metrics.DefectsResolved, metrics.DefectsDetected, "")
	return []dashboardRow{
		{"Story Points", storyPoints, storyPointsFraction},
		{"Horas", hours, hoursFraction},
		{"Historias completadas", stories, storiesFraction},
		{"Defectos", defects, defectsFraction},
		{"Velocidad", formatMetric(metrics.Velocity, ""), nil},
		{"Desviación", formatMetric(metrics.Deviation, ""), nil},
	}
}

// dashboardPair displays both API values and returns the relation used by the
// bar. A missing value or a non-positive denominator yields no bar.
func dashboardPair(value, total *float64, unit string) (string, *float64) {
	if value == nil || total == nil {
		return "sin datos", nil
	}
	text := formatMetric(value, "") + " / " + formatMetric(total, "")
	if unit != "" {
		text += " " + unit
	}
	if *total <= 0 {
		return text, nil
	}
	fraction := *value / *total
	return text, &fraction
}

// dashboardPercentage uses the percentage returned by US-19; it only divides by
// 100 to size the bar, never to recalculate the metric.
func dashboardPercentage(value *float64) (string, *float64) {
	if value == nil {
		return "sin datos", nil
	}
	fraction := *value / 100
	return formatMetric(value, "%"), &fraction
}

// dashboardBarWidth keeps the bar inside the panel at 44 columns and leaves it
// at a readable size on wider terminals.
func dashboardBarWidth(inner int) int {
	width := inner - 6
	if width < 4 {
		width = 4
	}
	if width > 28 {
		width = 28
	}
	return width
}

// dashboardBar draws a Unicode bar. Fractions outside [0,1] are clamped so an
// overrun still fits; zero is a real value and renders an empty bar.
func dashboardBar(fraction float64, width int) string {
	if width <= 0 {
		return ""
	}
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	filled := int(math.Round(fraction * float64(width)))
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
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
