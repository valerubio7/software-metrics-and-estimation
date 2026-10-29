package api_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/domain"
)

type defectListRepository struct {
	defects []domain.Defect
	err     error
	calls   int
}

func (r *defectListRepository) ListByProject(_ context.Context, projectID string) ([]domain.Defect, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	defects := make([]domain.Defect, 0, len(r.defects))
	for _, defect := range r.defects {
		if defect.ProjectID == projectID {
			defects = append(defects, defect)
		}
	}
	return defects, nil
}

func newDefectListHandler(repository *defectListRepository) http.Handler {
	return api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return defectRouteRecordID }, api.HTTPDependencies{
		Defect: &api.DefectDependencies{Repository: &fakeDefectRepository{}, GenerateID: func() string { return defectRouteRecordID }, Lister: repository},
	})
}

func TestDefectListRouteReturnsProjectScopedItems(t *testing.T) {
	repository := &defectListRepository{defects: []domain.Defect{
		{ID: defectRouteRecordID, ProjectID: defectRouteProjectID, StoryID: defectRouteStoryID, SprintID: defectRouteSprintID, Description: "Open issue", Severity: "alta", Status: "detectado"},
		{ID: "other", ProjectID: "other-project", Description: "Must not leak"},
		{ID: "resolved", ProjectID: defectRouteProjectID, StoryID: defectRouteStoryID, SprintID: defectRouteSprintID, Description: "Resolved issue", Severity: "media", Status: "resuelto", ResolutionSprintID: defectRouteSprintID},
	}}
	uppercaseProjectID := strings.ToUpper(defectRouteProjectID)
	response := serve(newDefectListHandler(repository), http.MethodGet, "/projects/"+uppercaseProjectID+"/defects", "")
	if response.Code != http.StatusOK || repository.calls != 1 {
		t.Fatalf("status/reads = %d/%d, want 200/1: %s", response.Code, repository.calls, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{`"project_id":"` + defectRouteProjectID + `"`, `"defects":[`, `"id":"` + defectRouteRecordID + `"`, `"description":"Open issue"`, `"severity":"alta"`, `"status":"detectado"`, `"story_id":"` + defectRouteStoryID + `"`, `"sprint_id":"` + defectRouteSprintID + `"`, `"resolution_sprint_id":null`, `"resolution_sprint_id":"` + defectRouteSprintID + `"`} {
		if !strings.Contains(body, want) {
			t.Errorf("response = %s, want %s", body, want)
		}
	}
	if strings.Contains(body, "Must not leak") || strings.Contains(body, "other-project") {
		t.Errorf("response leaked another project's defect: %s", body)
	}
}

func TestDefectListRouteReturnsExplicitEmptyArray(t *testing.T) {
	repository := &defectListRepository{defects: []domain.Defect{}}
	response := serve(newDefectListHandler(repository), http.MethodGet, defectRouteProject, "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"defects":[]`) {
		t.Fatalf("status/body = %d/%s, want successful empty defects array", response.Code, response.Body.String())
	}
}

func TestDefectListRouteMapsInvalidAndUnknownProjects(t *testing.T) {
	tests := []struct {
		name       string
		project    string
		err        error
		wantStatus int
		wantBody   string
	}{
		{name: "malformed project", project: "/projects/not-a-uuid/defects", wantStatus: http.StatusUnprocessableEntity, wantBody: `"project_id":"must be a valid UUID"`},
		{name: "missing project", project: defectRouteProject, err: application.ErrProjectNotFound, wantStatus: http.StatusNotFound, wantBody: `"error":"project_not_found"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repository := &defectListRepository{err: tc.err}
			response := serve(newDefectListHandler(repository), http.MethodGet, tc.project, "")
			if response.Code != tc.wantStatus || !strings.Contains(response.Body.String(), tc.wantBody) {
				t.Fatalf("status/body = %d/%s, want %d containing %s", response.Code, response.Body.String(), tc.wantStatus, tc.wantBody)
			}
			if tc.name == "malformed project" && repository.calls != 0 {
				t.Errorf("invalid project caused %d reads, want 0", repository.calls)
			}
		})
	}
}

func TestDefectListRouteRejectsMutationMethodsWithoutWrites(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			repository := &defectListRepository{}
			response := serve(newDefectListHandler(repository), method, defectRouteProject, defectRouteBody)
			if response.Code != http.StatusMethodNotAllowed || repository.calls != 0 {
				t.Fatalf("status/reads = %d/%d, want 405/0", response.Code, repository.calls)
			}
		})
	}
}

func TestDefectListRouteUnavailableWithoutV14Lister(t *testing.T) {
	handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return defectRouteRecordID }, api.HTTPDependencies{
		Defect: &api.DefectDependencies{Repository: &fakeDefectRepository{}, GenerateID: func() string { return defectRouteRecordID }},
	})
	response := serve(handler, http.MethodGet, defectRouteProject, "")
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d, want route absent before v14 (405)", response.Code)
	}
}
