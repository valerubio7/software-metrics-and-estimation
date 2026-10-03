package postgres_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	projectpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/project/infrastructure/postgres"
	taskpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/task/infrastructure/postgres"
)

// newTaskCompositionHandler composes the real HTTP handler over the fixture pool, exactly
// as cmd/api/main.go does once migration 000009 is clean.
func newTaskCompositionHandler(pool *pgxpool.Pool) http.Handler {
	return api.NewHTTPHandlerWithDependencies(
		projectpostgres.NewPostgresProjectRepository(pool),
		api.NewProjectID,
		api.HTTPDependencies{
			Tasks: &api.TaskDependencies{
				Repository: taskpostgres.NewPostgresTaskRepository(pool),
				GenerateID: api.NewProjectID,
			},
		},
	)
}

func TestCreateTasksHTTPEndToEnd(t *testing.T) {
	t.Run("valid batch persists and returns generated tasks", func(t *testing.T) {
		pool := taskFixture(t)
		handler := newTaskCompositionHandler(pool)

		request := httptest.NewRequest(http.MethodPost,
			"/projects/"+taskProjectAID+"/sprints/"+taskSprintS1ID+"/stories/"+taskStoryH1ID+"/tasks",
			strings.NewReader(`{"tasks":[{"title":"Primera","estimated_hours":4.5},{"title":"Segunda"}]}`))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", response.Code, response.Body.String())
		}
		var body struct {
			Tasks []struct{ ID string } `json:"tasks"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(body.Tasks) != 2 || body.Tasks[0].ID == "" || body.Tasks[1].ID == "" {
			t.Fatalf("response tasks = %#v, want 2 generated UUIDs", body.Tasks)
		}
		if got := taskCount(t, pool); got != 2 {
			t.Errorf("task count = %d, want 2", got)
		}
	})

	t.Run("story not assigned to sprint responds 409 without writes", func(t *testing.T) {
		pool := taskFixture(t)
		handler := newTaskCompositionHandler(pool)

		request := httptest.NewRequest(http.MethodPost,
			"/projects/"+taskProjectAID+"/sprints/"+taskSprintS1ID+"/stories/"+taskStoryH2ID+"/tasks",
			strings.NewReader(`{"tasks":[{"title":"Tarea"}]}`))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "story_not_in_sprint") {
			t.Fatalf("status/body = %d/%s, want 409 story_not_in_sprint", response.Code, response.Body.String())
		}
		if got := taskCount(t, pool); got != 0 {
			t.Errorf("task count = %d, want 0", got)
		}
	})

	t.Run("an invalid element in the batch responds 422 with the indexed key without writes", func(t *testing.T) {
		pool := taskFixture(t)
		handler := newTaskCompositionHandler(pool)

		request := httptest.NewRequest(http.MethodPost,
			"/projects/"+taskProjectAID+"/sprints/"+taskSprintS1ID+"/stories/"+taskStoryH1ID+"/tasks",
			strings.NewReader(`{"tasks":[{"title":"Valida"},{"title":"   "}]}`))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), `"tasks[1].title":"is required"`) {
			t.Fatalf("status/body = %d/%s, want 422 with tasks[1].title", response.Code, response.Body.String())
		}
		if got := taskCount(t, pool); got != 0 {
			t.Errorf("task count = %d, want 0", got)
		}
	})
}
