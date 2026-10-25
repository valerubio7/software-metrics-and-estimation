package postgres_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	defectpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/defect/infrastructure/postgres"
	projectpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/project/infrastructure/postgres"
)

// newDefectCompositionHandler composes the real HTTP handler over the fixture pool, exactly as
// cmd/api/main.go does once migration 000013 is clean.
func newDefectCompositionHandler(pool *pgxpool.Pool) http.Handler {
	return api.NewHTTPHandlerWithDependencies(
		projectpostgres.NewPostgresProjectRepository(pool),
		api.NewProjectID,
		api.HTTPDependencies{
			Defect: &api.DefectDependencies{
				Repository: defectpostgres.NewPostgresDefectRepository(pool),
				GenerateID: api.NewProjectID,
			},
		},
	)
}

func postDefect(handler http.Handler, projectID, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/defects", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

type defectResponseBody struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	StoryID     string `json:"story_id"`
	SprintID    string `json:"sprint_id"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Status      string `json:"status"`
}

func decodeDefectResponse(t *testing.T, response *httptest.ResponseRecorder) defectResponseBody {
	t.Helper()
	var body defectResponseBody
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response %q: %v", response.Body.String(), err)
	}
	return body
}

func validDefectHTTPBody(storyID, sprintID string) string {
	return `{"story_id":"` + storyID + `","sprint_id":"` + sprintID + `","description":"  El login acepta contraseñas vacías  ","severity":"alta","status":"detectado"}`
}

func TestDefectHTTPEndToEnd(t *testing.T) {
	t.Run("a valid defect responds 201 and persists the same data", func(t *testing.T) {
		pool := seededDefectDatabase(t)
		handler := newDefectCompositionHandler(pool)

		response := postDefect(handler, defectProjectAID, validDefectHTTPBody(defectStoryID, defectSprintID))

		if response.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", response.Code, response.Body.String())
		}
		if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", contentType)
		}
		body := decodeDefectResponse(t, response)
		if body.ID == "" || body.ProjectID != defectProjectAID || body.StoryID != defectStoryID ||
			body.SprintID != defectSprintID || body.Description != "El login acepta contraseñas vacías" ||
			body.Severity != "alta" || body.Status != "detectado" {
			t.Fatalf("response = %+v, want the registered defect", body)
		}
		if strings.Contains(response.Body.String(), "created_at") || strings.Contains(response.Body.String(), `"seq"`) {
			t.Errorf("body = %s, must not expose created_at or seq", response.Body.String())
		}

		var description, severity, status string
		if err := pool.QueryRow(context.Background(), `
			SELECT description, severity, status FROM defects WHERE id = $1
		`, body.ID).Scan(&description, &severity, &status); err != nil {
			t.Fatalf("read stored defect: %v", err)
		}
		if description != "El login acepta contraseñas vacías" || severity != "alta" || status != "detectado" {
			t.Errorf("stored defect = (%q, %q, %q), want the registered values", description, severity, status)
		}
		if got := defectCount(t, pool); got != 1 {
			t.Errorf("defects rows = %d, want 1", got)
		}
	})

	t.Run("exact duplicates both respond 201 and persist distinct rows", func(t *testing.T) {
		pool := seededDefectDatabase(t)
		handler := newDefectCompositionHandler(pool)

		first := postDefect(handler, defectProjectAID, validDefectHTTPBody(defectStoryID, defectSprintID))
		second := postDefect(handler, defectProjectAID, validDefectHTTPBody(defectStoryID, defectSprintID))
		if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
			t.Fatalf("statuses = %d/%d, want 201/201", first.Code, second.Code)
		}
		firstBody := decodeDefectResponse(t, first)
		secondBody := decodeDefectResponse(t, second)
		if firstBody.ID == "" || firstBody.ID == secondBody.ID {
			t.Errorf("duplicate identifiers = (%q, %q), want distinct server-generated values", firstBody.ID, secondBody.ID)
		}
		if got := defectCount(t, pool); got != 2 {
			t.Errorf("defects rows = %d, want 2", got)
		}
	})

	t.Run("reference failures answer 404 in precedence order without writes", func(t *testing.T) {
		pool := seededDefectDatabase(t)
		handler := newDefectCompositionHandler(pool)

		cases := []struct {
			projectID string
			storyID   string
			sprintID  string
			wantCode  int
			wantError string
		}{
			{projectID: unknownProjectID, storyID: defectStoryID, sprintID: defectSprintID, wantCode: http.StatusNotFound, wantError: "project_not_found"},
			{projectID: defectProjectAID, storyID: defectStoryBID, sprintID: defectSprintBID, wantCode: http.StatusNotFound, wantError: "story_not_found"},
			{projectID: defectProjectAID, storyID: defectStoryID, sprintID: defectSprintBID, wantCode: http.StatusNotFound, wantError: "sprint_not_found"},
		}
		for _, tc := range cases {
			response := postDefect(handler, tc.projectID, validDefectHTTPBody(tc.storyID, tc.sprintID))
			if response.Code != tc.wantCode {
				t.Errorf("status = %d (body=%s), want %d", response.Code, response.Body.String(), tc.wantCode)
				continue
			}
			var body struct {
				Error   string `json:"error"`
				Message string `json:"message"`
				Fields  map[string]string
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode 404 body: %v", err)
			}
			if body.Error != tc.wantError || body.Fields != nil {
				t.Errorf("404 body = %s, want %q without fields", response.Body.String(), tc.wantError)
			}
		}
		if got := defectCount(t, pool); got != 0 {
			t.Errorf("defects rows after 404s = %d, want 0", got)
		}
	})

	t.Run("GET on the route answers 405 without writes", func(t *testing.T) {
		pool := seededDefectDatabase(t)
		handler := newDefectCompositionHandler(pool)

		request := httptest.NewRequest(http.MethodGet, "/projects/"+defectProjectAID+"/defects", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusMethodNotAllowed || defectCount(t, pool) != 0 {
			t.Fatalf("GET status/rows = %d/%d, want 405/0", response.Code, defectCount(t, pool))
		}
	})
}
