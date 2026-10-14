package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	sprintapplication "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
	sprintdomain "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
)

const historyProjectID = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"

// fakeSprintHistoryReader is a read-only port double that counts reads.
type fakeSprintHistoryReader struct {
	reads   int
	sprints []sprintdomain.ClosedSprint
	err     error
}

func (r *fakeSprintHistoryReader) ListClosedByProject(_ context.Context, _ string) ([]sprintdomain.ClosedSprint, error) {
	r.reads++
	return r.sprints, r.err
}

// composeSprintDependencies mirrors the gating in cmd/api/main.go for a given schema state.
func composeSprintDependencies(version int, dirty bool, lookup error, closer *fakeSprintCloser, lister *fakeSprintHistoryReader) api.HTTPDependencies {
	readiness := api.ResolveMigrationReadiness(version, dirty, lookup)
	if !readiness.Sprints {
		return api.HTTPDependencies{}
	}
	dependencies := &api.SprintDependencies{Repository: &fakeSprintRepository{}, GenerateID: func() string { return historyProjectID }}
	if readiness.SprintClose {
		dependencies.Closer = closer
	}
	if readiness.SprintHistory {
		dependencies.Lister = lister
	}
	return api.HTTPDependencies{Sprints: dependencies}
}

func serve(handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(method, target, strings.NewReader(body)))
	return response
}

func TestMigrationReadinessGatesSprintHistoryFromVersionEleven(t *testing.T) {
	for _, scenario := range []struct {
		name         string
		version      int
		dirty        bool
		lookup       error
		wantHistory  bool
		wantClose    bool
		wantCreation bool
	}{
		{name: "v7", version: 7, wantCreation: true},
		{name: "v8", version: 8, wantClose: true, wantCreation: true},
		{name: "v10", version: 10, wantClose: true, wantCreation: true},
		{name: "v11", version: 11, wantHistory: true, wantClose: true, wantCreation: true},
		{name: "v12", version: 12, wantHistory: true, wantClose: true, wantCreation: true},
		{name: "dirty v11", version: 11, dirty: true},
		{name: "lookup error", version: 11, lookup: errors.New("lookup failed")},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			readiness := api.ResolveMigrationReadiness(scenario.version, scenario.dirty, scenario.lookup)
			if readiness.SprintHistory != scenario.wantHistory {
				t.Errorf("SprintHistory = %t, want %t", readiness.SprintHistory, scenario.wantHistory)
			}
			if readiness.SprintClose != scenario.wantClose {
				t.Errorf("SprintClose = %t, want %t", readiness.SprintClose, scenario.wantClose)
			}
			if readiness.Sprints != scenario.wantCreation {
				t.Errorf("Sprints = %t, want %t", readiness.Sprints, scenario.wantCreation)
			}
		})
	}
}

func TestSprintHistoryCollectionRouteWithLister(t *testing.T) {
	const collection = "/projects/" + historyProjectID + "/sprints"
	closed := []sprintdomain.ClosedSprint{{ID: "a", ProjectID: historyProjectID, SprintGoal: "Goal", Stories: []sprintdomain.PlannedStory{{ID: "s", Title: "Story", Priority: "alta"}}}}
	for _, scenario := range []struct {
		name      string
		method    string
		target    string
		reader    *fakeSprintHistoryReader
		wantCode  int
		wantReads int
		wantBody  string
	}{
		{name: "GET", method: http.MethodGet, target: collection, reader: &fakeSprintHistoryReader{sprints: closed}, wantCode: http.StatusOK, wantReads: 1, wantBody: `"sprint_goal":"Goal"`},
		{name: "GET missing project", method: http.MethodGet, target: collection, reader: &fakeSprintHistoryReader{err: sprintapplication.ErrProjectNotFound}, wantCode: http.StatusNotFound, wantReads: 1, wantBody: `"error":"project_not_found"`},
		{name: "GET invalid project", method: http.MethodGet, target: "/projects/not-a-uuid/sprints", reader: &fakeSprintHistoryReader{}, wantCode: http.StatusUnprocessableEntity, wantReads: 0, wantBody: `"project_id"`},
		{name: "HEAD", method: http.MethodHead, target: collection, reader: &fakeSprintHistoryReader{sprints: closed}, wantCode: http.StatusOK, wantReads: 1},
		{name: "POST keeps creating", method: http.MethodPost, target: collection, reader: &fakeSprintHistoryReader{}, wantCode: http.StatusCreated, wantReads: 0, wantBody: `"sprint_goal":"Deliver"`},
		{name: "PUT", method: http.MethodPut, target: collection, reader: &fakeSprintHistoryReader{}, wantCode: http.StatusMethodNotAllowed, wantReads: 0},
		{name: "DELETE", method: http.MethodDelete, target: collection, reader: &fakeSprintHistoryReader{}, wantCode: http.StatusMethodNotAllowed, wantReads: 0},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return historyProjectID }, api.HTTPDependencies{
				Sprints: &api.SprintDependencies{Repository: &fakeSprintRepository{}, GenerateID: func() string { return historyProjectID }, Lister: scenario.reader},
			})

			response := serve(handler, scenario.method, scenario.target, `{"sprint_goal":"Deliver"}`)

			if response.Code != scenario.wantCode || scenario.reader.reads != scenario.wantReads {
				t.Fatalf("status/reads = %d/%d, want %d/%d body=%s", response.Code, scenario.reader.reads, scenario.wantCode, scenario.wantReads, response.Body.String())
			}
			if scenario.wantBody != "" && !strings.Contains(response.Body.String(), scenario.wantBody) {
				t.Errorf("body = %s, want it to contain %s", response.Body.String(), scenario.wantBody)
			}
		})
	}
}

func TestSprintHistoryRouteIsNotRegisteredWithoutLister(t *testing.T) {
	const collection = "/projects/" + historyProjectID + "/sprints"
	withCreation := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return historyProjectID }, api.HTTPDependencies{
		Sprints: &api.SprintDependencies{Repository: &fakeSprintRepository{}, GenerateID: func() string { return historyProjectID }},
	})
	if got := serve(withCreation, http.MethodGet, collection, "").Code; got != http.StatusMethodNotAllowed {
		t.Errorf("GET with creation and without Lister = %d, want 405", got)
	}
	if got := serve(withCreation, http.MethodPost, collection, `{"sprint_goal":"Deliver"}`).Code; got != http.StatusCreated {
		t.Errorf("POST with creation and without Lister = %d, want 201", got)
	}

	withoutSprints := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return historyProjectID }, api.HTTPDependencies{})
	if got := serve(withoutSprints, http.MethodGet, collection, "").Code; got != http.StatusNotFound {
		t.Errorf("GET without Sprint dependencies = %d, want 404", got)
	}
}

func TestSprintHistoryAvailabilityFollowsSchemaState(t *testing.T) {
	const collection = "/projects/" + historyProjectID + "/sprints"
	for _, scenario := range []struct {
		name     string
		version  int
		dirty    bool
		lookup   error
		wantCode int
		wantRead int
	}{
		{name: "v4 has no sprint routes", version: 4, wantCode: http.StatusNotFound},
		{name: "v5 creation only", version: 5, wantCode: http.StatusMethodNotAllowed},
		{name: "v7 creation only", version: 7, wantCode: http.StatusMethodNotAllowed},
		{name: "v8 creation and closure", version: 8, wantCode: http.StatusMethodNotAllowed},
		{name: "v10 creation and closure", version: 10, wantCode: http.StatusMethodNotAllowed},
		{name: "v11 history", version: 11, wantCode: http.StatusOK, wantRead: 1},
		{name: "v12 history", version: 12, wantCode: http.StatusOK, wantRead: 1},
		{name: "dirty v11 has no sprint routes", version: 11, dirty: true, wantCode: http.StatusNotFound},
		{name: "lookup error has no sprint routes", version: 11, lookup: errors.New("lookup failed"), wantCode: http.StatusNotFound},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			lister := &fakeSprintHistoryReader{}
			handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return historyProjectID },
				composeSprintDependencies(scenario.version, scenario.dirty, scenario.lookup, &fakeSprintCloser{}, lister))

			response := serve(handler, http.MethodGet, collection, "")

			if response.Code != scenario.wantCode || lister.reads != scenario.wantRead {
				t.Fatalf("status/reads = %d/%d, want %d/%d body=%s", response.Code, lister.reads, scenario.wantCode, scenario.wantRead, response.Body.String())
			}
		})
	}
}

func TestSprintHistoryGateIsIndependentFromTheClosureGate(t *testing.T) {
	const closure = "/projects/" + historyProjectID + "/sprints/" + historyProjectID + "/closure"
	const collection = "/projects/" + historyProjectID + "/sprints"
	for _, scenario := range []struct {
		name         string
		version      int
		wantClosure  int
		wantHistory  int
		wantCloseUse int
	}{
		{name: "v8 closes and does not list", version: 8, wantClosure: http.StatusOK, wantHistory: http.StatusMethodNotAllowed, wantCloseUse: 1},
		{name: "v10 closes and does not list", version: 10, wantClosure: http.StatusOK, wantHistory: http.StatusMethodNotAllowed, wantCloseUse: 1},
		{name: "v11 closes and lists", version: 11, wantClosure: http.StatusOK, wantHistory: http.StatusOK, wantCloseUse: 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			closer, lister := &fakeSprintCloser{}, &fakeSprintHistoryReader{}
			handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return historyProjectID },
				composeSprintDependencies(scenario.version, false, nil, closer, lister))

			closed := serve(handler, http.MethodPost, closure, "")
			listed := serve(handler, http.MethodGet, collection, "")

			if closed.Code != scenario.wantClosure || closer.calls != scenario.wantCloseUse {
				t.Errorf("closure status/calls = %d/%d, want %d/%d", closed.Code, closer.calls, scenario.wantClosure, scenario.wantCloseUse)
			}
			if listed.Code != scenario.wantHistory {
				t.Errorf("history status = %d, want %d", listed.Code, scenario.wantHistory)
			}
		})
	}
}
