package api_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
)

// listableProjectRepository stands in for the PostgreSQL repository, which reads the collection.
type listableProjectRepository struct {
	*fakeProjectRepository
	listed    []domain.Project
	listErr   error
	listCalls int
}

func (r *listableProjectRepository) List(_ context.Context) ([]domain.Project, error) {
	r.listCalls++
	return r.listed, r.listErr
}

func TestProjectCollectionRouteIsComposedOnlyForListableRepository(t *testing.T) {
	const id = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	today := time.Now().UTC()
	repository := &listableProjectRepository{
		fakeProjectRepository: &fakeProjectRepository{},
		listed: []domain.Project{
			{ID: id, Name: "Portal", StartDate: today.AddDate(0, 0, -1), PlannedFinishDate: today.AddDate(0, 0, 1)},
		},
	}
	handler := api.NewHTTPHandlerWithDependencies(repository, func() string { return id }, api.HTTPDependencies{})

	response := serve(handler, http.MethodGet, "/projects", "")
	if response.Code != http.StatusOK || repository.listCalls != 1 {
		t.Fatalf("GET /projects = %d (reads %d) body=%s", response.Code, repository.listCalls, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"projects":[`) || !strings.Contains(response.Body.String(), `"status":"active"`) || !strings.Contains(response.Body.String(), `"name":"Portal"`) {
		t.Fatalf("collection body = %s", response.Body.String())
	}

	// A repository without the listing port keeps the collection unreadable: GET stays 405.
	legacy := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return id }, api.HTTPDependencies{})
	if got := serve(legacy, http.MethodGet, "/projects", ""); got.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /projects without lister = %d, want 405", got.Code)
	}
}

func TestProjectCollectionRouteSerializesEmptyArrayAndHidesInternalErrors(t *testing.T) {
	const id = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	empty := &listableProjectRepository{fakeProjectRepository: &fakeProjectRepository{}}
	handler := api.NewHTTPHandlerWithDependencies(empty, func() string { return id }, api.HTTPDependencies{})
	if got := serve(handler, http.MethodGet, "/projects", ""); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"projects":[]`) {
		t.Fatalf("empty collection = %d %s", got.Code, got.Body.String())
	}

	failing := &listableProjectRepository{fakeProjectRepository: &fakeProjectRepository{}, listErr: errors.New("connection refused: host=db")}
	failingHandler := api.NewHTTPHandlerWithDependencies(failing, func() string { return id }, api.HTTPDependencies{})
	got := serve(failingHandler, http.MethodGet, "/projects", "")
	if got.Code != http.StatusInternalServerError || strings.Contains(got.Body.String(), "connection refused") {
		t.Fatalf("failing collection = %d %s, want 500 without internal detail", got.Code, got.Body.String())
	}
}
