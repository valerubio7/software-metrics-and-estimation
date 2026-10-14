package tui

import (
	"context"
	"net/http"
	"net/url"
)

// ProjectInput contains the fields accepted by the existing project API.
type ProjectInput struct {
	Name              string `json:"name"`
	StartDate         string `json:"start_date"`
	PlannedFinishDate string `json:"planned_finish_date"`
}

// ProjectClient calls the existing HTTP project endpoints without duplicating domain validation.
type ProjectClient struct {
	api apiClient
}

// NewProjectClient creates a project API client using the configured base URL.
func NewProjectClient(baseURL string) *ProjectClient {
	return &ProjectClient{api: newAPIClient(baseURL)}
}

func (c *ProjectClient) Create(ctx context.Context, input ProjectInput) (APIResponse, error) {
	return c.api.do(ctx, http.MethodPost, "/projects", input)
}

func (c *ProjectClient) Update(ctx context.Context, id string, input ProjectInput) (APIResponse, error) {
	return c.api.do(ctx, http.MethodPut, "/projects/"+url.PathEscape(id), input)
}

func (c *ProjectClient) Status(ctx context.Context, id string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodGet, "/projects/"+url.PathEscape(id), nil)
}
