package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	storyapplication "github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	storydomain "github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

type storyPointsRepo struct{ calls int }

func (r *storyPointsRepo) SetStoryPoints(_ context.Context, project, story string, points int) (storydomain.Story, error) {
	r.calls++
	n := points
	return storydomain.Story{ID: story, ProjectID: project, Title: "Story", Priority: "alta", Status: storydomain.StatusPending, StoryPoints: &n}, nil
}

func TestStoryPointsRouteFollowsActualMigrationDependencySelection(t *testing.T) {
	const id = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	for _, tc := range []struct {
		name                    string
		version                 int
		dirty                   bool
		lookup                  error
		wantPoints, wantGeneric bool
	}{
		{name: "clean v1", version: 1}, {name: "clean v2", version: 2, wantPoints: true}, {name: "clean v3", version: 3, wantPoints: true, wantGeneric: true}, {name: "clean v4+", version: 12, wantPoints: true, wantGeneric: true}, {name: "dirty v2", version: 2, dirty: true}, {name: "dirty v3", version: 3, dirty: true}, {name: "lookup failure", version: 2, lookup: errors.New("lookup failed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			readiness := api.ResolveMigrationReadiness(tc.version, tc.dirty, tc.lookup)
			repo := &storyPointsRepo{}
			deps := api.HTTPDependencies{}
			if readiness.Stories {
				deps.Stories = &api.StoryDependencies{Repository: &fakeStoryRepository{}, GenerateID: func() string { return id }}
				if readiness.StoryPoints {
					deps.Stories.StoryPointsSetter = repo
				}
				if tc.version >= 3 && !tc.dirty && tc.lookup == nil {
					deps.Stories.Updater = storyUpdaterStub{}
				}
			}
			h := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return id }, deps)
			points := httptest.NewRecorder()
			h.ServeHTTP(points, httptest.NewRequest(http.MethodPut, "/projects/"+id+"/stories/"+id+"/story-points", strings.NewReader(`{"story_points":13}`)))
			want := http.StatusNotFound
			if tc.wantPoints {
				want = http.StatusOK
			}
			if points.Code != want || (repo.calls == 1) != tc.wantPoints {
				t.Errorf("story-points status/calls=%d/%d want=%d/enabled=%t", points.Code, repo.calls, want, tc.wantPoints)
			}
			generic := httptest.NewRecorder()
			h.ServeHTTP(generic, httptest.NewRequest(http.MethodPut, "/projects/"+id+"/stories/"+id, strings.NewReader(`{}`)))
			wantGeneric := http.StatusNotFound
			if tc.wantGeneric {
				wantGeneric = http.StatusUnprocessableEntity
			}
			if generic.Code != wantGeneric {
				t.Errorf("generic PUT status=%d want=%d", generic.Code, wantGeneric)
			}
		})
	}
}

type storyUpdaterStub struct{}

func (storyUpdaterStub) Update(context.Context, storydomain.Story) (storydomain.Story, error) {
	return storydomain.Story{}, storyapplication.ErrStoryNotFound
}
