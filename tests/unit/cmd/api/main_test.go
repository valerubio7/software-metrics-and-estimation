package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
	storydomain "github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

type fakeProjectRepository struct {
	projects []domain.Project
}

func (r *fakeProjectRepository) Create(_ context.Context, project domain.Project) error {
	r.projects = append(r.projects, project)
	return nil
}

type fakeStoryRepository struct {
	stories []storydomain.Story
}

func (r *fakeStoryRepository) Create(_ context.Context, story storydomain.Story) error {
	r.stories = append(r.stories, story)
	return nil
}

func TestNewHTTPHandlerRegistersStoryAndPreservesProjects(t *testing.T) {
	projects := &fakeProjectRepository{}
	stories := &fakeStoryRepository{}
	handler := api.NewHTTPHandler(projects, func() string { return "5c21cbd4-d9a7-42df-9c3a-c0866f058746" },
		api.StoryDependencies{Repository: stories, GenerateID: func() string { return "e99c05a4-03ea-4c19-8f59-286dd59e7aa1" }})
	projectID := "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	body := `{"title":"Registro","description":"Crear historia","priority":"media","acceptance_criteria":["Primero","Segundo"]}`
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/stories", strings.NewReader(body)))
	if created.Code != http.StatusCreated || len(stories.stories) != 1 {
		t.Fatalf("story status = %d, writes = %d; body = %s", created.Code, len(stories.stories), created.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["project_id"] != projectID || result["story_points"] != nil || result["status"] != "pendiente" {
		t.Errorf("story response = %v", result)
	}

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(method, "/projects/"+projectID+"/stories", strings.NewReader(body)))
		if response.Code != http.StatusMethodNotAllowed || len(stories.stories) != 1 {
			t.Errorf("%s status = %d, writes = %d", method, response.Code, len(stories.stories))
		}
	}

	project := httptest.NewRecorder()
	handler.ServeHTTP(project, httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(`{"name":"Metrics portal","start_date":"2026-03-01","planned_finish_date":"2026-06-30"}`)))
	if project.Code != http.StatusCreated || len(projects.projects) != 1 {
		t.Errorf("project status = %d, writes = %d; body = %s", project.Code, len(projects.projects), project.Body.String())
	}
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
