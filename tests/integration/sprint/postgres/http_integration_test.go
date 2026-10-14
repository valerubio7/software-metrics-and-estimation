package postgres_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

type historyStoryBody struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Priority    string  `json:"priority"`
	StoryPoints *int    `json:"story_points"`
	CompletedAt *string `json:"completed_at"`
	Completed   bool    `json:"completed"`
}

type historySprintBody struct {
	ID         string             `json:"id"`
	ProjectID  string             `json:"project_id"`
	SprintGoal string             `json:"sprint_goal"`
	IsClosed   bool               `json:"is_closed"`
	Stories    []historyStoryBody `json:"stories"`
}

func TestSprintHistoryHTTPWithMigratedPostgres(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)
	insertListProject(t, pool, historyOtherProjectID, "Without closed sprints")
	repository := sprintpostgres.NewPostgresSprintRepository(pool)
	handler := api.NewHTTPHandlerWithDependencies(projectpostgres.NewPostgresProjectRepository(pool), api.NewProjectID, api.HTTPDependencies{
		Sprints: &api.SprintDependencies{Repository: repository, GenerateID: api.NewProjectID, Closer: repository, Lister: repository},
	})
	get := func(target string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		return response
	}
	createSprint := func(goal string) string {
		t.Helper()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/sprints", strings.NewReader(`{"sprint_goal":"`+goal+`"}`)))
		var created struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil || response.Code != http.StatusCreated || created.ID == "" {
			t.Fatalf("create sprint %q status=%d err=%v body=%s", goal, response.Code, err, response.Body.String())
		}
		return created.ID
	}
	closeSprint := func(id string) {
		t.Helper()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/sprints/"+id+"/closure", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("close sprint %s status=%d body=%s", id, response.Code, response.Body.String())
		}
	}

	// Creation order: first, second, open (never closed), empty. Closure order differs on purpose.
	first, second, open, empty := createSprint("First"), createSprint("Second"), createSprint("Still open"), createSprint("Empty")
	five := 5
	carried, fresh := historyID(1), historyID(2)
	seedListStory(t, pool, projectID, carried, "Carried over", "alta", &five)
	seedListStory(t, pool, projectID, fresh, "Fresh", "baja", nil)
	// Non-UTC instant: the response must render it in UTC.
	completedAt := time.Date(2026, 10, 11, 7, 30, 0, 0, time.FixedZone("UTC-3", -3*60*60))
	assignListStory(t, pool, projectID, first, carried, &completedAt)
	assignListStory(t, pool, projectID, first, fresh, nil)
	assignListStory(t, pool, projectID, second, carried, nil)
	assignListStory(t, pool, projectID, open, fresh, nil)
	if _, err := pool.Exec(context.Background(), "UPDATE stories SET status = 'completada' WHERE id = $1", carried); err != nil {
		t.Fatalf("mark story completed: %v", err)
	}
	for _, id := range []string{second, empty, first} {
		closeSprint(id)
	}

	response := get("/projects/" + projectID + "/sprints")
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("GET status=%d content-type=%q body=%s", response.Code, response.Header().Get("Content-Type"), response.Body.String())
	}
	var history struct {
		ProjectID string              `json:"project_id"`
		Sprints   []historySprintBody `json:"sprints"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &history); err != nil {
		t.Fatalf("decode history: %v", err)
	}
	if history.ProjectID != projectID || len(history.Sprints) != 3 {
		t.Fatalf("history = %+v, want project %s with three closed sprints (open sprint excluded)", history, projectID)
	}
	if got := []string{history.Sprints[0].ID, history.Sprints[1].ID, history.Sprints[2].ID}; got[0] != empty || got[1] != second || got[2] != first {
		t.Fatalf("sprint order = %v, want most recently created first [%s %s %s]", got, empty, second, first)
	}
	for _, sprint := range history.Sprints {
		if sprint.ID == open || !sprint.IsClosed || sprint.ProjectID != projectID {
			t.Errorf("sprint %+v must be a closed sprint of the queried project", sprint)
		}
	}
	if !strings.Contains(response.Body.String(), `{"id":"`+empty+`","project_id":"`+projectID+`","sprint_goal":"Empty","is_closed":true,"stories":[]}`) {
		t.Errorf("empty closed sprint must render \"stories\":[]; body=%s", response.Body.String())
	}
	firstStories := history.Sprints[2].Stories
	if len(firstStories) != 2 || firstStories[0].ID != carried || firstStories[1].ID != fresh {
		t.Fatalf("first sprint stories = %+v, want [carried fresh] in creation order", firstStories)
	}
	if got := firstStories[0]; got.Title != "Carried over" || got.Priority != "alta" || got.StoryPoints == nil || *got.StoryPoints != 5 ||
		!got.Completed || got.CompletedAt == nil || *got.CompletedAt != "2026-10-11T10:30:00Z" {
		t.Errorf("completed story = %+v, want 5 points completed at 2026-10-11T10:30:00Z", got)
	}
	if got := firstStories[1]; got.StoryPoints != nil || got.Completed || got.CompletedAt != nil {
		t.Errorf("pending story = %+v, want null points and no completion", got)
	}
	secondStories := history.Sprints[1].Stories
	if len(secondStories) != 1 || secondStories[0].ID != carried || secondStories[0].Completed || secondStories[0].CompletedAt != nil {
		t.Errorf("second sprint stories = %+v, want the carried story not completed in this sprint despite its completed status", secondStories)
	}

	if again := get("/projects/" + projectID + "/sprints"); again.Code != http.StatusOK || again.Body.String() != response.Body.String() {
		t.Errorf("repeated GET differs:\nfirst: %s\nagain: %s", response.Body.String(), again.Body.String())
	}

	noClosed := get("/projects/" + historyOtherProjectID + "/sprints")
	if want := `{"project_id":"` + historyOtherProjectID + `","sprints":[]}`; noClosed.Code != http.StatusOK || strings.TrimSpace(noClosed.Body.String()) != want {
		t.Errorf("project without closed sprints status=%d body=%s, want %s", noClosed.Code, noClosed.Body.String(), want)
	}
	missing := get("/projects/" + historyMissingProjectID + "/sprints")
	if missing.Code != http.StatusNotFound || !strings.Contains(missing.Body.String(), `"error":"project_not_found"`) || strings.Contains(missing.Body.String(), "sprints") {
		t.Errorf("missing project status=%d body=%s", missing.Code, missing.Body.String())
	}
	invalid := get("/projects/not-a-uuid/sprints")
	if invalid.Code != http.StatusUnprocessableEntity || !strings.Contains(invalid.Body.String(), `"project_id"`) {
		t.Errorf("invalid project id status=%d body=%s", invalid.Code, invalid.Body.String())
	}
}
