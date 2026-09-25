package postgres_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	projectpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/project/infrastructure/postgres"
	storypostgres "github.com/valerubio7/software-metrics-and-estimation/internal/story/infrastructure/postgres"
)

func TestAPIStartupRoutesFollowMigrationState(t *testing.T) {
	const schemaTable = `CREATE TABLE schema_migrations (version bigint NOT NULL, dirty boolean NOT NULL); `
	for _, scenario := range []struct {
		name      string
		migration string
		storyCode int
		updates   bool
		logged    string
	}{
		{"version one", `DROP TABLE stories; ` + schemaTable + `INSERT INTO schema_migrations VALUES (1, false)`, http.StatusNotFound, false, "story creation unavailable"},
		{"version two", schemaTable + `INSERT INTO schema_migrations VALUES (2, false)`, http.StatusCreated, false, "story update unavailable"},
		{"version three", schemaTable + `INSERT INTO schema_migrations VALUES (3, false)`, http.StatusCreated, true, "story creation and update available"},
		{"dirty", schemaTable + `INSERT INTO schema_migrations VALUES (2, true)`, http.StatusNotFound, false, "story creation unavailable"},
		{"dirty version three", schemaTable + `INSERT INTO schema_migrations VALUES (3, true)`, http.StatusNotFound, false, "story creation unavailable"},
		{"lookup error", `SELECT 1`, http.StatusNotFound, false, "story creation unavailable"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			pool := storyDatabase(t)
			if _, err := pool.Exec(context.Background(), scenario.migration); err != nil {
				t.Fatalf("prepare migration state: %v", err)
			}
			testAPIStartupRoutes(t, pool.Config().ConnString(), scenario.storyCode, scenario.updates, scenario.logged)
		})
	}
}

func testAPIStartupRoutes(t *testing.T, databaseURL string, storyCode int, updates bool, logged string) {
	t.Helper()
	name := "api"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	build := exec.Command("go", "build", "-o", binary, "./cmd/api")
	build.Dir = storyModuleRoot(t)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build API: %v: %s", err, output)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	server := exec.CommandContext(ctx, binary)
	server.Env = append(os.Environ(), "DATABASE_URL="+databaseURL, "HTTP_ADDR="+address)
	var stderr bytes.Buffer
	server.Stderr = &stderr
	if err := server.Start(); err != nil {
		t.Fatalf("start API: %v", err)
	}
	finished := make(chan error, 1)
	go func() { finished <- server.Wait() }()
	defer func() {
		if server.Process != nil {
			_ = server.Process.Kill()
		}
		select {
		case <-finished:
		default:
		}
	}()

	client := &http.Client{Timeout: time.Second}
	for ctx.Err() == nil {
		select {
		case err := <-finished:
			t.Fatalf("API exited before serving US-01: %v; stderr: %s", err, stderr.String())
		default:
		}
		response, err := client.Post("http://"+address+"/projects", "application/json", strings.NewReader(`{"name":"Metrics portal","start_date":"2026-03-01","planned_finish_date":"2026-06-30"}`))
		if err == nil {
			if response.StatusCode != http.StatusCreated {
				response.Body.Close()
				t.Fatalf("project creation status = %d, want 201", response.StatusCode)
			}
			var project struct {
				ID string `json:"id"`
			}
			if err := json.NewDecoder(response.Body).Decode(&project); err != nil {
				response.Body.Close()
				t.Fatal(err)
			}
			response.Body.Close()
			story, err := client.Post("http://"+address+"/projects/"+project.ID+"/stories", "application/json", strings.NewReader(`{"title":"Registro","description":"Crear historia","priority":"media","acceptance_criteria":["Listo"]}`))
			if err != nil {
				t.Fatal(err)
			}
			var created struct {
				ID string `json:"id"`
			}
			if storyCode == http.StatusCreated {
				if err := json.NewDecoder(story.Body).Decode(&created); err != nil {
					story.Body.Close()
					t.Fatal(err)
				}
			}
			story.Body.Close()
			if story.StatusCode != storyCode {
				t.Errorf("story status = %d, want %d", story.StatusCode, storyCode)
			}
			if !strings.Contains(stderr.String(), logged) {
				t.Errorf("startup log = %q, want it to contain %q", stderr.String(), logged)
			}
			if created.ID == "" {
				created.ID = "9b6847be-c44c-47e8-a91d-69aa2874a80f"
			}
			assertStoryUpdateRoute(t, client, "http://"+address+"/projects/"+project.ID+"/stories/", project.ID, created.ID, updates)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("API did not serve projects for migration state")
}

func assertStoryUpdateRoute(t *testing.T, client *http.Client, storiesURL, projectID, storyID string, updates bool) {
	t.Helper()
	put := func(id, status string) (int, string) {
		body := `{"title":"Modificada","description":"Otra descripción","priority":"alta","status":"` + status + `","acceptance_criteria":["Uno","Dos"],"estimated_hours":2.5}`
		request, err := http.NewRequest(http.MethodPut, storiesURL+id, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var payload bytes.Buffer
		if _, err := payload.ReadFrom(response.Body); err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, payload.String()
	}

	code, body := put(storyID, "en_progreso")
	if !updates {
		if code != http.StatusNotFound || strings.Contains(body, "story_not_found") {
			t.Errorf("PUT without the route = %d %q, want the mux 404", code, body)
		}
		return
	}
	var updated struct {
		ID             string   `json:"id"`
		ProjectID      string   `json:"project_id"`
		Status         string   `json:"status"`
		StoryPoints    *int     `json:"story_points"`
		Criteria       []string `json:"acceptance_criteria"`
		EstimatedHours *float64 `json:"estimated_hours"`
	}
	if code != http.StatusOK || json.Unmarshal([]byte(body), &updated) != nil || updated.ID != storyID ||
		updated.ProjectID != projectID || updated.Status != "en_progreso" || updated.StoryPoints != nil ||
		updated.EstimatedHours == nil || *updated.EstimatedHours != 2.5 || !reflect.DeepEqual(updated.Criteria, []string{"Uno", "Dos"}) {
		t.Fatalf("PUT = %d %s; parsed %+v", code, body, updated)
	}
	if code, body = put(storyID, "completada"); code != http.StatusOK || !strings.Contains(body, `"status":"completada"`) {
		t.Errorf("second PUT = %d %s, want the last write to win", code, body)
	}
	if code, body = put("9b6847be-c44c-47e8-a91d-69aa2874a80f", "pendiente"); code != http.StatusNotFound || !strings.Contains(body, `"error":"story_not_found"`) {
		t.Errorf("PUT unknown story = %d %s, want 404 story_not_found", code, body)
	}
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete} {
		request, err := http.NewRequest(method, storiesURL+storyID, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("%s on the item route = %d, want 405", method, response.StatusCode)
		}
	}
}

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

func TestStoryUpdateHTTPWithMigratedPostgres(t *testing.T) {
	const otherProjectID = "c0a80121-7ac0-4e8e-8f3e-0a4c1b5d2f10"
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	insertProject(t, pool, otherProjectID)
	stories := storypostgres.NewPostgresStoryRepository(pool)
	handler := api.NewHTTPHandler(projectpostgres.NewPostgresProjectRepository(pool), api.NewProjectID,
		api.StoryDependencies{Repository: stories, GenerateID: api.NewProjectID, Updater: stories})
	serve := func(method, path, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
		return response
	}

	created := serve(http.MethodPost, "/projects/"+projectID+"/stories",
		`{"title":"Registro","description":"Crear historia","priority":"media","acceptance_criteria":["Primero"]}`)
	var story struct {
		ID string `json:"id"`
	}
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &story) != nil || story.ID == "" {
		t.Fatalf("create = %d; body = %s", created.Code, created.Body.String())
	}
	if _, err := pool.Exec(context.Background(), `UPDATE stories SET story_points = 8 WHERE id = $1`, story.ID); err != nil {
		t.Fatalf("seed story points: %v", err)
	}

	itemPath := "/projects/" + projectID + "/stories/" + story.ID
	body := `{"title":"Modificada","description":"Otra descripción","priority":"alta","status":"completada","acceptance_criteria":["Segundo","Primero"],"estimated_hours":99999.99}`
	updated := serve(http.MethodPut, itemPath, body)
	var result struct {
		ID             string   `json:"id"`
		ProjectID      string   `json:"project_id"`
		Status         string   `json:"status"`
		StoryPoints    *int     `json:"story_points"`
		Criteria       []string `json:"acceptance_criteria"`
		EstimatedHours *float64 `json:"estimated_hours"`
	}
	if updated.Code != http.StatusOK || json.Unmarshal(updated.Body.Bytes(), &result) != nil ||
		result.ID != story.ID || result.ProjectID != projectID || result.Status != "completada" ||
		result.StoryPoints == nil || *result.StoryPoints != 8 || result.EstimatedHours == nil ||
		*result.EstimatedHours != 99999.99 || !reflect.DeepEqual(result.Criteria, []string{"Segundo", "Primero"}) {
		t.Fatalf("update = %d; body = %s", updated.Code, updated.Body.String())
	}
	var storedProject, storedTitle string
	if err := pool.QueryRow(context.Background(), `SELECT project_id::text, title FROM stories WHERE id = $1`, story.ID).
		Scan(&storedProject, &storedTitle); err != nil || storedProject != projectID || storedTitle != "Modificada" {
		t.Errorf("stored story = (%q, %q), err = %v; want project %q and the new title", storedProject, storedTitle, err, projectID)
	}

	for name, path := range map[string]string{
		"unknown story":          "/projects/" + projectID + "/stories/9b6847be-c44c-47e8-a91d-69aa2874a80f",
		"story of other project": "/projects/" + otherProjectID + "/stories/" + story.ID,
	} {
		missing := serve(http.MethodPut, path, body)
		if missing.Code != http.StatusNotFound || !strings.Contains(missing.Body.String(), `"error":"story_not_found"`) {
			t.Errorf("%s = %d; body = %s", name, missing.Code, missing.Body.String())
		}
	}
	if err := pool.QueryRow(context.Background(), `SELECT project_id::text, title FROM stories WHERE id = $1`, story.ID).
		Scan(&storedProject, &storedTitle); err != nil || storedProject != projectID || storedTitle != "Modificada" {
		t.Errorf("story after 404s = (%q, %q), err = %v; want it untouched", storedProject, storedTitle, err)
	}
}
