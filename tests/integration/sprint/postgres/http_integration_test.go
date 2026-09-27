package postgres_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	projectpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/project/infrastructure/postgres"
	sprintpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/infrastructure/postgres"
)

func TestSprintHTTPWithMigratedPostgres(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)
	repository := sprintpostgres.NewPostgresSprintRepository(pool)
	projectRepository := projectpostgres.NewPostgresProjectRepository(pool)
	handler := api.NewHTTPHandlerWithDependencies(projectRepository, api.NewProjectID, api.HTTPDependencies{
		Sprints: &api.SprintDependencies{Repository: repository, GenerateID: api.NewProjectID},
	})
	body := `{"sprint_goal":"  Deliver the first metrics flow  "}`
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/sprints", strings.NewReader(body)))
	var result struct {
		ID        string `json:"id"`
		ProjectID string `json:"project_id"`
		Goal      string `json:"sprint_goal"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if created.Code != http.StatusCreated || result.ID == "" || result.ProjectID != projectID || result.Goal != "  Deliver the first metrics flow  " {
		t.Fatalf("create status=%d result=%+v body=%s", created.Code, result, created.Body.String())
	}
	var storedProjectID, storedGoal string
	if err := pool.QueryRow(context.Background(), "SELECT project_id::text, sprint_goal FROM sprints WHERE id = $1", result.ID).Scan(&storedProjectID, &storedGoal); err != nil {
		t.Fatalf("read persisted sprint: %v", err)
	}
	if storedProjectID != result.ProjectID || storedGoal != result.Goal || sprintCount(t, pool) != 1 {
		t.Fatalf("persisted sprint = (%q, %q), response = (%q, %q)", storedProjectID, storedGoal, result.ProjectID, result.Goal)
	}

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodPost, "/projects/9b6847be-c44c-47e8-a91d-69aa2874a80f/sprints", strings.NewReader(body)))
	if missing.Code != http.StatusNotFound || !strings.Contains(missing.Body.String(), `"error":"project_not_found"`) || sprintCount(t, pool) != 1 {
		t.Fatalf("missing project status=%d count=%d body=%s", missing.Code, sprintCount(t, pool), missing.Body.String())
	}
	invalid := httptest.NewRecorder()
	handler.ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/sprints", strings.NewReader(`{}`)))
	if invalid.Code != http.StatusUnprocessableEntity || sprintCount(t, pool) != 1 {
		t.Fatalf("rejected request status=%d count=%d body=%s", invalid.Code, sprintCount(t, pool), invalid.Body.String())
	}
}
