package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// MetricsClient reads the metrics calculated by the API; it never writes.
type MetricsClient struct{ api apiClient }

func NewMetricsClient(baseURL string) *MetricsClient {
	return &MetricsClient{api: newAPIClient(baseURL)}
}
func (c *MetricsClient) Project(ctx context.Context, projectID string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodGet, "/projects/"+url.PathEscape(projectID)+"/metrics", nil)
}
func (c *MetricsClient) Sprint(ctx context.Context, projectID, sprintID string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodGet, "/projects/"+url.PathEscape(projectID)+"/sprints/"+url.PathEscape(sprintID)+"/metrics", nil)
}

// Metrics preserves null separately from a numeric zero returned by the API.
type Metrics struct {
	StoryPointsPlanned         *float64         `json:"story_points_planned"`
	StoryPointsCompleted       *float64         `json:"story_points_completed"`
	Velocity                   *float64         `json:"velocity"`
	EstimatedHours             *float64         `json:"estimated_hours"`
	ActualHours                *float64         `json:"actual_hours"`
	Deviation                  *float64         `json:"deviation"`
	DeviationPercentage        *float64         `json:"deviation_percentage"`
	CompletedStoriesPercentage *float64         `json:"completed_stories_percentage"`
	DefectsDetected            *float64         `json:"defects_detected"`
	DefectsResolved            *float64         `json:"defects_resolved"`
	Coverage                   *MetricsCoverage `json:"coverage"`
}

// MetricsCoverage mirrors the nested coverage object returned by US-19.
type MetricsCoverage struct {
	StoryPointsPlanned   MetricValueCoverage  `json:"story_points_planned"`
	StoryPointsCompleted MetricValueCoverage  `json:"story_points_completed"`
	Velocity             MetricValueCoverage  `json:"velocity"`
	EstimatedHours       MetricValueCoverage  `json:"estimated_hours"`
	ActualHours          MetricActualCoverage `json:"actual_hours"`
}

type MetricValueCoverage struct {
	MissingObservations int `json:"missing_observations"`
}

type MetricActualCoverage struct {
	StoriesWithoutRecords         int `json:"stories_without_records"`
	UnattributedStoryLevelRecords int `json:"unattributed_story_level_records"`
}

func decodeMetrics(body []byte) (Metrics, error) {
	var metrics Metrics
	err := json.Unmarshal(body, &metrics)
	return metrics, err
}

func metricValue(value *float64) string {
	if value == nil {
		return "sin datos"
	}
	return fmt.Sprintf("%g", *value)
}

func (m Model) viewMetrics() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Métricas · "+m.section.selected.Label) + "\n")
	if m.section.selector.err != "" {
		b.WriteString(m.errorBlock(m.section.selector.err) + "\n\n")
		b.WriteString(hintStyle.Render("r para reintentar · Esc para volver."))
		return b.String()
	}
	if m.section.selector.loading {
		return "Métricas\n\nCargando…"
	}
	if m.section.metrics == nil {
		return "Métricas\n\nSin respuesta de métricas."
	}
	metrics := m.section.metrics
	rows := []struct {
		label string
		value *float64
	}{
		{"Puntos planificados", metrics.StoryPointsPlanned}, {"Puntos completados", metrics.StoryPointsCompleted},
		{"Velocidad", metrics.Velocity}, {"Horas estimadas", metrics.EstimatedHours}, {"Horas reales", metrics.ActualHours},
		{"Desviación", metrics.Deviation}, {"Desviación %", metrics.DeviationPercentage},
		{"Historias completadas %", metrics.CompletedStoriesPercentage}, {"Defectos detectados", metrics.DefectsDetected},
		{"Defectos resueltos", metrics.DefectsResolved},
	}
	for _, row := range rows {
		b.WriteString(row.label + ": " + metricValue(row.value) + "\n")
	}
	b.WriteString("Cobertura\n")
	if metrics.Coverage == nil {
		b.WriteString("Cobertura: sin datos\n")
	} else {
		coverage := metrics.Coverage
		b.WriteString(fmt.Sprintf("Faltan puntos planificados: %d\n", coverage.StoryPointsPlanned.MissingObservations))
		b.WriteString(fmt.Sprintf("Faltan puntos completados: %d\n", coverage.StoryPointsCompleted.MissingObservations))
		b.WriteString(fmt.Sprintf("Faltan observaciones velocidad: %d\n", coverage.Velocity.MissingObservations))
		b.WriteString(fmt.Sprintf("Faltan horas estimadas: %d\n", coverage.EstimatedHours.MissingObservations))
		b.WriteString(fmt.Sprintf("Historias sin esfuerzo atribuible: %d\n", coverage.ActualHours.StoriesWithoutRecords))
		b.WriteString(fmt.Sprintf("Registros directos no atribuibles: %d\n", coverage.ActualHours.UnattributedStoryLevelRecords))
	}
	return strings.TrimRight(b.String(), "\n")
}
