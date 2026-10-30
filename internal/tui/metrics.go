package tui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

type MetricsClient struct{ api apiClient }

func NewMetricsClient(baseURL string) *MetricsClient { return &MetricsClient{api: newAPIClient(baseURL)} }

// Project fetches read-only project metrics.
func (c *MetricsClient) Project(ctx context.Context, projectID string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodGet, "/projects/"+url.PathEscape(projectID)+"/metrics", nil)
}

// Sprint fetches read-only Sprint metrics for a project.
func (c *MetricsClient) Sprint(ctx context.Context, projectID, sprintID string) (APIResponse, error) {
	path := "/projects/" + url.PathEscape(projectID) + "/sprints/" + url.PathEscape(sprintID) + "/metrics"
	return c.api.do(ctx, http.MethodGet, path, nil)
}

type metricsPayload struct {
	ProjectID                  string   `json:"project_id"`
	SprintID                   string   `json:"sprint_id"`
	StoryPointsPlanned         *float64 `json:"story_points_planned"`
	StoryPointsCompleted       *float64 `json:"story_points_completed"`
	Velocity                   *float64 `json:"velocity"`
	EstimatedHours             *float64 `json:"estimated_hours"`
	ActualHours                *float64 `json:"actual_hours"`
	Deviation                  *float64 `json:"deviation"`
	CompletedStoriesPercentage *float64 `json:"completed_stories_percentage"`
	DefectsDetected            *float64 `json:"defects_detected"`
	DefectsResolved            *float64 `json:"defects_resolved"`
}

func parseMetricsPayload(body []byte) (metricsPayload, bool) {
	var payload metricsPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return metricsPayload{}, false
	}
	return payload, true
}

func formatMetric(value *float64, unit string) string {
	if value == nil {
		return "sin datos"
	}
	text := strconv.FormatFloat(*value, 'f', -1, 64)
	if unit == "" {
		return text
	}
	return text + unit
}
