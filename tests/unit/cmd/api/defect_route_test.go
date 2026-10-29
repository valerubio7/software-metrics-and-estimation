package api_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	defectapplication "github.com/valerubio7/software-metrics-and-estimation/internal/defect/application"
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

func (r *fakeDefectRepository) Update(_ context.Context, command defectapplication.UpdateDefectCommand) (defectdomain.Defect, error) {
	r.calls++
	r.record.Status = command.Status
	r.record.ResolutionSprintID = command.ResolutionSprintID
	if r.err != nil {
		return defectdomain.Defect{}, r.err
	}
	return r.record, nil
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
			if readiness.DefectUpdate != (tc.lookup == nil && !tc.dirty && tc.version >= 14) {
				t.Fatalf("DefectUpdate = %t, want %t", readiness.DefectUpdate, tc.lookup == nil && !tc.dirty && tc.version >= 14)
			}
			if readiness.DefectList != (tc.lookup == nil && !tc.dirty && tc.version >= 14) {
				t.Fatalf("DefectList = %t, want %t", readiness.DefectList, tc.lookup == nil && !tc.dirty && tc.version >= 14)
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

func TestDefectUpdateRoutePersistsLifecycleState(t *testing.T) {
	repository := &fakeDefectRepository{record: defectdomain.Defect{
		ID: defectRouteRecordID, ProjectID: defectRouteProjectID, StoryID: defectRouteStoryID, SprintID: defectRouteSprintID,
	}}
	dependencies := defectDependencies(repository)
	dependencies.Defect.Updater = repository
	handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return defectRouteRecordID }, dependencies)
	path := defectRouteProject + "/" + defectRouteRecordID
	body := `{"status":"resuelto","resolution_sprint_id":"` + defectRouteSprintID + `"}`

	response := serve(handler, http.MethodPatch, path, body)
	if response.Code != http.StatusOK || repository.calls != 1 {
		t.Fatalf("status/writes = %d/%d, want 200/1: %s", response.Code, repository.calls, response.Body.String())
	}
	if repository.record.Status != "resuelto" || repository.record.ResolutionSprintID != defectRouteSprintID ||
		!contains(response.Body.String(), `"status":"resuelto"`, `"resolution_sprint_id":"`+defectRouteSprintID+`"`) {
		t.Fatalf("persisted=%+v response=%s", repository.record, response.Body.String())
	}
}

func TestDefectUpdateRouteMapsValidationAndDomainErrors(t *testing.T) {
	for _, tc := range []struct {
		name       string
		body       string
		repository error
		wantStatus int
		wantError  string
	}{
		{name: "missing resolution Sprint", body: `{"status":"resuelto"}`, wantStatus: http.StatusUnprocessableEntity, wantError: "validation_failed"},
		{name: "missing defect", body: `{"status":"detectado"}`, repository: defectapplication.ErrDefectNotFound, wantStatus: http.StatusNotFound, wantError: "defect_not_found"},
		{name: "invalid resolution Sprint", body: `{"status":"resuelto","resolution_sprint_id":"` + defectRouteSprintID + `"}`, repository: defectapplication.ErrResolutionSprintNotFound, wantStatus: http.StatusNotFound, wantError: "resolution_sprint_not_found"},
		{name: "chronology conflict", body: `{"status":"resuelto","resolution_sprint_id":"` + defectRouteSprintID + `"}`, repository: defectapplication.ErrResolutionSprintBeforeDetection, wantStatus: http.StatusConflict, wantError: "resolution_sprint_before_detection"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository := &fakeDefectRepository{err: tc.repository}
			dependencies := defectDependencies(repository)
			dependencies.Defect.Updater = repository
			handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return defectRouteRecordID }, dependencies)
			response := serve(handler, http.MethodPatch, defectRouteProject+"/"+defectRouteRecordID, tc.body)
			if response.Code != tc.wantStatus || !strings.Contains(response.Body.String(), `"error":"`+tc.wantError+`"`) {
				t.Fatalf("response = %d %s, want status %d and error %s", response.Code, response.Body.String(), tc.wantStatus, tc.wantError)
			}
			if tc.wantStatus == http.StatusUnprocessableEntity && repository.calls != 0 {
				t.Errorf("invalid request caused %d writes, want 0", repository.calls)
			}
		})
	}
}

func TestDefectUpdateRouteIsGatedWithoutResolutionMigration(t *testing.T) {
	repository := &fakeDefectRepository{}
	handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return defectRouteRecordID }, defectDependencies(repository))
	response := serve(handler, http.MethodPatch, defectRouteProject+"/"+defectRouteRecordID, `{"status":"resuelto"}`)
	if response.Code != http.StatusNotFound || repository.calls != 0 {
		t.Fatalf("status/writes = %d/%d, want 404/0", response.Code, repository.calls)
	}
}

func TestDefectUpdateRouteRejectsMalformedJSONWithoutWrites(t *testing.T) {
	repository := &fakeDefectRepository{}
	dependencies := defectDependencies(repository)
	dependencies.Defect.Updater = repository
	handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return defectRouteRecordID }, dependencies)
	response := serve(handler, http.MethodPatch, defectRouteProject+"/"+defectRouteRecordID, `{"status":"resuelto","unknown":true}`)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"error":"invalid_request"`) || repository.calls != 0 {
		t.Fatalf("status/writes/body = %d/%d/%s, want 400/0 invalid_request", response.Code, repository.calls, response.Body.String())
	}
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
