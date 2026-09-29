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
		lists     bool
		logged    string
	}{
		{"version one", `DROP TABLE stories; ` + schemaTable + `INSERT INTO schema_migrations VALUES (1, false)`, http.StatusNotFound, false, false, "story creation unavailable"},
		{"version two", schemaTable + `INSERT INTO schema_migrations VALUES (2, false)`, http.StatusCreated, false, false, "story update unavailable"},
		{"version three", schemaTable + `INSERT INTO schema_migrations VALUES (3, false)`, http.StatusCreated, true, false, "story backlog unavailable"},
		{"version four", schemaTable + `INSERT INTO schema_migrations VALUES (4, false)`, http.StatusCreated, true, true, "story creation, update and backlog available"},
		{"dirty", schemaTable + `INSERT INTO schema_migrations VALUES (2, true)`, http.StatusNotFound, false, false, "story creation unavailable"},
		{"dirty version three", schemaTable + `INSERT INTO schema_migrations VALUES (3, true)`, http.StatusNotFound, false, false, "story creation unavailable"},
		{"dirty version four", schemaTable + `INSERT INTO schema_migrations VALUES (4, true)`, http.StatusNotFound, false, false, "story creation unavailable"},
		{"lookup error", `SELECT 1`, http.StatusNotFound, false, false, "story creation unavailable"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			pool := storyDatabase(t)
			if _, err := pool.Exec(context.Background(), scenario.migration); err != nil {
				t.Fatalf("prepare migration state: %v", err)
			}
			testAPIStartupRoutes(t, pool.Config().ConnString(), scenario.storyCode, scenario.updates, scenario.lists, scenario.logged)
		})
	}
}

func testAPIStartupRoutes(t *testing.T, databaseURL string, storyCode int, updates, lists bool, logged string) {
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
			assertBacklogRoute(t, client, "http://"+address+"/projects/"+project.ID+"/stories", project.ID, created.ID, storyCode == http.StatusCreated, lists)
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

// assertBacklogRoute checks GET on the collection route: the backlog when the query is composed, 405
// when only creation exists on that path, and the mux 404 when no story route exists at all. It runs
// after the update assertions, so a composed query must return the story as it was last modified.
func assertBacklogRoute(t *testing.T, client *http.Client, collectionURL, projectID, storyID string, creates, lists bool) {
	t.Helper()
	send := func(method, url string) (int, string) {
		request, err := http.NewRequest(method, url, nil)
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

	code, body := send(http.MethodGet, collectionURL)
	if !lists {
		want := http.StatusNotFound
		if creates {
			want = http.StatusMethodNotAllowed
		}
		if code != want || strings.Contains(body, `"stories"`) {
			t.Errorf("GET collection without the query = %d %q, want %d and no backlog", code, body, want)
		}
		return
	}
	var backlog struct {
		ProjectID string `json:"project_id"`
		Stories   []struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Status string `json:"status"`
		} `json:"stories"`
	}
	if code != http.StatusOK || json.Unmarshal([]byte(body), &backlog) != nil || backlog.ProjectID != projectID ||
		len(backlog.Stories) != 1 || backlog.Stories[0].ID != storyID ||
		backlog.Stories[0].Title != "Modificada" || backlog.Stories[0].Status != "completada" {
		t.Fatalf("GET collection = %d %s; want the modified story in the container of project %s", code, body, projectID)
	}
	if code, body = send(http.MethodHead, collectionURL); code != http.StatusOK || body != "" {
		t.Errorf("HEAD collection = %d %q, want 200 without a body", code, body)
	}
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		if code, _ = send(method, collectionURL); code != http.StatusMethodNotAllowed {
			t.Errorf("%s collection = %d, want 405", method, code)
		}
	}
	if code, _ = send(http.MethodGet, collectionURL+"/"); code != http.StatusNotFound {
		t.Errorf("GET collection with a trailing slash = %d, want 404", code)
	}
	unknown := strings.Replace(collectionURL, projectID, "9b6847be-c44c-47e8-a91d-69aa2874a80f", 1)
	if code, body = send(http.MethodGet, unknown); code != http.StatusNotFound ||
		!strings.Contains(body, `"error":"project_not_found"`) || strings.Contains(body, `"stories"`) {
		t.Errorf("GET unknown project = %d %s, want 404 project_not_found without stories", code, body)
	}
	invalid := strings.Replace(collectionURL, projectID, "not-a-uuid", 1)
	if code, body = send(http.MethodGet, invalid); code != http.StatusUnprocessableEntity ||
		!strings.Contains(body, `"error":"validation_failed"`) {
		t.Errorf("GET malformed project = %d %s, want 422 validation_failed", code, body)
	}
}

// TestBacklogHTTPOrdersByPriorityThenCreationEndToEnd runs the real handler over the real repository
// and follows the spec scenario S1..S5 through creation, modification and consultation.
func TestBacklogHTTPOrdersByPriorityThenCreationEndToEnd(t *testing.T) {
	const (
		otherProjectID = "c0a80121-7ac0-4e8e-8f3e-0a4c1b5d2f10"
		emptyProjectID = "1b4e28ba-2fa1-41d2-883f-0016d3cca427"
	)
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	insertProject(t, pool, otherProjectID)
	insertProject(t, pool, emptyProjectID)
	stories := storypostgres.NewPostgresStoryRepository(pool)
	handler := api.NewHTTPHandler(projectpostgres.NewPostgresProjectRepository(pool), api.NewProjectID,
		api.StoryDependencies{Repository: stories, GenerateID: api.NewProjectID, Updater: stories, Lister: stories})
	serve := func(method, path, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
		return response
	}
	create := func(project, title, priority string) string {
		response := serve(http.MethodPost, "/projects/"+project+"/stories",
			`{"title":"`+title+`","description":"Descripción","priority":"`+priority+`","acceptance_criteria":["Listo"]}`)
		var story struct {
			ID string `json:"id"`
		}
		if response.Code != http.StatusCreated || json.Unmarshal(response.Body.Bytes(), &story) != nil || story.ID == "" {
			t.Fatalf("create %s = %d; body = %s", title, response.Code, response.Body.String())
		}
		return story.ID
	}
	modify := func(project, id, title, priority, status string) {
		response := serve(http.MethodPut, "/projects/"+project+"/stories/"+id,
			`{"title":"`+title+`","description":"Descripción","priority":"`+priority+`","status":"`+status+`","acceptance_criteria":["Listo"],"estimated_hours":null}`)
		if response.Code != http.StatusOK {
			t.Fatalf("modify %s = %d; body = %s", title, response.Code, response.Body.String())
		}
	}
	backlog := func(project string) (string, []string) {
		response := serve(http.MethodGet, "/projects/"+project+"/stories", "")
		var body struct {
			ProjectID string `json:"project_id"`
			Stories   []struct {
				Title string `json:"title"`
			} `json:"stories"`
		}
		if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &body) != nil || body.ProjectID != project {
			t.Fatalf("GET backlog of %s = %d; body = %s", project, response.Code, response.Body.String())
		}
		titles := make([]string, 0, len(body.Stories))
		for _, story := range body.Stories {
			titles = append(titles, story.Title)
		}
		return response.Body.String(), titles
	}

	// Another project with stories must never leak into the queried backlog.
	create(otherProjectID, "Ajena", "alta")
	ids := map[string]string{}
	for _, story := range [][2]string{{"S1", "media"}, {"S2", "alta"}, {"S3", "media"}, {"S4", "baja"}, {"S5", "alta"}} {
		ids[story[0]] = create(projectID, story[0], story[1])
	}

	body, titles := backlog(projectID)
	if want := []string{"S2", "S5", "S1", "S3", "S4"}; !reflect.DeepEqual(titles, want) {
		t.Fatalf("initial backlog = %v, want %v", titles, want)
	}
	if again, _ := backlog(projectID); again != body {
		t.Errorf("two consecutive queries returned different bodies:\n%s\n%s", body, again)
	}

	modify(projectID, ids["S3"], "S3", "alta", "pendiente")
	if _, titles = backlog(projectID); !reflect.DeepEqual(titles, []string{"S2", "S3", "S5", "S1", "S4"}) {
		t.Errorf("backlog after raising S3 = %v, want S2, S3, S5, S1, S4", titles)
	}
	modify(projectID, ids["S1"], "S1 renombrada", "media", "completada")
	if _, titles = backlog(projectID); !reflect.DeepEqual(titles, []string{"S2", "S3", "S5", "S1 renombrada", "S4"}) {
		t.Errorf("backlog after renaming S1 = %v, want S1 to keep its position", titles)
	}
	if _, titles = backlog(otherProjectID); !reflect.DeepEqual(titles, []string{"Ajena"}) {
		t.Errorf("other project backlog = %v, want only its own story", titles)
	}

	if response := serve(http.MethodGet, "/projects/"+emptyProjectID+"/stories", ""); response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"stories":[]`) || strings.Contains(response.Body.String(), `"stories":null`) {
		t.Errorf("empty project = %d %s, want 200 with an empty array", response.Code, response.Body.String())
	}
	if response := serve(http.MethodGet, "/projects/9b6847be-c44c-47e8-a91d-69aa2874a80f/stories", ""); response.Code != http.StatusNotFound ||
		!strings.Contains(response.Body.String(), `"error":"project_not_found"`) || strings.Contains(response.Body.String(), `"stories"`) {
		t.Errorf("unknown project = %d %s, want 404 without stories", response.Code, response.Body.String())
	}
	if response := serve(http.MethodGet, "/projects/not-a-uuid/stories", ""); response.Code != http.StatusUnprocessableEntity ||
		!strings.Contains(response.Body.String(), `"error":"validation_failed"`) {
		t.Errorf("malformed project = %d %s, want 422", response.Code, response.Body.String())
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
