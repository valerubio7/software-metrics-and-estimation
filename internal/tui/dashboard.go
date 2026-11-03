package tui

import (
	"context"
	"encoding/json"
	"math"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// dashboardState is the read-only state of the project Dashboard. Both reads
// belong to the same load sequence so a late answer from a previous project,
// scope or refresh can be discarded. An empty sprintID is the whole project.
type dashboardState struct {
	seq         uint64
	sprintID    string
	sprintLabel string
	scopes      selector
	scopesSeq   uint64
	loading     bool
	status      string
	statusErr   string
	metrics     metricsPayload
	metricsOK   bool
	metricsErr  string
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
		return m.leaveWorkspace()
	}
	m.Screen = "dashboard"
	m.dashboard.sprintID, m.dashboard.sprintLabel = "", ""
	return m.reloadDashboard()
}

// reloadDashboard starts a new load of the status and the metrics of the
// active scope; earlier answers are discarded by the sequence.
func (m Model) reloadDashboard() (Model, tea.Cmd) {
	m.dashboard.seq++
	m.dashboard.loading = true
	m.dashboard.status, m.dashboard.statusErr = "", ""
	m.dashboard.metrics = metricsPayload{}
	m.dashboard.metricsOK, m.dashboard.metricsErr = false, ""
	return m, m.loadDashboardCmd()
}

// loadDashboardCmd reads the project status and the metrics of the active
// scope (the project or one Sprint). It returns
// one message so the screen leaves its loading state only when both reads end.
func (m Model) loadDashboardCmd() tea.Cmd {
	seq, projectID, sprintID := m.dashboard.seq, m.activeProjectID, m.dashboard.sprintID
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

		if sprintID != "" {
			response, err = metricsClient.Sprint(context.Background(), projectID, sprintID)
		} else {
			response, err = metricsClient.Project(context.Background(), projectID)
		}
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
// active Dashboard load, also while the scope selector covers it.
func (m Model) applyDashboardResult(result dashboardResultMsg) (Model, tea.Cmd) {
	if (m.Screen != "dashboard" && m.Screen != "dashboard-sprint-select") || !m.dashboard.loading || result.seq != m.dashboard.seq {
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
		return m.leaveWorkspace()
	case key.Type == tea.KeyRunes && string(key.Runes) == "r" && !m.dashboard.loading:
		return m.reloadDashboard()
	case key.Type == tea.KeyRunes && string(key.Runes) == "s":
		return m.openDashboardScopes()
	case key.Type == tea.KeyRunes && string(key.Runes) == "d":
		if m.dashboard.sprintID != "" {
			return m.startReport(sprintReportScope, m.activeProjectID, m.dashboard.sprintID)
		}
		return m.startReport(projectReportScope, m.activeProjectID, "")
	}
	return m, nil
}

// dashboardScopesMsg is the answer of the open Sprints read behind the scope
// selector.
type dashboardScopesMsg struct {
	seq      uint64
	response APIResponse
	err      error
}

// wholeProjectLabel is the first scope row; its empty ID means the project.
const wholeProjectLabel = "Proyecto completo"

// openDashboardScopes shows the scope selector and loads the open Sprints.
func (m Model) openDashboardScopes() (Model, tea.Cmd) {
	m.Screen = "dashboard-sprint-select"
	m.dashboard.scopes = selector{}
	return m.loadDashboardScopes()
}

func (m Model) loadDashboardScopes() (Model, tea.Cmd) {
	m.dashboard.scopesSeq++
	m.dashboard.scopes.startLoading()
	seq, projectID, client := m.dashboard.scopesSeq, m.activeProjectID, m.sprintClient
	return m, func() tea.Msg {
		response, err := client.ListOpen(context.Background(), projectID)
		return dashboardScopesMsg{seq: seq, response: response, err: err}
	}
}

func (m Model) applyDashboardScopes(result dashboardScopesMsg) (Model, tea.Cmd) {
	if m.Screen != "dashboard-sprint-select" || !m.dashboard.scopes.loading || result.seq != m.dashboard.scopesSeq {
		return m, nil
	}
	if message := sectionFailure(result.err, result.response); message != "" {
		m.dashboard.scopes.fail(message)
		return m, nil
	}
	sprints, ok := sprintSelectorItems(result.response.Body)
	if !ok {
		m.dashboard.scopes.fail("No se pudo interpretar la respuesta de la API.")
		return m, nil
	}
	items := append([]selectorItem{{Label: wholeProjectLabel, Meta: "métricas consolidadas"}}, sprints...)
	m.dashboard.scopes.setItems(items)
	return m, nil
}

// updateDashboardScopes drives the scope selector. Esc returns to the
// Dashboard without changing the scope; choosing a row reloads the Dashboard.
func (m Model) updateDashboardScopes(key tea.KeyMsg) (Model, tea.Cmd) {
	switch m.dashboard.scopes.update(key) {
	case selectorBack:
		m.Screen = "dashboard"
	case selectorRefresh:
		return m.loadDashboardScopes()
	case selectorChosen:
		item, _ := m.dashboard.scopes.selected()
		m.Screen = "dashboard"
		m.dashboard.sprintID, m.dashboard.sprintLabel = item.ID, ""
		if item.ID != "" {
			m.dashboard.sprintLabel = item.Label
		}
		return m.reloadDashboard()
	}
	return m, nil
}

func (m Model) viewDashboardScopes() string {
	return m.viewSelector(&m.dashboard.scopes, "Dashboard · Ámbito", "Elegí el proyecto completo o un Sprint abierto.", "No hay ámbitos disponibles.", "")
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
	if m.dashboard.sprintID != "" {
		b.WriteString(panelTitleStyle.Render("Métricas del Sprint"))
		b.WriteString("\n")
		b.WriteString(hintStyle.Render("Sprint: " + m.dashboard.sprintLabel))
	} else {
		b.WriteString(panelTitleStyle.Render("Métricas del proyecto"))
	}
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
	reportScope := projectReportScope
	if m.dashboard.sprintID != "" {
		reportScope = sprintReportScope
	}
	if block := m.reportView(reportScope, m.activeProjectID); block != "" {
		b.WriteString("\n\n")
		b.WriteString(block)
		b.WriteString("\n")
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
	return m.keyHints("r", "refrescar", "s", "ámbito", "d", "PDF", "Esc", "proyectos", "?", "ayuda", "q", "salir")
}

func dashboardHelpRows() [][2]string {
	return [][2]string{
		{"r", "Refrescar el estado y las métricas del proyecto"},
		{"s", "Elegir proyecto o Sprint para las métricas"},
		{"d", "Descargar el reporte PDF del ámbito mostrado"},
		{"Esc", "Volver a Proyectos"},
		{"?", "Mostrar u ocultar esta ayuda"},
		{"q", "Salir"},
		{"Ctrl+C", "Salir desde cualquier pantalla"},
	}
}

func (m Model) dashboardScopesFooter() string {
	s := &m.dashboard.scopes
	switch {
	case s.filtering:
		return m.keyHints("escribí", "para filtrar", "↑/↓", "mover", "Enter", "dejar de filtrar", "Esc", "limpiar")
	case s.loading:
		return m.keyHints("Esc", "volver", "?", "ayuda", "q", "salir")
	case s.err != "":
		return m.keyHints("r", "reintentar", "Esc", "volver", "?", "ayuda", "q", "salir")
	}
	pairs := []string{"↑/↓", "mover"}
	if len(s.visible()) > 0 {
		pairs = append(pairs, "Enter", "elegir ámbito")
	}
	pairs = append(pairs, "/", "buscar", "r", "refrescar", "Esc", "volver", "?", "ayuda", "q", "salir")
	return m.keyHints(pairs...)
}

func (m Model) dashboardScopesHelpRows() [][2]string {
	rows := [][2]string{{"↑ / ↓", "Mover la selección"}}
	if !m.dashboard.scopes.loading && m.dashboard.scopes.err == "" && len(m.dashboard.scopes.visible()) > 0 {
		rows = append(rows, [2]string{"Enter", "Mostrar las métricas del ámbito resaltado"})
	}
	return append(rows,
		[2]string{"/", "Buscar o filtrar"},
		[2]string{"r", "Refrescar los Sprints abiertos"},
		[2]string{"Esc", "Limpiar el filtro o volver al Dashboard sin cambiar el ámbito"},
		[2]string{"?", "Mostrar u ocultar esta ayuda"},
		[2]string{"q", "Salir (no dentro de un filtro)"},
		[2]string{"Ctrl+C", "Salir desde cualquier pantalla"},
	)
}
