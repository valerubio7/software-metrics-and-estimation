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
	"strings"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	projectpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/project/infrastructure/postgres"
	storypostgres "github.com/valerubio7/software-metrics-and-estimation/internal/story/infrastructure/postgres"
)

func TestProjectAPIStartsWithoutStoryMigration(t *testing.T) {
	pool := storyDatabase(t)
	// The disposable fixture applied both migrations; restore the version-one state.
	if _, err := pool.Exec(context.Background(), `DROP TABLE stories; CREATE TABLE schema_migrations (version bigint NOT NULL, dirty boolean NOT NULL); INSERT INTO schema_migrations VALUES (1, false)`); err != nil {
		t.Fatalf("prepare version-one database: %v", err)
	}
	// Use the live pool's connection configuration for the same disposable container.
	databaseURL := pool.Config().ConnString()
	binary := filepath.Join(t.TempDir(), "api")
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
			response.Body.Close()
			if response.StatusCode != http.StatusCreated {
				t.Fatalf("project creation status = %d, want 201", response.StatusCode)
			}
			story, err := client.Post("http://"+address+"/projects/"+projectID+"/stories", "application/json", strings.NewReader(`{"title":"Registro","description":"Crear historia","priority":"media","acceptance_criteria":["Listo"]}`))
			if err != nil {
				t.Fatal(err)
			}
			story.Body.Close()
			if story.StatusCode != http.StatusNotFound {
				t.Errorf("version-one story status = %d, want 404", story.StatusCode)
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("API did not serve projects with migration 000001 only")
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
