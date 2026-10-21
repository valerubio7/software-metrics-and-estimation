package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/domain"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/transport/http"
)

type listMembersUseCase struct {
	result  application.ProjectMembers
	err     error
	calls   int
	project string
}

func (u *listMembersUseCase) Execute(_ context.Context, query application.ListMembersQuery) (application.ProjectMembers, error) {
	u.calls++
	u.project = query.ProjectID
	return u.result, u.err
}

func TestListMembersHandlerRejectsNonGetMethods(t *testing.T) {
	useCase := &listMembersUseCase{}
	request := httptest.NewRequest(http.MethodPost, "/projects/00000000-0000-0000-0000-000000000001/members", nil)
	request.SetPathValue("project_id", "00000000-0000-0000-0000-000000000001")
	recorder := httptest.NewRecorder()

	transporthttp.NewListMembersHandler(useCase).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, body %s; want 405", recorder.Code, recorder.Body.String())
	}
	if useCase.calls != 0 {
		t.Fatalf("use case called %d times for unsupported method", useCase.calls)
	}
}

func TestListMembersHandlerResponses(t *testing.T) {
	email := "ada@example.com"
	projectID := "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	for _, tc := range []struct {
		name       string
		pathValue  string
		result     application.ProjectMembers
		err        error
		wantStatus int
		wantBody   []string
	}{
		{
			name: "invalid uuid", pathValue: "not-a-uuid",
			wantStatus: http.StatusUnprocessableEntity, wantBody: []string{"validation_failed", "project_id"},
		},
		{
			name: "project not found", pathValue: projectID, err: application.ErrProjectNotFound,
			wantStatus: http.StatusNotFound, wantBody: []string{"project_not_found"},
		},
		{
			name: "empty collection", pathValue: projectID,
			result:     application.ProjectMembers{ProjectID: projectID, Members: []domain.Member{}},
			wantStatus: http.StatusOK, wantBody: []string{`"members":[]`},
		},
		{
			name: "members with and without email", pathValue: projectID,
			result: application.ProjectMembers{ProjectID: projectID, Members: []domain.Member{
				{ID: "m1", ProjectID: projectID, FullName: "Ada Lovelace", Email: &email},
				{ID: "m2", ProjectID: projectID, FullName: "Grace Hopper"},
			}},
			wantStatus: http.StatusOK,
			wantBody:   []string{`"full_name":"Ada Lovelace"`, `"email":"ada@example.com"`, `"full_name":"Grace Hopper"`, `"email":null`},
		},
		{
			name: "internal error", pathValue: projectID, err: errors.New("secret db detail"),
			wantStatus: http.StatusInternalServerError, wantBody: []string{"internal_error"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			useCase := &listMembersUseCase{result: tc.result, err: tc.err}
			request := httptest.NewRequest(http.MethodGet, "/projects/"+tc.pathValue+"/members", nil)
			request.SetPathValue("project_id", tc.pathValue)
			recorder := httptest.NewRecorder()
			transporthttp.NewListMembersHandler(useCase).ServeHTTP(recorder, request)

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, body %s; want %d", recorder.Code, recorder.Body.String(), tc.wantStatus)
			}
			for _, want := range tc.wantBody {
				if !strings.Contains(recorder.Body.String(), want) {
					t.Errorf("body %s does not contain %q", recorder.Body.String(), want)
				}
			}
			if tc.err != nil && strings.Contains(recorder.Body.String(), "secret") {
				t.Fatal("internal details leaked")
			}
			if tc.wantStatus == http.StatusUnprocessableEntity && useCase.calls != 0 {
				t.Fatal("use case called for invalid UUID")
			}
		})
	}
}
