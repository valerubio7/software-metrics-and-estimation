package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// viewDashboard renders the status and the metrics of the active scope. The
// values are never calculated here; formatMetric keeps nil as «sin datos».
func (m Model) viewDashboard() string {
	var b strings.Builder
	scope := "Métricas del proyecto completo"
	if m.dashboard.sprintID != "" {
		scope = "Métricas del Sprint · " + m.dashboard.sprintLabel
	}
	b.WriteString(screenHeader("", scope, "Proyecto activo: "+m.dashboardProjectName()))
	b.WriteString("\n")
	b.WriteString(m.sectionIntroView(m.innerWidth()))
	b.WriteString("\n\n")
	// The workspace header shows the project status on wide terminals, but the
	// compact header drops it; the body keeps one always-visible status line.
	b.WriteString(detailField("Estado", m.dashboardStatusLabel()))
	b.WriteString("\n")
	if m.dashboard.statusErr != "" {
		b.WriteString(m.errorBlock(m.dashboard.statusErr))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	switch {
	case m.dashboard.loading:
		b.WriteString(statusMutedStyle.Render("Cargando métricas…"))
	case m.dashboard.metricsErr != "":
		b.WriteString(m.errorBlock(m.dashboard.metricsErr))
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

// dashboardStatusLabel keeps the status line readable in every state: loading,
// missing data or the API badge.
func (m Model) dashboardStatusLabel() string {
	switch {
	case m.dashboard.loading && m.dashboard.status == "" && m.dashboard.statusErr == "":
		return statusMutedStyle.Render("Cargando…")
	case m.dashboard.statusErr != "" || strings.TrimSpace(m.dashboard.status) == "":
		return "sin datos"
	default:
		return projectStatusBadge(m.dashboard.status)
	}
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

// viewDashboardMetrics renders the six metrics as a compact table: one
// "Label: value" line per metric and, when the API expresses a relation, a
// styled bar aligned to the right edge. Values are never calculated here.
func (m Model) viewDashboardMetrics() string {
	rows := dashboardMetricRows(m.dashboard.metrics)
	inner := m.innerWidth()
	if inner < 60 {
		for i := range rows {
			if rows[i].label == "Historias completadas" {
				rows[i].label = "Historias"
			}
		}
	}
	// Keep the exact "Label: value" shape of the dashboard contract; the
	// alignment happens after the value so the bar can use the remaining width.
	prefixes := make([]string, len(rows))
	maxPrefix := 0
	for i, row := range rows {
		head := "  " + row.label + ": "
		value := row.value
		if limit := inner - lipgloss.Width(head) - 5; lipgloss.Width(value) > limit {
			value = truncate(value, limit)
		}
		prefixes[i] = head + value
		if width := lipgloss.Width(prefixes[i]); width > maxPrefix {
			maxPrefix = width
		}
	}
	rightWidth := inner - maxPrefix - 1
	if rightWidth < 4 {
		rightWidth = 4
	}
	if rightWidth > 32 {
		rightWidth = 32
	}
	suffixWidth := 0
	for _, row := range rows {
		if row.fraction != nil && !strings.HasSuffix(row.value, "%") {
			if width := 1 + lipgloss.Width(dashboardPercent(*row.fraction)); width > suffixWidth {
				suffixWidth = width
			}
		}
	}
	if rightWidth-suffixWidth < 4 {
		suffixWidth = 0
	}

	var b strings.Builder
	writeSection := func(title string, start, end int) {
		b.WriteString(sectionLabel(title))
		b.WriteString("\n")
		for i := start; i < end; i++ {
			b.WriteString(dashboardMetricLine(rows[i], prefixes[i], inner, rightWidth, suffixWidth))
			b.WriteString("\n")
		}
	}
	writeSection("Avance", 0, 4)
	b.WriteString("\n")
	writeSection("Ritmo", 4, len(rows))
	return strings.TrimRight(b.String(), "\n")
}

// dashboardMetricLine renders one table row. The left side always keeps the
// "Label: value" shape and the optional bar is right-aligned in the panel.
func dashboardMetricLine(row dashboardRow, prefix string, inner, rightWidth, suffixWidth int) string {
	if row.fraction == nil {
		return prefix
	}
	available := inner - lipgloss.Width(prefix) - 1
	if rightWidth > available {
		rightWidth = available
	}
	if rightWidth < 4 {
		return prefix
	}
	meta := ""
	if suffixWidth > 0 && !strings.HasSuffix(row.value, "%") && rightWidth >= suffixWidth+4 {
		rightWidth -= suffixWidth
		meta = " " + dashboardPercent(*row.fraction)
	}
	bar := styledBar(*row.fraction, rightWidth) + metricMetaStyle.Render(meta)
	gap := inner - lipgloss.Width(prefix) - lipgloss.Width(bar)
	if gap < 1 {
		gap = 1
	}
	return prefix + strings.Repeat(" ", gap) + bar
}

// dashboardPercent rounds a relation to the whole percentage shown beside a bar.
func dashboardPercent(fraction float64) string {
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	return fmt.Sprintf("%.0f%%", fraction*100)
}

// styledBar draws the same Unicode bar as dashboardBar, with the filled cells in
// the accent color and the track in the hairline color.
func styledBar(fraction float64, width int) string {
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
	fill := lipgloss.NewStyle().Foreground(accent).Render(strings.Repeat("█", filled))
	track := lipgloss.NewStyle().Foreground(line).Render(strings.Repeat("░", width-filled))
	return fill + track
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
