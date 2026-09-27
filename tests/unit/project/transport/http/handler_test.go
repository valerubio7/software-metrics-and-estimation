package transporthttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/project/transport/http"
)

type fakeRepository struct {
	calls int
	err   error
}

func (r *fakeRepository) Create(_ context.Context, _ domain.Project) error {
	r.calls++
	return r.err
}

func (r *fakeRepository) Update(_ context.Context, _ domain.Project) error {
	r.calls++
	return r.err
}

func TestCreateProjectHandlerCreatesProjectWithoutStatus(t *testing.T) {
	repository := &fakeRepository{}
	handler := newHandler(repository)
	request := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewBufferString(`{
		"name":"Metrics portal",
		"start_date":"2026-03-01",
		"planned_finish_date":"2026-06-30"
	}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if string(body["id"]) != `"generated-id"` {
		t.Errorf("id = %s, want generated id", body["id"])
	}
	if string(body["name"]) != `"Metrics portal"` {
		t.Errorf("name = %s, want Metrics portal", body["name"])
	}
	if string(body["start_date"]) != `"2026-03-01"` {
		t.Errorf("start_date = %s, want 2026-03-01", body["start_date"])
	}
	if string(body["planned_finish_date"]) != `"2026-06-30"` {
		t.Errorf("planned_finish_date = %s, want 2026-06-30", body["planned_finish_date"])
	}
	if _, ok := body["status"]; ok {
		t.Error("response must not contain status")
	}
	if repository.calls != 1 {
		t.Errorf("repository calls = %d, want 1", repository.calls)
	}
}

func TestCreateProjectHandlerRejectsMalformedRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ""},
		{name: "malformed JSON", body: `{"name":`},
		{name: "incompatible field type", body: `{"name":123,"start_date":"2026-03-01","planned_finish_date":"2026-06-30"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeRepository{}
			response := executeRequest(newHandler(repository), tt.body)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
			}
			assertActionableError(t, response)
			if repository.calls != 0 {
				t.Errorf("repository calls = %d, want 0", repository.calls)
			}
		})
	}
}

func TestCreateProjectHandlerReturnsValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		field string
	}{
		{name: "missing name", body: `{"start_date":"2026-03-01","planned_finish_date":"2026-06-30"}`, field: "name"},
		{name: "invalid start date", body: `{"name":"Metrics portal","start_date":"2026-02-30","planned_finish_date":"2026-06-30"}`, field: "start_date"},
		{name: "missing finish date", body: `{"name":"Metrics portal","start_date":"2026-03-01"}`, field: "planned_finish_date"},
		{name: "invalid finish date", body: `{"name":"Metrics portal","start_date":"2026-03-01","planned_finish_date":"not-a-date"}`, field: "planned_finish_date"},
		{name: "finish before start", body: `{"name":"Metrics portal","start_date":"2026-03-01","planned_finish_date":"2026-02-28"}`, field: "planned_finish_date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeRepository{}
			response := executeRequest(newHandler(repository), tt.body)

			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusUnprocessableEntity, response.Body.String())
			}
			var body struct {
				Error  string            `json:"error"`
				Fields map[string]string `json:"fields"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("response is not JSON: %v", err)
			}
			if body.Error != "validation_failed" {
				t.Errorf("error = %q, want validation_failed", body.Error)
			}
			if body.Fields[tt.field] == "" {
				t.Errorf("fields = %#v, want actionable %q error", body.Fields, tt.field)
			}
			if repository.calls != 0 {
				t.Errorf("repository calls = %d, want 0", repository.calls)
			}
		})
	}
}

func TestCreateProjectHandlerAcceptsEqualDates(t *testing.T) {
	repository := &fakeRepository{}
	response := executeRequest(newHandler(repository), `{"name":"Metrics portal","start_date":"2026-03-01","planned_finish_date":"2026-03-01"}`)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if repository.calls != 1 {
		t.Errorf("repository calls = %d, want 1", repository.calls)
	}
}

func TestCreateProjectHandlerHidesRepositoryErrors(t *testing.T) {
	repository := &fakeRepository{err: errors.New("database password is invalid")}
	response := executeRequest(newHandler(repository), `{"name":"Metrics portal","start_date":"2026-03-01","planned_finish_date":"2026-06-30"}`)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
	if bytes.Contains(response.Body.Bytes(), []byte("database password is invalid")) {
		t.Error("response leaked repository error details")
	}
	assertActionableError(t, response)
}

func newHandler(repository application.ProjectRepository) http.Handler {
	useCase := application.NewCreateProjectUseCase(repository, func() string { return "generated-id" })
	return transporthttp.NewCreateProjectHandler(useCase)
}

func executeRequest(handler http.Handler, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertActionableError(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	var body struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if body.Error == "" || body.Message == "" {
		t.Errorf("error response = %#v, want error and actionable message", body)
	}
}
