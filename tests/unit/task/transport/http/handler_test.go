package transporthttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/task/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/task/domain"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/task/transport/http"
)

const (
	validProjectID = "82d38423-f02d-4259-9e35-4a291585bb1e"
	validSprintID  = "b2a6a455-06e2-41ee-b011-7462c71375a0"
	validStoryID   = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
)

// fakeTaskRepository stands in for application.TaskRepository, injected into a real use case.
type fakeTaskRepository struct {
	calls int
	err   error
}

func (r *fakeTaskRepository) CreateForSprintStory(_ context.Context, _, _, _ string, _ []domain.Task) error {
	r.calls++
	return r.err
}

func newHandler(repo *fakeTaskRepository) http.Handler {
	useCase := application.NewCreateTasksUseCase(repo, func() string { return "generated-id" })
	return transporthttp.NewCreateTasksHandler(useCase)
}

func TestCreateTasksHandler(t *testing.T) {
	const estimatedHoursMessage = "must be greater than 0, at most 99999.99 and have at most 2 decimals"

	for _, tc := range []struct {
		name       string
		body       string
		projectID  string
		sprintID   string
		storyID    string
		repoErr    error
		wantStatus int
		wantError  string
		wantFields map[string]string
		wantCalls  int
	}{
		{
			name:       "success",
			body:       `{"tasks":[{"title":"Diseñar el esquema","estimated_hours":4.5},{"title":"Escribir pruebas"}]}`,
			wantStatus: http.StatusCreated, wantCalls: 1,
		},
		{
			name:       "success without estimate key",
			body:       `{"tasks":[{"title":"Solo titulo"}]}`,
			wantStatus: http.StatusCreated, wantCalls: 1,
		},
		{
			name: "invalid project route", body: `{"tasks":[{"title":"A"}]}`, projectID: "not-a-uuid",
			wantStatus: http.StatusUnprocessableEntity, wantError: "validation_failed",
			wantFields: map[string]string{"project_id": "must be a valid UUID"},
		},
		{
			name: "invalid sprint route", body: `{"tasks":[{"title":"A"}]}`, sprintID: "not-a-uuid",
			wantStatus: http.StatusUnprocessableEntity, wantError: "validation_failed",
			wantFields: map[string]string{"sprint_id": "must be a valid UUID"},
		},
		{
			name: "invalid story route", body: `{"tasks":[{"title":"A"}]}`, storyID: "not-a-uuid",
			wantStatus: http.StatusUnprocessableEntity, wantError: "validation_failed",
			wantFields: map[string]string{"story_id": "must be a valid UUID"},
		},
		{name: "empty tasks", body: `{"tasks":[]}`, wantStatus: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "missing tasks", body: `{}`, wantStatus: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "null tasks", body: `{"tasks":null}`, wantStatus: http.StatusBadRequest, wantError: "invalid_request"},
		{
			name: "blank title", body: `{"tasks":[{"title":"   "}]}`,
			wantStatus: http.StatusUnprocessableEntity, wantError: "validation_failed",
			wantFields: map[string]string{"tasks[0].title": "is required"},
		},
		{
			name: "invalid estimate", body: `{"tasks":[{"title":"A","estimated_hours":-1}]}`,
			wantStatus: http.StatusUnprocessableEntity, wantError: "validation_failed",
			wantFields: map[string]string{"tasks[0].estimated_hours": estimatedHoursMessage},
		},
		{
			name:       "multiple invalid elements",
			body:       `{"tasks":[{"title":""},{"title":"Valida"},{"title":"Valida","estimated_hours":100000}]}`,
			wantStatus: http.StatusUnprocessableEntity, wantError: "validation_failed",
			wantFields: map[string]string{"tasks[0].title": "is required", "tasks[2].estimated_hours": estimatedHoursMessage},
		},
		{name: "unknown root field", body: `{"tasks":[{"title":"A"}],"extra":true}`, wantStatus: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "unknown task field", body: `{"tasks":[{"title":"A","status":"done"}]}`, wantStatus: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "trailing JSON", body: `{"tasks":[{"title":"A"}]} {}`, wantStatus: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "null body", body: `null`, wantStatus: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "array body", body: `[]`, wantStatus: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "malformed", body: `{`, wantStatus: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "null task element", body: `{"tasks":[null]}`, wantStatus: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "estimate as string", body: `{"tasks":[{"title":"A","estimated_hours":"8"}]}`, wantStatus: http.StatusBadRequest, wantError: "invalid_request"},
		{
			name: "project not found", body: `{"tasks":[{"title":"A"}]}`, repoErr: application.ErrProjectNotFound,
			wantStatus: http.StatusNotFound, wantError: "project_not_found", wantCalls: 1,
		},
		{
			name: "sprint not found", body: `{"tasks":[{"title":"A"}]}`, repoErr: application.ErrSprintNotFound,
			wantStatus: http.StatusNotFound, wantError: "sprint_not_found", wantCalls: 1,
		},
		{
			name: "story not found", body: `{"tasks":[{"title":"A"}]}`, repoErr: application.ErrStoryNotFound,
			wantStatus: http.StatusNotFound, wantError: "story_not_found", wantCalls: 1,
		},
		{
			name: "story not in sprint", body: `{"tasks":[{"title":"A"}]}`, repoErr: application.ErrStoryNotInSprint,
			wantStatus: http.StatusConflict, wantError: "story_not_in_sprint", wantCalls: 1,
		},
		{
			name: "unexpected", body: `{"tasks":[{"title":"A"}]}`, repoErr: errors.New("secret db detail"),
			wantStatus: http.StatusInternalServerError, wantError: "internal_error", wantCalls: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeTaskRepository{err: tc.repoErr}
			handler := newHandler(repo)
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			projectID, sprintID, storyID := validProjectID, validSprintID, validStoryID
			if tc.projectID != "" {
				projectID = tc.projectID
			}
			if tc.sprintID != "" {
				sprintID = tc.sprintID
			}
			if tc.storyID != "" {
				storyID = tc.storyID
			}
			request.SetPathValue("project_id", projectID)
			request.SetPathValue("sprint_id", sprintID)
			request.SetPathValue("story_id", storyID)

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != tc.wantStatus || repo.calls != tc.wantCalls {
				t.Fatalf("status/calls = %d/%d, want %d/%d: %s", response.Code, repo.calls, tc.wantStatus, tc.wantCalls, response.Body.String())
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", response.Header().Get("Content-Type"))
			}
			if tc.wantError != "" && !strings.Contains(response.Body.String(), `"error":"`+tc.wantError+`"`) {
				t.Errorf("body = %s, want error %q", response.Body.String(), tc.wantError)
			}
			for field, message := range tc.wantFields {
				var body struct {
					Fields map[string]string `json:"fields"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatalf("decode response body: %v", err)
				}
				if body.Fields[field] != message {
					t.Errorf("fields[%q] = %q, want %q (body=%s)", field, body.Fields[field], message, response.Body.String())
				}
			}
			if strings.Contains(response.Body.String(), "secret db detail") {
				t.Error("internal error detail leaked in response body")
			}
		})
	}

	t.Run("wrong method", func(t *testing.T) {
		repo := &fakeTaskRepository{}
		handler := newHandler(repo)
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.SetPathValue("project_id", validProjectID)
		request.SetPathValue("sprint_id", validSprintID)
		request.SetPathValue("story_id", validStoryID)

		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusMethodNotAllowed || repo.calls != 0 {
			t.Fatalf("status/calls = %d/%d, want %d/0", response.Code, repo.calls, http.StatusMethodNotAllowed)
		}
	})

	t.Run("success response preserves order and exposes route IDs and estimates", func(t *testing.T) {
		repo := &fakeTaskRepository{}
		handler := newHandler(repo)
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(
			`{"tasks":[{"title":"Primera","estimated_hours":4.5},{"title":"Segunda","estimated_hours":null}]}`,
		))
		request.SetPathValue("project_id", validProjectID)
		request.SetPathValue("sprint_id", validSprintID)
		request.SetPathValue("story_id", validStoryID)

		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		var body struct {
			Tasks []struct {
				ID             string   `json:"id"`
				ProjectID      string   `json:"project_id"`
				SprintID       string   `json:"sprint_id"`
				StoryID        string   `json:"story_id"`
				Title          string   `json:"title"`
				EstimatedHours *float64 `json:"estimated_hours"`
			} `json:"tasks"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response.Code != http.StatusCreated || len(body.Tasks) != 2 {
			t.Fatalf("status/len = %d/%d, body=%s", response.Code, len(body.Tasks), response.Body.String())
		}
		first, second := body.Tasks[0], body.Tasks[1]
		if first.Title != "Primera" || first.EstimatedHours == nil || *first.EstimatedHours != 4.5 {
			t.Errorf("first task = %#v, want Primera with estimate 4.5", first)
		}
		if second.Title != "Segunda" || second.EstimatedHours != nil {
			t.Errorf("second task = %#v, want Segunda with nil estimate", second)
		}
		if first.ProjectID != validProjectID || first.SprintID != validSprintID || first.StoryID != validStoryID {
			t.Errorf("route IDs = (%s,%s,%s), want (%s,%s,%s)", first.ProjectID, first.SprintID, first.StoryID, validProjectID, validSprintID, validStoryID)
		}
	})
}
