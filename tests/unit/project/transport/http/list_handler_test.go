package transporthttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/project/transport/http"
)

type listReader struct {
	projects []domain.Project
	err      error
	calls    int
}

func (r *listReader) List(_ context.Context) ([]domain.Project, error) {
	r.calls++
	return r.projects, r.err
}

func newListHandler(reader *listReader) http.Handler {
	now := func() time.Time { return time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC) }
	return transporthttp.NewListProjectsHandler(application.NewListProjectsUseCase(reader), now)
}

func TestListProjectsHandlerSerializesCollectionAndStatus(t *testing.T) {
	reader := &listReader{projects: []domain.Project{
		{ID: "dc46073f-51fb-4393-8e80-6b7f84201cc1", Name: "Portal", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), PlannedFinishDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)},
	}}
	response := httptest.NewRecorder()
	newListHandler(reader).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/projects", nil))

	if response.Code != http.StatusOK || reader.calls != 1 {
		t.Fatalf("status=%d calls=%d body=%s", response.Code, reader.calls, response.Body.String())
	}
	var body struct {
		Projects []struct {
			ID                string `json:"id"`
			Name              string `json:"name"`
			StartDate         string `json:"start_date"`
			PlannedFinishDate string `json:"planned_finish_date"`
			Status            string `json:"status"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Projects) != 1 {
		t.Fatalf("projects = %#v", body.Projects)
	}
	got := body.Projects[0]
	if got.ID != "dc46073f-51fb-4393-8e80-6b7f84201cc1" || got.Name != "Portal" || got.Status != "active" ||
		got.StartDate != "2026-01-01" || got.PlannedFinishDate != "2026-12-31" {
		t.Fatalf("project = %#v", got)
	}
}

func TestListProjectsHandlerSerializesEmptyCollectionAsArray(t *testing.T) {
	response := httptest.NewRecorder()
	newListHandler(&listReader{}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/projects", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	var body struct {
		Projects []json.RawMessage `json:"projects"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Projects == nil || len(body.Projects) != 0 {
		t.Fatalf("projects = %s, want empty array", response.Body.String())
	}
}

func TestListProjectsHandlerRejectsOtherMethodsAndHidesInternalErrors(t *testing.T) {
	post := httptest.NewRecorder()
	newListHandler(&listReader{}).ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/projects", nil))
	if post.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want 405", post.Code)
	}

	failing := &listReader{err: errors.New("connection refused: host=db")}
	internal := httptest.NewRecorder()
	newListHandler(failing).ServeHTTP(internal, httptest.NewRequest(http.MethodGet, "/projects", nil))
	if internal.Code != http.StatusInternalServerError || strings.Contains(internal.Body.String(), "connection refused") {
		t.Fatalf("internal status=%d body=%s, want 500 without internal detail", internal.Code, internal.Body.String())
	}
}
