package transporthttp_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/transport/http"
)

type closeSprintRepository struct {
	projectID string
	sprintID  string
	err       error
	calls     int
}

func (r *closeSprintRepository) CloseSprint(_ context.Context, projectID, sprintID string) error {
	r.calls++
	r.projectID, r.sprintID = projectID, sprintID
	return r.err
}

func newCloseSprintHandler(repository *closeSprintRepository) http.Handler {
	return transporthttp.NewCloseSprintHandler(application.NewCloseSprintUseCase(repository))
}

func newCloseSprintRequest(method, projectID, sprintID, body string) *http.Request {
	request := httptest.NewRequest(method, "/", strings.NewReader(body))
	request.SetPathValue("project_id", projectID)
	request.SetPathValue("sprint_id", sprintID)
	return request
}

func TestCloseSprintHandlerClosesAndReturnsProjectScopedSprint(t *testing.T) {
	repository := &closeSprintRepository{}
	response := httptest.NewRecorder()
	newCloseSprintHandler(repository).ServeHTTP(response, newCloseSprintRequest(http.MethodPost,
		"5c21cbd4-d9a7-42df-9c3a-c0866f058746", "b2062ec4-e75d-4dba-88e2-2d83daf982d2", "ignored body"))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"is_closed":true`) || repository.calls != 1 {
		t.Fatalf("status=%d calls=%d body=%s", response.Code, repository.calls, response.Body.String())
	}
}

func TestCloseSprintHandlerRejectsInvalidUUIDsAndMethodsBeforePersistence(t *testing.T) {
	for _, scenario := range []struct {
		name, method, projectID, sprintID string
		status                            int
	}{
		{"invalid project", http.MethodPost, "bad", "b2062ec4-e75d-4dba-88e2-2d83daf982d2", http.StatusUnprocessableEntity},
		{"invalid sprint", http.MethodPost, "5c21cbd4-d9a7-42df-9c3a-c0866f058746", "bad", http.StatusUnprocessableEntity},
		{"unsupported method", http.MethodGet, "5c21cbd4-d9a7-42df-9c3a-c0866f058746", "b2062ec4-e75d-4dba-88e2-2d83daf982d2", http.StatusMethodNotAllowed},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			repository := &closeSprintRepository{}
			response := httptest.NewRecorder()
			newCloseSprintHandler(repository).ServeHTTP(response, newCloseSprintRequest(scenario.method, scenario.projectID, scenario.sprintID, ""))
			if response.Code != scenario.status || repository.calls != 0 {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, repository.calls, response.Body.String())
			}
		})
	}
}

func TestCloseSprintHandlerMapsNotFoundConflictAndInternalErrors(t *testing.T) {
	for _, scenario := range []struct {
		name, code string
		err        error
		status     int
	}{
		{"project", "project_not_found", application.ErrProjectNotFound, http.StatusNotFound},
		{"sprint", "sprint_not_found", application.ErrSprintNotFound, http.StatusNotFound},
		{"already closed", "sprint_closed", application.ErrSprintClosed, http.StatusConflict},
		{"unexpected", "internal_error", errors.New("private database detail"), http.StatusInternalServerError},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			repository := &closeSprintRepository{err: scenario.err}
			response := httptest.NewRecorder()
			newCloseSprintHandler(repository).ServeHTTP(response, newCloseSprintRequest(http.MethodPost,
				"5c21cbd4-d9a7-42df-9c3a-c0866f058746", "b2062ec4-e75d-4dba-88e2-2d83daf982d2", ""))
			if response.Code != scenario.status || !strings.Contains(response.Body.String(), `"error":"`+scenario.code+`"`) {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if scenario.name == "unexpected" && strings.Contains(response.Body.String(), scenario.err.Error()) {
				t.Errorf("internal error leaked: %s", response.Body.String())
			}
		})
	}
}
