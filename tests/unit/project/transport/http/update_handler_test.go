package transporthttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/project/transport/http"
)

type updateRepository struct {
	calls   int
	project domain.Project
	err     error
}

func (r *updateRepository) Create(context.Context, domain.Project) error { return nil }
func (r *updateRepository) Update(_ context.Context, project domain.Project) error {
	r.calls++
	r.project = project
	return r.err
}

func TestUpdateProjectHandlerReturnsReplacementAndMapsErrors(t *testing.T) {
	const projectID = "dc46073f-51fb-4393-8e80-6b7f84201cc1"
	tests := []struct {
		name            string
		id              string
		body            string
		repositoryError error
		status          int
		writes          int
	}{
		{name: "successful replacement", id: projectID, body: validUpdateBody, status: http.StatusOK, writes: 1},
		{name: "equal dates", id: projectID, body: `{"name":"Updated","start_date":"2026-03-01","planned_finish_date":"2026-03-01"}`, status: http.StatusOK, writes: 1},
		{name: "invalid UUID", id: "not-a-uuid", body: validUpdateBody, status: http.StatusUnprocessableEntity},
		{name: "not found", id: projectID, body: validUpdateBody, repositoryError: application.ErrProjectNotFound, status: http.StatusNotFound, writes: 1},
		{name: "repository failure", id: projectID, body: validUpdateBody, repositoryError: errors.New("database secret"), status: http.StatusInternalServerError, writes: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &updateRepository{err: tt.repositoryError}
			handler := transporthttp.NewUpdateProjectHandler(application.NewUpdateProjectUseCase(repository))
			response := executeUpdateRequest(handler, tt.id, tt.body)
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, tt.status, response.Body.String())
			}
			if repository.calls != tt.writes {
				t.Fatalf("repository writes = %d, want %d", repository.calls, tt.writes)
			}
			if tt.status == http.StatusOK {
				var body map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body["id"] != projectID || body["name"] != "Updated" || body["start_date"] != "2026-03-01" || body["planned_finish_date"] != finishDateFor(tt.body) {
					t.Errorf("response = %v", body)
				}
			}
			if tt.status == http.StatusInternalServerError && strings.Contains(response.Body.String(), "database secret") {
				t.Error("response leaked repository error")
			}
		})
	}
}

func TestUpdateProjectHandlerRejectsMalformedOrIncompleteRequests(t *testing.T) {
	tests := []struct{ name, body string }{
		{"malformed JSON", `{"name":`},
		{"trailing JSON", validUpdateBody + ` {}`},
		{"unknown field", `{"name":"Updated","start_date":"2026-03-01","planned_finish_date":"2026-06-30","status":"done"}`},
		{"missing name", `{"start_date":"2026-03-01","planned_finish_date":"2026-06-30"}`},
		{"missing start date", `{"name":"Updated","planned_finish_date":"2026-06-30"}`},
		{"missing finish date", `{"name":"Updated","start_date":"2026-03-01"}`},
		{"blank name", `{"name":"  ","start_date":"2026-03-01","planned_finish_date":"2026-06-30"}`},
		{"invalid date", `{"name":"Updated","start_date":"2026-02-30","planned_finish_date":"2026-06-30"}`},
		{"finish before start", `{"name":"Updated","start_date":"2026-03-01","planned_finish_date":"2026-02-28"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &updateRepository{}
			handler := transporthttp.NewUpdateProjectHandler(application.NewUpdateProjectUseCase(repository))
			response := executeUpdateRequest(handler, "dc46073f-51fb-4393-8e80-6b7f84201cc1", tt.body)
			if response.Code != http.StatusBadRequest && response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 400 or 422", response.Code)
			}
			if repository.calls != 0 {
				t.Errorf("repository writes = %d, want 0", repository.calls)
			}
			assertActionableError(t, response)
		})
	}
}

const validUpdateBody = `{"name":"Updated","start_date":"2026-03-01","planned_finish_date":"2026-06-30"}`

func executeUpdateRequest(handler http.Handler, projectID, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPut, "/projects/"+projectID, strings.NewReader(body))
	request.SetPathValue("project_id", projectID)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func finishDateFor(body string) string {
	var request struct {
		PlannedFinishDate string `json:"planned_finish_date"`
	}
	if err := json.Unmarshal([]byte(body), &request); err != nil {
		return ""
	}
	return request.PlannedFinishDate
}
