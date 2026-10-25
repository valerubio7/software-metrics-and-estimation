package api_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	effortdomain "github.com/valerubio7/software-metrics-and-estimation/internal/effort/domain"
)

const (
	effortRouteProjectID = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	effortRouteStoryID   = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	effortRouteMemberID  = "7256b0bb-835c-4166-9600-3d774e919477"
	effortRouteRecordID  = "1d6b5c58-1f87-4a5e-bb0f-0b2f8e4a7c21"

	effortRouteTarget = "/projects/" + effortRouteProjectID + "/stories/" + effortRouteStoryID + "/effort-records"
	effortRouteBody   = `{"member_id":"` + effortRouteMemberID + `","work_date":"2026-03-10","description":"Revision","hours":2.5}`
)

// fakeEffortRepository counts writes and captures the last record received.
type fakeEffortRepository struct {
	calls  int
	record effortdomain.EffortRecord
	err    error
}

func (r *fakeEffortRepository) Create(_ context.Context, record effortdomain.EffortRecord) error {
	r.calls++
	r.record = record
	return r.err
}

func fixedEffortClock() time.Time { return time.Date(2026, time.March, 10, 15, 0, 0, 0, time.UTC) }

func effortDependencies(repository *fakeEffortRepository, now func() time.Time) api.HTTPDependencies {
	return api.HTTPDependencies{Effort: &api.EffortDependencies{
		Repository: repository,
		GenerateID: func() string { return effortRouteRecordID },
		Now:        now,
	}}
}

func TestMigrationReadinessResolvesEffortByVersion(t *testing.T) {
	for _, tc := range []struct {
		name    string
		version int
		dirty   bool
		lookup  error
		want    bool
	}{
		{name: "v11", version: 11},
		{name: "v12", version: 12, want: true},
		{name: "v13", version: 13, want: true},
		{name: "dirty v12", version: 12, dirty: true},
		{name: "lookup error", version: 12, lookup: errors.New("query failed")},
		{name: "missing migration table", lookup: errors.New("missing schema_migrations")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			readiness := api.ResolveMigrationReadiness(tc.version, tc.dirty, tc.lookup)
			if readiness.Effort != tc.want {
				t.Fatalf("Effort = %t, want %t (readiness = %+v)", readiness.Effort, tc.want, readiness)
			}
			// The effort flag never alters the flags of the existing modules.
			clean := tc.lookup == nil && !tc.dirty
			if readiness.Tasks != (clean && tc.version >= 9) || readiness.SprintHistory != (clean && tc.version >= 11) || !readiness.Projects {
				t.Fatalf("existing readiness flags changed: %+v", readiness)
			}
		})
	}
}

func TestEffortRouteRequiresExplicitDependency(t *testing.T) {
	t.Run("nil dependency answers 404 without writes", func(t *testing.T) {
		repository := &fakeEffortRepository{}
		handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return effortRouteProjectID }, api.HTTPDependencies{})

		response := serve(handler, http.MethodPost, effortRouteTarget, effortRouteBody)
		if response.Code != http.StatusNotFound || repository.calls != 0 {
			t.Fatalf("status/writes = %d/%d, want 404/0: %s", response.Code, repository.calls, response.Body.String())
		}
	})

	t.Run("present dependency answers 201 with one write", func(t *testing.T) {
		repository := &fakeEffortRepository{}
		handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return effortRouteProjectID }, effortDependencies(repository, fixedEffortClock))

		response := serve(handler, http.MethodPost, effortRouteTarget, effortRouteBody)
		if response.Code != http.StatusCreated || repository.calls != 1 {
			t.Fatalf("status/writes = %d/%d, want 201/1: %s", response.Code, repository.calls, response.Body.String())
		}
		if repository.record.ID != effortRouteRecordID || repository.record.ProjectID != effortRouteProjectID || repository.record.StoryID != effortRouteStoryID {
			t.Errorf("stored record = %+v, want generated ID and route IDs", repository.record)
		}
		if !strings.Contains(response.Body.String(), `"id":"`+effortRouteRecordID+`"`) {
			t.Errorf("body = %s, want the generated id", response.Body.String())
		}
	})

	t.Run("GET on the route answers 405", func(t *testing.T) {
		repository := &fakeEffortRepository{}
		handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return effortRouteProjectID }, effortDependencies(repository, fixedEffortClock))

		response := serve(handler, http.MethodGet, effortRouteTarget, "")
		if response.Code != http.StatusMethodNotAllowed || repository.calls != 0 {
			t.Fatalf("GET status/writes = %d/%d, want 405/0", response.Code, repository.calls)
		}
	})
}

func TestEffortRouteUsesInjectedClock(t *testing.T) {
	repository := &fakeEffortRepository{}
	handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return effortRouteProjectID }, effortDependencies(repository, fixedEffortClock))

	// The injected clock reads 2026-03-10: that day is accepted and the next one is a future date.
	accepted := serve(handler, http.MethodPost, effortRouteTarget, effortRouteBody)
	future := serve(handler, http.MethodPost, effortRouteTarget, strings.Replace(effortRouteBody, "2026-03-10", "2026-03-11", 1))
	if accepted.Code != http.StatusCreated || future.Code != http.StatusUnprocessableEntity || repository.calls != 1 {
		t.Fatalf("injected clock: accepted=%d future=%d writes=%d, want 201/422/1", accepted.Code, future.Code, repository.calls)
	}
	if !strings.Contains(future.Body.String(), `"work_date":"must not be in the future"`) {
		t.Errorf("future body = %s, want work_date in fields", future.Body.String())
	}
}

func TestEffortRouteDefaultsToCurrentUTCClockWhenNowIsNil(t *testing.T) {
	repository := &fakeEffortRepository{}
	handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return effortRouteProjectID }, effortDependencies(repository, nil))

	past := serve(handler, http.MethodPost, effortRouteTarget, strings.Replace(effortRouteBody, "2026-03-10", "2000-01-01", 1))
	far := serve(handler, http.MethodPost, effortRouteTarget, strings.Replace(effortRouteBody, "2026-03-10", "2999-12-31", 1))
	if past.Code != http.StatusCreated || far.Code != http.StatusUnprocessableEntity || repository.calls != 1 {
		t.Fatalf("default clock: past=%d far future=%d writes=%d, want 201/422/1", past.Code, far.Code, repository.calls)
	}
}

func TestEffortRouteIsIndependentFromOtherModules(t *testing.T) {
	repository := &fakeEffortRepository{}
	handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return effortRouteProjectID }, effortDependencies(repository, fixedEffortClock))

	effort := serve(handler, http.MethodPost, effortRouteTarget, effortRouteBody)
	if effort.Code != http.StatusCreated || repository.calls != 1 {
		t.Fatalf("effort route status/writes = %d/%d, want 201/1 without Stories, Members or Tasks: %s", effort.Code, repository.calls, effort.Body.String())
	}
	story := serve(handler, http.MethodPost, "/projects/"+effortRouteProjectID+"/stories", `{"title":"T","description":"D","priority":"media","acceptance_criteria":["ok"]}`)
	if story.Code != http.StatusNotFound {
		t.Fatalf("story route status = %d, want 404 (Stories dependency absent)", story.Code)
	}
}

func TestEffortRouteCoexistsWithExistingStoryAndTaskRoutes(t *testing.T) {
	effort := &fakeEffortRepository{}
	points := &storyPointsRepo{}
	tasks := &listableTaskRepository{fakeTaskRepository: &fakeTaskRepository{}}
	dependencies := effortDependencies(effort, fixedEffortClock)
	dependencies.Stories = &api.StoryDependencies{
		Repository:        &fakeStoryRepository{},
		GenerateID:        func() string { return effortRouteStoryID },
		Updater:           storyUpdaterStub{},
		StoryPointsSetter: points,
	}
	dependencies.Tasks = &api.TaskDependencies{Repository: tasks, GenerateID: func() string { return effortRouteRecordID }, Lister: tasks}
	handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return effortRouteProjectID }, dependencies)

	story := "/projects/" + effortRouteProjectID + "/stories/" + effortRouteStoryID
	if got := serve(handler, http.MethodPut, story+"/story-points", `{"story_points":13}`); got.Code != http.StatusOK || points.calls != 1 {
		t.Errorf("PUT story-points = %d (writes %d), want 200/1: %s", got.Code, points.calls, got.Body.String())
	}
	if got := serve(handler, http.MethodPut, story, `{}`); got.Code != http.StatusUnprocessableEntity {
		t.Errorf("PUT story = %d, want 422 from the generic update handler", got.Code)
	}
	if got := serve(handler, http.MethodGet, story+"/tasks", ""); got.Code != http.StatusOK || tasks.listCalls != 1 {
		t.Errorf("GET tasks = %d (reads %d), want 200/1: %s", got.Code, tasks.listCalls, got.Body.String())
	}
	if got := serve(handler, http.MethodPost, story+"/effort-records", effortRouteBody); got.Code != http.StatusCreated || effort.calls != 1 {
		t.Errorf("POST effort-records = %d (writes %d), want 201/1: %s", got.Code, effort.calls, got.Body.String())
	}
	if got := serve(handler, http.MethodGet, story+"/effort-records", ""); got.Code != http.StatusMethodNotAllowed || effort.calls != 1 {
		t.Errorf("GET effort-records = %d (writes %d), want 405 and no new write", got.Code, effort.calls)
	}
}
