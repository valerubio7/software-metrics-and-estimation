package api_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	defectdomain "github.com/valerubio7/software-metrics-and-estimation/internal/defect/domain"
)

const (
	defectRouteProjectID = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	defectRouteStoryID   = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	defectRouteSprintID  = "b2a6a455-06e2-41ee-b011-7462c71375a0"
	defectRouteRecordID  = "f47ac10b-58cc-4372-a567-0e02b2c3d479"

	defectRouteProject = "/projects/" + defectRouteProjectID + "/defects"
	defectRouteBody    = `{"story_id":"` + defectRouteStoryID + `","sprint_id":"` + defectRouteSprintID + `","description":"El login acepta contraseñas vacías","severity":"alta","status":"detectado"}`
)

// fakeDefectRepository counts writes and captures the last record received.
type fakeDefectRepository struct {
	calls  int
	record defectdomain.Defect
	err    error
}

func (r *fakeDefectRepository) Create(_ context.Context, defect defectdomain.Defect) error {
	r.calls++
	r.record = defect
	return r.err
}

func defectDependencies(repository *fakeDefectRepository) api.HTTPDependencies {
	return api.HTTPDependencies{Defect: &api.DefectDependencies{
		Repository: repository,
		GenerateID: func() string { return defectRouteRecordID },
	}}
}

func TestMigrationReadinessResolvesDefectByVersion(t *testing.T) {
	for _, tc := range []struct {
		name    string
		version int
		dirty   bool
		lookup  error
		want    bool
	}{
		{name: "v12", version: 12},
		{name: "v13", version: 13, want: true},
		{name: "v14", version: 14, want: true},
		{name: "dirty v13", version: 13, dirty: true},
		{name: "lookup error", version: 13, lookup: errors.New("query failed")},
		{name: "missing migration table", lookup: errors.New("missing schema_migrations")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			readiness := api.ResolveMigrationReadiness(tc.version, tc.dirty, tc.lookup)
			if readiness.Defect != tc.want {
				t.Fatalf("Defect = %t, want %t (readiness = %+v)", readiness.Defect, tc.want, readiness)
			}
			// The defect flag never alters the flags of the existing modules.
			clean := tc.lookup == nil && !tc.dirty
			if readiness.Effort != (clean && tc.version >= 12) || readiness.Tasks != (clean && tc.version >= 9) || !readiness.Projects {
				t.Fatalf("existing readiness flags changed: %+v", readiness)
			}
		})
	}
}

func TestDefectRouteRequiresExplicitDependency(t *testing.T) {
	t.Run("nil dependency answers 404 without writes", func(t *testing.T) {
		repository := &fakeDefectRepository{}
		handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return defectRouteRecordID }, api.HTTPDependencies{})

		response := serve(handler, http.MethodPost, defectRouteProject, defectRouteBody)
		if response.Code != http.StatusNotFound || repository.calls != 0 {
			t.Fatalf("status/writes = %d/%d, want 404/0: %s", response.Code, repository.calls, response.Body.String())
		}
	})

	t.Run("present dependency answers 201 with one write", func(t *testing.T) {
		repository := &fakeDefectRepository{}
		handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return defectRouteRecordID }, defectDependencies(repository))

		response := serve(handler, http.MethodPost, defectRouteProject, defectRouteBody)
		if response.Code != http.StatusCreated || repository.calls != 1 {
			t.Fatalf("status/writes = %d/%d, want 201/1: %s", response.Code, repository.calls, response.Body.String())
		}
		if repository.record.ID != defectRouteRecordID || repository.record.ProjectID != defectRouteProjectID || repository.record.StoryID != defectRouteStoryID || repository.record.SprintID != defectRouteSprintID {
			t.Errorf("stored defect = %+v, want generated ID and route/body IDs", repository.record)
		}
		if !contains(response.Body.String(), `"id":"`+defectRouteRecordID+`"`, `"severity":"alta"`, `"status":"detectado"`) {
			t.Errorf("body = %s, want the generated id and catalog values", response.Body.String())
		}
		if contains(response.Body.String(), `"created_at"`, `"seq"`) {
			t.Errorf("body = %s, want no created_at and no seq", response.Body.String())
		}
	})

	t.Run("GET on the route answers 405 without writes", func(t *testing.T) {
		repository := &fakeDefectRepository{}
		handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return defectRouteRecordID }, defectDependencies(repository))

		response := serve(handler, http.MethodGet, defectRouteProject, "")
		if response.Code != http.StatusMethodNotAllowed || repository.calls != 0 {
			t.Fatalf("GET status/writes = %d/%d, want 405/0", response.Code, repository.calls)
		}
	})
}

// contains reports whether the body includes every supplied fragment.
func contains(body string, fragments ...string) bool {
	for _, fragment := range fragments {
		if !strings.Contains(body, fragment) {
			return false
		}
	}
	return true
}
