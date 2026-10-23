package tui

import (
	"context"
	"net/http"
	"net/url"
)

// Story is a user story as serialized by the backlog API.
type Story struct {
	ID                 string   `json:"id"`
	ProjectID          string   `json:"project_id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Priority           string   `json:"priority"`
	Status             string   `json:"status"`
	StoryPoints        *int     `json:"story_points"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	EstimatedHours     *float64 `json:"estimated_hours"`
}

// StoryCreateInput is the exact body accepted by POST /projects/{pid}/stories.
type StoryCreateInput struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Priority           string   `json:"priority"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
}

// StoryUpdateInput is the full-replacement body of PUT /projects/{pid}/stories/{sid}.
type StoryUpdateInput struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Priority           string   `json:"priority"`
	Status             string   `json:"status"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	EstimatedHours     *float64 `json:"estimated_hours"`
}

// BacklogClient calls the existing HTTP story endpoints without duplicating domain validation.
type BacklogClient struct{ api apiClient }

// NewBacklogClient creates a backlog API client using the configured base URL.
func NewBacklogClient(baseURL string) *BacklogClient {
	return &BacklogClient{api: newAPIClient(baseURL)}
}

// List fetches the backlog of a project.
func (c *BacklogClient) List(ctx context.Context, projectID string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodGet, storiesPath(projectID), nil)
}

// Create adds a story to the backlog of a project.
func (c *BacklogClient) Create(ctx context.Context, projectID string, input StoryCreateInput) (APIResponse, error) {
	return c.api.do(ctx, http.MethodPost, storiesPath(projectID), input)
}

// Update replaces a story of a project.
func (c *BacklogClient) Update(ctx context.Context, projectID, storyID string, input StoryUpdateInput) (APIResponse, error) {
	return c.api.do(ctx, http.MethodPut, storiesPath(projectID)+"/"+url.PathEscape(storyID), input)
}

func storiesPath(projectID string) string {
	return "/projects/" + url.PathEscape(projectID) + "/stories"
}

// StoryPointsClient writes an explicitly agreed story estimate.
type StoryPointsClient struct{ api apiClient }

func NewStoryPointsClient(baseURL string) *StoryPointsClient {
	return &StoryPointsClient{api: newAPIClient(baseURL)}
}
func (c *StoryPointsClient) Set(ctx context.Context, projectID, storyID string, points int) (APIResponse, error) {
	return c.api.do(ctx, http.MethodPut, storiesPath(projectID)+"/"+url.PathEscape(storyID)+"/story-points", struct {
		StoryPoints int `json:"story_points"`
	}{points})
}

// BacklogOp identifies which backlog operation produced a BacklogResultMsg.
type BacklogOp string

const (
	BacklogOpList   BacklogOp = "list"
	BacklogOpCreate BacklogOp = "create"
	BacklogOpUpdate BacklogOp = "update"
)

// BacklogResultMsg is the asynchronous result of a backlog API request.
type BacklogResultMsg struct {
	Op       BacklogOp
	Response APIResponse
	Err      error
}
