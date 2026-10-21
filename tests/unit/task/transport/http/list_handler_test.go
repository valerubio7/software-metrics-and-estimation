package transporthttp_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/task/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/task/domain"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/task/transport/http"
)

type listTasksUseCase struct {
	result  application.ProjectStoryTasks
	err     error
	calls   int
	project string
	story   string
}

func (u *listTasksUseCase) Execute(_ context.Context, query application.ListTasksQuery) (application.ProjectStoryTasks, error) {
	u.calls++
	u.project, u.story = query.ProjectID, query.StoryID
	return u.result, u.err
}

func TestListTasksHandlerRejectsNonGetMethods(t *testing.T) {
	useCase := &listTasksUseCase{}
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.SetPathValue("project_id", validProjectID)
	request.SetPathValue("story_id", validStoryID)
	recorder := httptest.NewRecorder()

	transporthttp.NewListTasksHandler(useCase).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed || useCase.calls != 0 {
		t.Fatalf("status/calls = %d/%d, want 405/0", recorder.Code, useCase.calls)
	}
}

func TestListTasksHandlerResponses(t *testing.T) {
	hours := 4.5
	for _, tc := range []struct {
		name       string
		projectID  string
		storyID    string
		result     application.ProjectStoryTasks
		err        error
		wantStatus int
		wantBody   []string
	}{
		{
			name: "invalid project", projectID: "not-a-uuid",
			wantStatus: http.StatusUnprocessableEntity, wantBody: []string{"validation_failed", "project_id"},
		},
		{
			name: "invalid story", storyID: "not-a-uuid",
			wantStatus: http.StatusUnprocessableEntity, wantBody: []string{"validation_failed", "story_id"},
		},
		{
			name: "project not found", err: application.ErrProjectNotFound,
			wantStatus: http.StatusNotFound, wantBody: []string{"project_not_found"},
		},
		{
			name: "story not found", err: application.ErrStoryNotFound,
			wantStatus: http.StatusNotFound, wantBody: []string{"story_not_found"},
		},
		{
			name: "empty collection",
			result: application.ProjectStoryTasks{ProjectID: validProjectID, StoryID: validStoryID, Tasks: []domain.Task{}},
			wantStatus: http.StatusOK, wantBody: []string{`"project_id":"` + validProjectID + `"`, `"story_id":"` + validStoryID + `"`, `"tasks":[]`},
		},
		{
			name: "tasks with and without estimate",
			result: application.ProjectStoryTasks{ProjectID: validProjectID, StoryID: validStoryID, Tasks: []domain.Task{
				{ID: "t1", ProjectID: validProjectID, SprintID: validSprintID, StoryID: validStoryID, Title: "Diseñar", EstimatedHours: &hours},
				{ID: "t2", ProjectID: validProjectID, SprintID: validSprintID, StoryID: validStoryID, Title: "Probar"},
			}},
			wantStatus: http.StatusOK,
			wantBody: []string{`"title":"Diseñar"`, `"estimated_hours":4.5`, `"title":"Probar"`, `"estimated_hours":null`},
		},
		{
			name: "internal error", err: errors.New("secret db detail"),
			wantStatus: http.StatusInternalServerError, wantBody: []string{"internal_error"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			projectID, storyID := validProjectID, validStoryID
			if tc.projectID != "" {
				projectID = tc.projectID
			}
			if tc.storyID != "" {
				storyID = tc.storyID
			}
			useCase := &listTasksUseCase{result: tc.result, err: tc.err}
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.SetPathValue("project_id", projectID)
			request.SetPathValue("story_id", storyID)
			recorder := httptest.NewRecorder()

			transporthttp.NewListTasksHandler(useCase).ServeHTTP(recorder, request)

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, body %s; want %d", recorder.Code, recorder.Body.String(), tc.wantStatus)
			}
			for _, want := range tc.wantBody {
				if !strings.Contains(recorder.Body.String(), want) {
					t.Errorf("body %s does not contain %q", recorder.Body.String(), want)
				}
			}
			if tc.err != nil && strings.Contains(recorder.Body.String(), "secret") {
				t.Fatal("internal details leaked")
			}
			if tc.wantStatus == http.StatusUnprocessableEntity && useCase.calls != 0 {
				t.Fatal("use case called for invalid UUID")
			}
		})
	}
}
