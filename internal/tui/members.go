package tui

import (
	"context"
	"net/http"
	"net/url"
)

// Member is a project member as serialized by GET /projects/{id}/members.
type Member struct {
	ID        string  `json:"id"`
	ProjectID string  `json:"project_id"`
	FullName  string  `json:"full_name"`
	Email     *string `json:"email"`
}

// MemberClient calls the read-only member collection endpoint.
type MemberClient struct{ api apiClient }

// NewMemberClient creates a member API client using the configured base URL.
func NewMemberClient(baseURL string) *MemberClient {
	return &MemberClient{api: newAPIClient(baseURL)}
}

// List fetches the members of a project.
func (c *MemberClient) List(ctx context.Context, projectID string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodGet, "/projects/"+url.PathEscape(projectID)+"/members", nil)
}
