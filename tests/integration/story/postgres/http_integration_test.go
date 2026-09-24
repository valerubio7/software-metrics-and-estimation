package postgres_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	projectpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/project/infrastructure/postgres"
	storypostgres "github.com/valerubio7/software-metrics-and-estimation/internal/story/infrastructure/postgres"
)

func TestStoryHTTPWithMigratedPostgres(t *testing.T) {
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	handler := api.NewHTTPHandler(projectpostgres.NewPostgresProjectRepository(pool), api.NewProjectID,
		api.StoryDependencies{Repository: storypostgres.NewPostgresStoryRepository(pool), GenerateID: api.NewProjectID})
	body := `{"title":"Registro","description":"Crear historia","priority":"media","acceptance_criteria":["Primero","Segundo"]}`

	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/stories", strings.NewReader(body)))
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body = %s", created.Code, created.Body.String())
	}
	var result struct {
		ID                 string   `json:"id"`
		ProjectID          string   `json:"project_id"`
		Status             string   `json:"status"`
		StoryPoints        *int     `json:"story_points"`
		AcceptanceCriteria []string `json:"acceptance_criteria"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	criteria := []string{"Primero", "Segundo"}
	if result.ID == "" || result.ProjectID != projectID || result.Status != "pendiente" ||
		result.StoryPoints != nil || !reflect.DeepEqual(result.AcceptanceCriteria, criteria) ||
		!strings.Contains(created.Body.String(), `"story_points":null`) {
		t.Fatalf("created story = %+v; body = %s", result, created.Body.String())
	}
	var linkedID, status string
	var storedCriteria []string
	var pointsNull bool
	if err := pool.QueryRow(context.Background(), `SELECT project_id::text, status, story_points IS NULL, acceptance_criteria FROM stories WHERE id = $1`, result.ID).
		Scan(&linkedID, &status, &pointsNull, &storedCriteria); err != nil {
		t.Fatalf("read persisted story: %v", err)
	}
	if linkedID != projectID || status != "pendiente" || !pointsNull || !reflect.DeepEqual(storedCriteria, criteria) || storyCount(t, pool) != 1 {
		t.Errorf("stored story: project=%q status=%q null=%v criteria=%v", linkedID, status, pointsNull, storedCriteria)
	}

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodPost,
		"/projects/9b6847be-c44c-47e8-a91d-69aa2874a80f/stories", strings.NewReader(body)))
	if missing.Code != http.StatusNotFound || !strings.Contains(missing.Body.String(), `"error":"project_not_found"`) {
		t.Errorf("missing project status = %d; body = %s", missing.Code, missing.Body.String())
	}
	if got := storyCount(t, pool); got != 1 {
		t.Errorf("story count after 404 = %d, want 1", got)
	}
}
