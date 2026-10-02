package transporthttp_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
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
		{"invalid sprint route", `{"story_ids":["` + story + `"]}`, nil, http.StatusUnprocessableEntity, 0},
		{"invalid story", `{"story_ids":["invalid"]}`, nil, http.StatusUnprocessableEntity, 0},
		{"empty", `{"story_ids":[]}`, nil, http.StatusUnprocessableEntity, 0},
		{"repeated", `{"story_ids":["` + story + `","` + story + `"]}`, nil, http.StatusUnprocessableEntity, 0},
		{"closed sprint", `{"story_ids":["` + story + `"]}`, application.ErrSprintClosed, http.StatusConflict, 1},
		{"not found", `{"story_ids":["` + story + `"]}`, application.ErrStoryNotFound, http.StatusNotFound, 1},
		{"sprint not found", `{"story_ids":["` + story + `"]}`, application.ErrSprintNotFound, http.StatusNotFound, 1},
		{"wrong project", `{"story_ids":["` + story + `"]}`, application.ErrProjectMismatch, http.StatusConflict, 1},
		{"already assigned", `{"story_ids":["` + story + `"]}`, application.ErrStoryAlreadyAssigned, http.StatusConflict, 1},
		{"unexpected", `{"story_ids":["` + story + `"]}`, errors.New("secret db detail"), http.StatusInternalServerError, 1},
		{"unknown field", `{"story_ids":["` + story + `"],"extra":true}`, nil, http.StatusBadRequest, 0},
		{"trailing JSON", `{"story_ids":[]} {}`, nil, http.StatusBadRequest, 0},
		{"null", `null`, nil, http.StatusBadRequest, 0},
		{"array", `[]`, nil, http.StatusBadRequest, 0},
		{"malformed", `{`, nil, http.StatusBadRequest, 0},
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
			if tc.name == "invalid sprint route" {
				req.SetPathValue("sprint_id", "not-a-uuid")
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

type con01UUIDAliasAssigner struct {
	calls int
	ids   []string
}

func (a *con01UUIDAliasAssigner) AssignStories(context.Context, string, []string) error {
	a.calls++
	return nil
}

func (a *con01UUIDAliasAssigner) AssignStoriesForProject(_ context.Context, _, _ string, ids []string) error {
	a.calls++
	a.ids = append([]string(nil), ids...)
	return nil
}

func TestCON01UUIDAliasesAreDuplicate(t *testing.T) {
	const lower = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	upper := strings.ToUpper(lower)
	lowID, err := uuid.Parse(lower)
	if err != nil {
		t.Fatal(err)
	}
	upID, err := uuid.Parse(upper)
	if err != nil {
		t.Fatal(err)
	}
	if lower == upper || lowID != upID {
		t.Fatal("fixture must differ in text but identify the same UUID")
	}
	assigner := &con01UUIDAliasAssigner{}
	handler := storyhttp.NewAssignStoriesHandler(application.NewAssignStoriesUseCase(assigner))
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"story_ids":["`+lower+`","`+upper+`"]}`))
	request.SetPathValue("project_id", "a0368d03-0bdb-4d88-9cc2-dcf092166743")
	request.SetPathValue("sprint_id", "b2a6a455-06e2-41ee-b011-7462c71375a0")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	t.Logf("canonical_equal=%t textual_equal=%t status=%d persistence_calls=%d passed_ids=%q body=%s", lowID == upID, lower == upper, response.Code, assigner.calls, assigner.ids, response.Body.String())
	if response.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d want 422 for duplicate canonical story UUID", response.Code)
	}
	if assigner.calls != 0 {
		t.Errorf("persistence_calls=%d want 0 for duplicate canonical story UUID", assigner.calls)
	}
}

func TestAssignStoriesHandlerCanonicalizesWithoutChangingSelectionOrder(t *testing.T) {
	const first = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	const second = "7256b0bb-835c-4166-9600-3d774e919477"
	assigner := &con01UUIDAliasAssigner{}
	handler := storyhttp.NewAssignStoriesHandler(application.NewAssignStoriesUseCase(assigner))
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"story_ids":["`+strings.ToUpper(second)+`","`+strings.ToUpper(first)+`"]}`))
	request.SetPathValue("project_id", "a0368d03-0bdb-4d88-9cc2-dcf092166743")
	request.SetPathValue("sprint_id", "b2a6a455-06e2-41ee-b011-7462c71375a0")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || assigner.calls != 1 || len(assigner.ids) != 2 || assigner.ids[0] != second || assigner.ids[1] != first {
		t.Fatalf("status/calls/ids = %d/%d/%q, want 201/1 and canonical selection in input order", response.Code, assigner.calls, assigner.ids)
	}
	if !strings.Contains(response.Body.String(), `"story_ids":["`+second+`","`+first+`"]`) {
		t.Fatalf("response does not preserve canonical selection: %s", response.Body.String())
	}
}

func TestAssignStoriesHandlerRejectsOtherMethods(t *testing.T) {
	assigner := &assignmentAssigner{}
	rec := httptest.NewRecorder()
	storyhttp.NewAssignStoriesHandler(application.NewAssignStoriesUseCase(assigner)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusMethodNotAllowed || assigner.calls != 0 {
		t.Fatalf("status/calls = %d/%d", rec.Code, assigner.calls)
	}
}
