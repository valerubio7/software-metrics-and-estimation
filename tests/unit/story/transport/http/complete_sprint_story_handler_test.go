package transporthttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	storyhttp "github.com/valerubio7/software-metrics-and-estimation/internal/story/transport/http"
)

type completionFake struct {
	calls                        int
	projectID, sprintID, storyID string
	err                          error
}

// completedInstant carries a non-UTC zone so the test proves the handler normalizes to UTC.
var completedInstant = time.Date(2026, 10, 4, 11, 3, 21, 123456000, time.FixedZone("UTC-3", -3*60*60))

func (f *completionFake) CompleteSprintStory(_ context.Context, projectID, sprintID, storyID string) (application.SprintStoryCompletion, error) {
	f.calls++
	f.projectID, f.sprintID, f.storyID = projectID, sprintID, storyID
	if f.err != nil {
		return application.SprintStoryCompletion{}, f.err
	}
	return application.SprintStoryCompletion{ProjectID: projectID, SprintID: sprintID, StoryID: storyID, CompletedAt: completedInstant}, nil
}

func TestCompleteSprintStoryHandler(t *testing.T) {
	const (
		project = "82d38423-f02d-4259-9e35-4a291585bb1e"
		sprint  = "b2a6a455-06e2-41ee-b011-7462c71375a0"
		story   = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	)
	tests := []struct {
		name                         string
		projectID, sprintID, storyID string
		body                         string
		completerErr                 error
		wantStatus                   int
		wantError                    string
		wantFields                   []string
		wantCalls                    int
	}{
		{name: "success", projectID: project, sprintID: sprint, storyID: story, wantStatus: http.StatusOK, wantCalls: 1},
		{name: "uppercase UUIDs are canonicalized", projectID: strings.ToUpper(project), sprintID: strings.ToUpper(sprint), storyID: strings.ToUpper(story), wantStatus: http.StatusOK, wantCalls: 1},
		{name: "json body is ignored", projectID: project, sprintID: sprint, storyID: story, body: `{"x":1}`, wantStatus: http.StatusOK, wantCalls: 1},
		{name: "invalid body is ignored", projectID: project, sprintID: sprint, storyID: story, body: `not json`, wantStatus: http.StatusOK, wantCalls: 1},
		{name: "invalid project_id", projectID: "bad", sprintID: sprint, storyID: story, wantStatus: http.StatusUnprocessableEntity, wantError: "validation_failed", wantFields: []string{"project_id"}},
		{name: "invalid sprint_id", projectID: project, sprintID: "bad", storyID: story, wantStatus: http.StatusUnprocessableEntity, wantError: "validation_failed", wantFields: []string{"sprint_id"}},
		{name: "invalid story_id", projectID: project, sprintID: sprint, storyID: "bad", wantStatus: http.StatusUnprocessableEntity, wantError: "validation_failed", wantFields: []string{"story_id"}},
		{name: "all route IDs invalid", projectID: "x", sprintID: "y", storyID: "z", wantStatus: http.StatusUnprocessableEntity, wantError: "validation_failed", wantFields: []string{"project_id", "sprint_id", "story_id"}},
		{name: "project not found", projectID: project, sprintID: sprint, storyID: story, completerErr: application.ErrProjectNotFound, wantStatus: http.StatusNotFound, wantError: "project_not_found", wantCalls: 1},
		{name: "sprint not found", projectID: project, sprintID: sprint, storyID: story, completerErr: application.ErrSprintNotFound, wantStatus: http.StatusNotFound, wantError: "sprint_not_found", wantCalls: 1},
		{name: "story not found", projectID: project, sprintID: sprint, storyID: story, completerErr: application.ErrStoryNotFound, wantStatus: http.StatusNotFound, wantError: "story_not_found", wantCalls: 1},
		{name: "sprint closed", projectID: project, sprintID: sprint, storyID: story, completerErr: application.ErrSprintClosed, wantStatus: http.StatusConflict, wantError: "sprint_closed", wantCalls: 1},
		{name: "story not in sprint", projectID: project, sprintID: sprint, storyID: story, completerErr: application.ErrStoryNotInSprint, wantStatus: http.StatusConflict, wantError: "story_not_in_sprint", wantCalls: 1},
		{name: "story already completed", projectID: project, sprintID: sprint, storyID: story, completerErr: application.ErrStoryAlreadyCompleted, wantStatus: http.StatusConflict, wantError: "story_already_completed", wantCalls: 1},
		{name: "unexpected", projectID: project, sprintID: sprint, storyID: story, completerErr: errors.New("secret db detail"), wantStatus: http.StatusInternalServerError, wantError: "internal_error", wantCalls: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			completer := &completionFake{err: tc.completerErr}
			mux := http.NewServeMux()
			mux.Handle("POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/completion", storyhttp.NewCompleteSprintStoryHandler(application.NewCompleteSprintStoryUseCase(completer)))
			request := httptest.NewRequest(http.MethodPost, "/projects/"+tc.projectID+"/sprints/"+tc.sprintID+"/stories/"+tc.storyID+"/completion", strings.NewReader(tc.body))
			response := httptest.NewRecorder()

			mux.ServeHTTP(response, request)

			if response.Code != tc.wantStatus || completer.calls != tc.wantCalls {
				t.Fatalf("status/calls = %d/%d, want %d/%d; body %s", response.Code, completer.calls, tc.wantStatus, tc.wantCalls, response.Body)
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("response is not JSON: %v (%s)", err, response.Body)
			}
			if tc.wantStatus == http.StatusOK {
				if response.Header().Get("Content-Type") != "application/json" {
					t.Fatalf("Content-Type = %q", response.Header().Get("Content-Type"))
				}
				if completer.projectID != project || completer.sprintID != sprint || completer.storyID != story {
					t.Fatalf("port IDs = %q, %q, %q; want canonical lowercase IDs", completer.projectID, completer.sprintID, completer.storyID)
				}
				if body["project_id"] != project || body["sprint_id"] != sprint || body["story_id"] != story || body["completed_at"] != "2026-10-04T14:03:21.123456Z" {
					t.Fatalf("success body = %v, want IDs and completed_at in UTC", body)
				}
				return
			}
			if body["error"] != tc.wantError {
				t.Fatalf("error code = %v, want %s", body["error"], tc.wantError)
			}
			fields, _ := body["fields"].(map[string]any)
			if len(fields) != len(tc.wantFields) {
				t.Fatalf("fields = %v, want keys %v", fields, tc.wantFields)
			}
			for _, key := range tc.wantFields {
				if fields[key] != "must be a valid UUID" {
					t.Fatalf("fields[%s] = %v, want %q", key, fields[key], "must be a valid UUID")
				}
			}
			if strings.Contains(response.Body.String(), "secret") {
				t.Fatalf("response leaks internal detail: %s", response.Body)
			}
			if tc.wantStatus == http.StatusUnprocessableEntity && response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("error Content-Type = %q", response.Header().Get("Content-Type"))
			}
		})
	}

	t.Run("wrong method", func(t *testing.T) {
		completer := &completionFake{}
		handler := storyhttp.NewCompleteSprintStoryHandler(application.NewCompleteSprintStoryUseCase(completer))
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/completion", nil))

		if response.Code != http.StatusMethodNotAllowed || completer.calls != 0 || !strings.Contains(response.Body.String(), "method_not_allowed") {
			t.Fatalf("status/calls/body = %d/%d/%s, want 405/0/method_not_allowed", response.Code, completer.calls, response.Body)
		}
	})
}
