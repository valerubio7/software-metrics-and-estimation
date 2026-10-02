package transporthttp_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	storyhttp "github.com/valerubio7/software-metrics-and-estimation/internal/story/transport/http"
)

type assignmentAssigner struct {
	calls     int
	projectID string
	err       error
}

func (a *assignmentAssigner) AssignStories(_ context.Context, _ string, _ []string) error {
	return a.assign("")
}

func (a *assignmentAssigner) AssignStoriesForProject(_ context.Context, projectID string, _ string, _ []string) error {
	return a.assign(projectID)
}

func (a *assignmentAssigner) assign(projectID string) error {
	a.calls++
	a.projectID = projectID
	return a.err
}

func TestAssignStoriesHandler(t *testing.T) {
	const sprint = "b2a6a455-06e2-41ee-b011-7462c71375a0"
	const story = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	for _, tc := range []struct {
		name  string
		body  string
		err   error
		want  int
		calls int
	}{
		{"success", `{"story_ids":["` + story + `"]}`, nil, http.StatusCreated, 1},
		{"invalid project route", `{"story_ids":["` + story + `"]}`, nil, http.StatusUnprocessableEntity, 0},
		{"empty", `{"story_ids":[]}`, nil, http.StatusUnprocessableEntity, 0},
		{"repeated", `{"story_ids":["` + story + `","` + story + `"]}`, nil, http.StatusUnprocessableEntity, 0},
		{"closed sprint", `{"story_ids":["` + story + `"]}`, application.ErrSprintClosed, http.StatusConflict, 1},
		{"not found", `{"story_ids":["` + story + `"]}`, application.ErrStoryNotFound, http.StatusNotFound, 1},
		{"unexpected", `{"story_ids":["` + story + `"]}`, errors.New("secret db detail"), http.StatusInternalServerError, 1},
		{"unknown field", `{"story_ids":["` + story + `"],"extra":true}`, nil, http.StatusBadRequest, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assigner := &assignmentAssigner{err: tc.err}
			h := storyhttp.NewAssignStoriesHandler(application.NewAssignStoriesUseCase(assigner))
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			req.SetPathValue("project_id", "a0368d03-0bdb-4d88-9cc2-dcf092166743")
			req.SetPathValue("sprint_id", sprint)
			if tc.name == "invalid project route" {
				req.SetPathValue("project_id", "not-a-uuid")
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want || assigner.calls != tc.calls {
				t.Fatalf("status/calls = %d/%d, want %d/%d: %s", rec.Code, assigner.calls, tc.want, tc.calls, rec.Body.String())
			}
			if tc.name == "success" && assigner.projectID != "a0368d03-0bdb-4d88-9cc2-dcf092166743" {
				t.Fatalf("project passed to use case = %q, want route project ID", assigner.projectID)
			}
			if strings.Contains(rec.Body.String(), "secret db detail") {
				t.Fatal("internal error leaked")
			}
		})
	}
}
