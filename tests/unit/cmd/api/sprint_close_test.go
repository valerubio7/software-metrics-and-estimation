package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
)

type fakeSprintCloser struct{ calls int }

func (c *fakeSprintCloser) CloseSprint(context.Context, string, string) error {
	c.calls++
	return nil
}

func TestSprintClosureRouteRequiresCleanVersionEightAndExplicitDependency(t *testing.T) {
	const id = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	for _, scenario := range []struct {
		name       string
		version    int
		dirty      bool
		lookup     error
		dependency bool
		wantRoute  bool
	}{
		{name: "v7", version: 7},
		{name: "v8", version: 8, dependency: true, wantRoute: true},
		{name: "future", version: 12, dependency: true, wantRoute: true},
		{name: "dirty", version: 8, dirty: true},
		{name: "lookup error", version: 8, lookup: errors.New("lookup failed")},
		{name: "missing dependency", version: 8},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			readiness := api.ResolveMigrationReadiness(scenario.version, scenario.dirty, scenario.lookup)
			if readiness.SprintClose != (scenario.version >= 8 && !scenario.dirty && scenario.lookup == nil) {
				t.Fatalf("SprintClose readiness = %t", readiness.SprintClose)
			}
			closer := &fakeSprintCloser{}
			deps := api.HTTPDependencies{}
			if scenario.dependency {
				deps.Sprints = &api.SprintDependencies{Repository: &fakeSprintRepository{}, GenerateID: func() string { return id }, Closer: closer}
			}
			handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return id }, deps)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/projects/"+id+"/sprints/"+id+"/closure", strings.NewReader("")))
			wantStatus, wantCalls := http.StatusNotFound, 0
			if scenario.wantRoute {
				wantStatus, wantCalls = http.StatusOK, 1
			}
			if response.Code != wantStatus || closer.calls != wantCalls {
				t.Fatalf("status/calls=%d/%d want=%d/%d body=%s", response.Code, closer.calls, wantStatus, wantCalls, response.Body.String())
			}
		})
	}
}

func TestMigrationReadinessGatesSprintClosureIndependently(t *testing.T) {
	for _, scenario := range []struct {
		version int
		dirty   bool
		lookup  error
		want    bool
	}{
		{version: 7, want: false},
		{version: 8, want: true},
		{version: 8, dirty: true, want: false},
		{version: 8, lookup: errors.New("unavailable"), want: false},
	} {
		readiness := api.ResolveMigrationReadiness(scenario.version, scenario.dirty, scenario.lookup)
		if readiness.SprintClose != scenario.want {
			t.Errorf("ResolveMigrationReadiness(%d, %t, %v).SprintClose=%t, want %t", scenario.version, scenario.dirty, scenario.lookup, readiness.SprintClose, scenario.want)
		}
	}
}
