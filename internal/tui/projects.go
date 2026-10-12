package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ProjectInput contains the fields accepted by the existing project API.
type ProjectInput struct {
	Name              string `json:"name"`
	StartDate         string `json:"start_date"`
	PlannedFinishDate string `json:"planned_finish_date"`
}

// APIResponse preserves the API status and body, including validation details.
type APIResponse struct {
	StatusCode int
	Body       []byte
}

// ProjectClient calls the existing HTTP project endpoints without duplicating domain validation.
type ProjectClient struct {
	baseURL string
	http    *http.Client
}

// NewProjectClient creates a project API client using the configured base URL.
func NewProjectClient(baseURL string) *ProjectClient {
	return &ProjectClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *ProjectClient) Create(ctx context.Context, input ProjectInput) (APIResponse, error) {
	return c.request(ctx, http.MethodPost, "/projects", input)
}

func (c *ProjectClient) Update(ctx context.Context, id string, input ProjectInput) (APIResponse, error) {
	return c.request(ctx, http.MethodPut, "/projects/"+url.PathEscape(id), input)
}

func (c *ProjectClient) Status(ctx context.Context, id string) (APIResponse, error) {
	return c.request(ctx, http.MethodGet, "/projects/"+url.PathEscape(id), nil)
}

func (c *ProjectClient) request(ctx context.Context, method, path string, input any) (APIResponse, error) {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return APIResponse{}, err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return APIResponse{}, err
	}
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return APIResponse{}, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return APIResponse{}, err
	}
	return APIResponse{StatusCode: response.StatusCode, Body: responseBody}, nil
}
