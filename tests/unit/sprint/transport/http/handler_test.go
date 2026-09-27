package transporthttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/transport/http"
)

const sprintProjectID = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"

type sprintRepository struct {
	sprints []domain.Sprint
	err     error
}

func (r *sprintRepository) Create(_ context.Context, sprint domain.Sprint) error {
	r.sprints = append(r.sprints, sprint)
	return r.err
}

func newSprintHandler(repository *sprintRepository) http.Handler {
	useCase := application.NewCreateSprintUseCase(repository, func() string { return "b2062ec4-e75d-4dba-88e2-2d83daf982d2" })
	return transporthttp.NewCreateSprintHandler(useCase)
}

func newSprintRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	parts := strings.Split(path, "/")
	if len(parts) > 2 {
		request.SetPathValue("project_id", parts[2])
	}
	return request
}

func TestCreateSprintHandlerRejectsMethodAndInvalidProjectIDWithoutWriting(t *testing.T) {
	for _, scenario := range []struct {
		name, method, path string
		status             int
		field              string
	}{
		{"method", http.MethodGet, "/projects/" + sprintProjectID + "/sprints", http.StatusMethodNotAllowed, ""},
		{"invalid project UUID", http.MethodPost, "/projects/not-a-uuid/sprints", http.StatusUnprocessableEntity, "project_id"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			repository := &sprintRepository{}
			response := httptest.NewRecorder()
			newSprintHandler(repository).ServeHTTP(response, newSprintRequest(scenario.method, scenario.path, `{"sprint_goal":"Deliver value"}`))
			if response.Code != scenario.status || len(repository.sprints) != 0 {
				t.Fatalf("status=%d writes=%d body=%s", response.Code, len(repository.sprints), response.Body.String())
			}
			if scenario.field != "" && !strings.Contains(response.Body.String(), `"`+scenario.field+`"`) {
				t.Errorf("response does not identify %s: %s", scenario.field, response.Body.String())
			}
		})
	}
}

func TestCreateSprintHandlerRejectsInvalidJSONAndMissingGoalWithoutWriting(t *testing.T) {
	for _, scenario := range []struct {
		name, body string
		status     int
	}{
		{"malformed JSON", `{"sprint_goal":`, http.StatusBadRequest},
		{"multiple JSON values", `{"sprint_goal":"one"}{"sprint_goal":"two"}`, http.StatusBadRequest},
		{"unknown server-owned fields", `{"sprint_goal":"one","id":"client-id","status":"done","stories":[]}`, http.StatusBadRequest},
		{"missing goal", `{}`, http.StatusUnprocessableEntity},
		{"blank goal", `{"sprint_goal":"  "}`, http.StatusUnprocessableEntity},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			repository := &sprintRepository{}
			response := httptest.NewRecorder()
			newSprintHandler(repository).ServeHTTP(response, newSprintRequest(http.MethodPost, "/projects/"+sprintProjectID+"/sprints", scenario.body))
			if response.Code != scenario.status || len(repository.sprints) != 0 {
				t.Fatalf("status=%d writes=%d body=%s", response.Code, len(repository.sprints), response.Body.String())
			}
			if scenario.status == http.StatusUnprocessableEntity && !strings.Contains(response.Body.String(), `"sprint_goal":"is required"`) {
				t.Errorf("response does not identify missing goal: %s", response.Body.String())
			}
		})
	}
}

func TestCreateSprintHandlerReturnsCreatedSprint(t *testing.T) {
	repository := &sprintRepository{}
	response := httptest.NewRecorder()
	newSprintHandler(repository).ServeHTTP(response, newSprintRequest(http.MethodPost, "/projects/"+strings.ToUpper(sprintProjectID)+"/sprints", `{"sprint_goal":"  Ship metrics  "}`))
	var body struct {
		ID        string `json:"id"`
		ProjectID string `json:"project_id"`
		Goal      string `json:"sprint_goal"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusCreated || body.ID != "b2062ec4-e75d-4dba-88e2-2d83daf982d2" || body.ProjectID != sprintProjectID || body.Goal != "  Ship metrics  " {
		t.Fatalf("status=%d response=%+v body=%s", response.Code, body, response.Body.String())
	}
	if len(repository.sprints) != 1 || repository.sprints[0].SprintGoal != body.Goal {
		t.Fatalf("persisted sprints = %+v", repository.sprints)
	}
}

func TestCreateSprintHandlerMapsProjectAndUnexpectedErrors(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"missing project", application.ErrProjectNotFound, http.StatusNotFound, "project_not_found"},
		{"unexpected failure", errors.New("database password leaked"), http.StatusInternalServerError, "internal_error"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			repository := &sprintRepository{err: scenario.err}
			response := httptest.NewRecorder()
			newSprintHandler(repository).ServeHTTP(response, newSprintRequest(http.MethodPost, "/projects/"+sprintProjectID+"/sprints", `{"sprint_goal":"Deliver value"}`))
			if response.Code != scenario.status || !strings.Contains(response.Body.String(), `"error":"`+scenario.code+`"`) {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if scenario.err != application.ErrProjectNotFound && strings.Contains(response.Body.String(), scenario.err.Error()) {
				t.Errorf("unexpected error leaked in response: %s", response.Body.String())
			}
			if len(repository.sprints) != 1 {
				t.Errorf("writes=%d, want attempted insert", len(repository.sprints))
			}
		})
	}
}
