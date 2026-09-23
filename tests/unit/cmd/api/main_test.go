package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-estimation/internal/api"
	"github.com/valerubio7/software-metrics-estimation/internal/project/domain"
)

type fakeProjectRepository struct {
	projects []domain.Project
}

func (r *fakeProjectRepository) Create(_ context.Context, project domain.Project) error {
	r.projects = append(r.projects, project)
	return nil
}

func TestLoadConfigRequiresDatabaseURL(t *testing.T) {
	_, err := api.LoadConfig(func(string) string { return "" })

	if err == nil {
		t.Fatal("loadConfig() error = nil, want missing DATABASE_URL error")
	}
}

func TestLoadConfigUsesDatabaseURLAndDefaultAddress(t *testing.T) {
	config, err := api.LoadConfig(func(key string) string {
		if key == "DATABASE_URL" {
			return "postgres://postgres:postgres@localhost:5432/projects?sslmode=disable"
		}
		return ""
	})

	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if config.DatabaseURL == "" {
		t.Fatal("DatabaseURL is empty")
	}
	if config.Address != ":8080" {
		t.Errorf("Address = %q, want :8080", config.Address)
	}
}

func TestLoadConfigUsesConfiguredAddress(t *testing.T) {
	config, err := api.LoadConfig(func(key string) string {
		switch key {
		case "DATABASE_URL":
			return "postgres://postgres:postgres@localhost:5432/projects?sslmode=disable"
		case "HTTP_ADDR":
			return ":9090"
		default:
			return ""
		}
	})

	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if config.Address != ":9090" {
		t.Errorf("Address = %q, want :9090", config.Address)
	}
}

func TestNewProjectIDReturnsUUID(t *testing.T) {
	id := api.NewProjectID()

	if len(id) != 36 {
		t.Fatalf("newProjectID() = %q, want UUID string", id)
	}
}

func TestNewHTTPHandlerRegistersCreateProjectRoute(t *testing.T) {
	repository := &fakeProjectRepository{}
	handler := api.NewHTTPHandler(repository, func() string { return "5c21cbd4-d9a7-42df-9c3a-c0866f058746" })
	request := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(`{
		"name":"Metrics portal",
		"start_date":"2026-03-01",
		"planned_finish_date":"2026-06-30"
	}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if body["id"] != "5c21cbd4-d9a7-42df-9c3a-c0866f058746" {
		t.Errorf("id = %v, want generated UUID", body["id"])
	}
	if len(repository.projects) != 1 {
		t.Fatalf("repository projects = %d, want 1", len(repository.projects))
	}
}
