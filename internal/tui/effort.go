package tui

import (
	"context"
	"net/http"
	"net/url"
)

// EffortInput is the body of POST /projects/{pid}/stories/{sid}/effort-records.
// An empty TaskID registers effort over the whole story and is omitted.
type EffortInput struct {
	MemberID    string  `json:"member_id"`
	TaskID      string  `json:"task_id,omitempty"`
	WorkDate    string  `json:"work_date"`
	Description string  `json:"description"`
	Hours       float64 `json:"hours"`
}

// EffortRecord is the flat 201 response of the effort registration endpoint.
type EffortRecord struct {
	ID          string  `json:"id"`
	ProjectID   string  `json:"project_id"`
	StoryID     string  `json:"story_id"`
	TaskID      *string `json:"task_id"`
	MemberID    string  `json:"member_id"`
	WorkDate    string  `json:"work_date"`
	Description string  `json:"description"`
	Hours       float64 `json:"hours"`
}

// EffortClient registers effort records through the HTTP API.
type EffortClient struct{ api apiClient }

// NewEffortClient creates an effort API client using the configured base URL.
func NewEffortClient(baseURL string) *EffortClient {
	return &EffortClient{api: newAPIClient(baseURL)}
}

// Register sends one effort record for a story. The API is not idempotent:
// every successful call creates a new record, so callers must not retry blindly
// after a transport error because the first request may already have been applied.
func (c *EffortClient) Register(ctx context.Context, projectID, storyID string, input EffortInput) (APIResponse, error) {
	return c.api.do(ctx, http.MethodPost, storiesPath(projectID)+"/"+url.PathEscape(storyID)+"/effort-records", input)
}
