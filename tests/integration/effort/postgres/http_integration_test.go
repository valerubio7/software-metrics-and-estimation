package postgres_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	effortpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/effort/infrastructure/postgres"
	projectpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/project/infrastructure/postgres"
)

// effortHTTPNow is the injected clock: 2026-03-10 is "today" in UTC for every request below.
func effortHTTPNow() time.Time { return time.Date(2026, time.March, 10, 15, 0, 0, 0, time.UTC) }

// newEffortCompositionHandler composes the real HTTP handler over the fixture pool, exactly as
// cmd/api/main.go does once migration 000012 is clean.
func newEffortCompositionHandler(pool *pgxpool.Pool) http.Handler {
	return api.NewHTTPHandlerWithDependencies(
		projectpostgres.NewPostgresProjectRepository(pool),
		api.NewProjectID,
		api.HTTPDependencies{
			Effort: &api.EffortDependencies{
				Repository: effortpostgres.NewPostgresEffortRepository(pool),
				GenerateID: api.NewProjectID,
				Now:        effortHTTPNow,
			},
		},
	)
}

func postEffort(handler http.Handler, projectID, storyID, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/stories/"+storyID+"/effort-records", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

type effortResponseBody struct {
	ID          string  `json:"id"`
	ProjectID   string  `json:"project_id"`
	StoryID     string  `json:"story_id"`
	TaskID      *string `json:"task_id"`
	MemberID    string  `json:"member_id"`
	WorkDate    string  `json:"work_date"`
	Description string  `json:"description"`
	Hours       float64 `json:"hours"`
}

func decodeEffortResponse(t *testing.T, response *httptest.ResponseRecorder) effortResponseBody {
	t.Helper()
	var body effortResponseBody
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response %q: %v", response.Body.String(), err)
	}
	return body
}

func TestEffortHTTPEndToEnd(t *testing.T) {
	t.Run("effort over a story responds 201 and persists the same data", func(t *testing.T) {
		pool := seededEffortDatabase(t)
		handler := newEffortCompositionHandler(pool)

		response := postEffort(handler, effortProjectAID, effortStoryID,
			`{"member_id":"`+effortMemberID+`","work_date":"2026-03-10","description":"  Revision de codigo  ","hours":2.5}`)

		if response.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", response.Code, response.Body.String())
		}
		if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", contentType)
		}
		body := decodeEffortResponse(t, response)
		if body.ID == "" || body.ProjectID != effortProjectAID || body.StoryID != effortStoryID || body.TaskID != nil ||
			body.MemberID != effortMemberID || body.WorkDate != "2026-03-10" || body.Description != "Revision de codigo" || body.Hours != 2.5 {
			t.Fatalf("response = %+v, want the registered story effort", body)
		}
		if !strings.Contains(response.Body.String(), `"task_id":null`) {
			t.Errorf("body = %s, want task_id present and null", response.Body.String())
		}
		if strings.Contains(response.Body.String(), "created_at") || strings.Contains(response.Body.String(), `"seq"`) {
			t.Errorf("body = %s, must not expose created_at or seq", response.Body.String())
		}

		stored := storedEffortRecord(t, pool, body.ID)
		want := storedEffort{
			ProjectID: effortProjectAID, StoryID: effortStoryID, MemberID: effortMemberID,
			WorkDate: effortStoredWorkDate, Description: "Revision de codigo", Hours: "2.50", HasCreatedAt: true,
		}
		if stored != want {
			t.Errorf("stored record = %+v, want %+v", stored, want)
		}
		if got := effortRecordCount(t, pool); got != 1 {
			t.Errorf("effort_records rows = %d, want 1", got)
		}
	})

	t.Run("effort over a task responds 201 with the task and its story", func(t *testing.T) {
		pool := seededEffortDatabase(t)
		handler := newEffortCompositionHandler(pool)

		response := postEffort(handler, effortProjectAID, effortStoryID,
			`{"member_id":"`+effortMemberID+`","task_id":"`+strings.ToUpper(effortTaskID)+`","work_date":"2026-03-09","description":"Pruebas","hours":8}`)

		if response.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", response.Code, response.Body.String())
		}
		body := decodeEffortResponse(t, response)
		if body.TaskID == nil || *body.TaskID != effortTaskID || body.StoryID != effortStoryID || body.WorkDate != "2026-03-09" || body.Hours != 8 {
			t.Fatalf("response = %+v, want canonical lowercase task_id %s", body, effortTaskID)
		}
		stored := storedEffortRecord(t, pool, body.ID)
		if stored.TaskID == nil || *stored.TaskID != effortTaskID || stored.StoryID != effortStoryID || stored.Hours != "8.00" {
			t.Errorf("stored record = %+v, want the task and its story", stored)
		}
	})

	t.Run("effort over a story of a closed sprint and over a backlog story responds 201", func(t *testing.T) {
		pool := seededEffortDatabase(t)
		handler := newEffortCompositionHandler(pool)
		if _, err := pool.Exec(context.Background(), "UPDATE sprints SET is_closed = true WHERE id = $1", effortSprintID); err != nil {
			t.Fatalf("close sprint: %v", err)
		}
		body := `{"member_id":"` + effortMemberID + `","work_date":"2026-03-10","description":"Trabajo","hours":1}`

		for name, storyID := range map[string]string{"closed sprint": effortStoryID, "backlog": effortBacklogID} {
			if response := postEffort(handler, effortProjectAID, storyID, body); response.Code != http.StatusCreated {
				t.Errorf("%s: status = %d, want 201: %s", name, response.Code, response.Body.String())
			}
		}
		if got := effortRecordCount(t, pool); got != 2 {
			t.Errorf("effort_records rows = %d, want 2", got)
		}
	})

	t.Run("repeated identical requests create distinct records", func(t *testing.T) {
		pool := seededEffortDatabase(t)
		handler := newEffortCompositionHandler(pool)
		body := `{"member_id":"` + effortMemberID + `","work_date":"2026-03-10","description":"Igual","hours":1}`

		first := decodeEffortResponse(t, postEffort(handler, effortProjectAID, effortStoryID, body))
		second := decodeEffortResponse(t, postEffort(handler, effortProjectAID, effortStoryID, body))

		if first.ID == "" || first.ID == second.ID {
			t.Errorf("ids = %q and %q, want two distinct generated ids", first.ID, second.ID)
		}
		if got := effortRecordCount(t, pool); got != 2 {
			t.Errorf("effort_records rows = %d, want 2", got)
		}
	})

	rejections := []struct {
		name       string
		projectID  string
		storyID    string
		body       string
		wantStatus int
		wantInBody []string
	}{
		{
			name: "unknown member responds 404 member_not_found", projectID: effortProjectAID, storyID: effortStoryID,
			body:       `{"member_id":"` + effortUnknownID + `","work_date":"2026-03-10","description":"Trabajo","hours":1}`,
			wantStatus: http.StatusNotFound, wantInBody: []string{`"error":"member_not_found"`},
		},
		{
			name: "member of another project responds 404 member_not_found", projectID: effortProjectAID, storyID: effortStoryID,
			body:       `{"member_id":"` + effortMemberBID + `","work_date":"2026-03-10","description":"Trabajo","hours":1}`,
			wantStatus: http.StatusNotFound, wantInBody: []string{`"error":"member_not_found"`},
		},
		{
			name: "task of another story responds 404 task_not_found", projectID: effortProjectAID, storyID: effortStoryID,
			body:       `{"member_id":"` + effortMemberID + `","task_id":"` + effortTaskA2ID + `","work_date":"2026-03-10","description":"Trabajo","hours":1}`,
			wantStatus: http.StatusNotFound, wantInBody: []string{`"error":"task_not_found"`},
		},
		{
			name: "task of another project responds 404 task_not_found", projectID: effortProjectAID, storyID: effortStoryID,
			body:       `{"member_id":"` + effortMemberID + `","task_id":"` + effortTaskBID + `","work_date":"2026-03-10","description":"Trabajo","hours":1}`,
			wantStatus: http.StatusNotFound, wantInBody: []string{`"error":"task_not_found"`},
		},
		{
			name: "unknown story responds 404 story_not_found", projectID: effortProjectAID, storyID: effortUnknownID,
			body:       `{"member_id":"` + effortMemberID + `","work_date":"2026-03-10","description":"Trabajo","hours":1}`,
			wantStatus: http.StatusNotFound, wantInBody: []string{`"error":"story_not_found"`},
		},
		{
			name: "story of another project responds 404 story_not_found", projectID: effortProjectAID, storyID: effortStoryBID,
			body:       `{"member_id":"` + effortMemberID + `","work_date":"2026-03-10","description":"Trabajo","hours":1}`,
			wantStatus: http.StatusNotFound, wantInBody: []string{`"error":"story_not_found"`},
		},
		{
			name: "unknown project responds 404 project_not_found", projectID: effortUnknownID, storyID: effortStoryID,
			body:       `{"member_id":"` + effortMemberID + `","work_date":"2026-03-10","description":"Trabajo","hours":1}`,
			wantStatus: http.StatusNotFound, wantInBody: []string{`"error":"project_not_found"`},
		},
		{
			name: "several invalid fields respond 422 with every field", projectID: effortProjectAID, storyID: effortStoryID,
			body:       `{"member_id":"` + effortMemberID + `","work_date":"2026-03-11","description":"   ","hours":0}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantInBody: []string{`"error":"validation_failed"`, `"hours"`, `"description"`, `"work_date":"must not be in the future"`},
		},
		{
			name: "invalid fields win over a missing member", projectID: effortProjectAID, storyID: effortStoryID,
			body:       `{"member_id":"` + effortUnknownID + `","work_date":"2026-03-10","description":"Trabajo","hours":24.01}`,
			wantStatus: http.StatusUnprocessableEntity, wantInBody: []string{`"hours"`},
		},
		{
			name: "unknown key responds 400 invalid_request", projectID: effortProjectAID, storyID: effortStoryID,
			body:       `{"member_id":"` + effortMemberID + `","work_date":"2026-03-10","description":"Trabajo","hours":1,"status":"done"}`,
			wantStatus: http.StatusBadRequest, wantInBody: []string{`"error":"invalid_request"`},
		},
	}
	for _, test := range rejections {
		t.Run(test.name+" without writes", func(t *testing.T) {
			pool := seededEffortDatabase(t)
			handler := newEffortCompositionHandler(pool)

			response := postEffort(handler, test.projectID, test.storyID, test.body)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.wantStatus, response.Body.String())
			}
			for _, fragment := range test.wantInBody {
				if !strings.Contains(response.Body.String(), fragment) {
					t.Errorf("body = %s, want it to contain %s", response.Body.String(), fragment)
				}
			}
			if got := effortRecordCount(t, pool); got != 0 {
				t.Errorf("effort_records rows after a rejected request = %d, want 0", got)
			}
		})
	}
}

func TestEffortHTTPStorageFailureRespondsInternalErrorWithoutRows(t *testing.T) {
	pool := seededEffortDatabase(t)
	handler := newEffortCompositionHandler(pool)
	_, err := pool.Exec(context.Background(), `
		CREATE FUNCTION effort_records_force_failure() RETURNS trigger AS $$
		BEGIN
			RAISE EXCEPTION 'forced storage failure: secret internal detail';
		END;
		$$ LANGUAGE plpgsql;
		CREATE TRIGGER effort_records_force_failure_trigger BEFORE INSERT ON effort_records
			FOR EACH ROW EXECUTE FUNCTION effort_records_force_failure();
	`)
	if err != nil {
		t.Fatalf("create failing trigger: %v", err)
	}

	response := postEffort(handler, effortProjectAID, effortStoryID,
		`{"member_id":"`+effortMemberID+`","work_date":"2026-03-10","description":"Trabajo","hours":1}`)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"error":"internal_error"`) {
		t.Errorf("body = %s, want internal_error", response.Body.String())
	}
	if strings.Contains(response.Body.String(), "secret internal detail") || strings.Contains(response.Body.String(), "forced storage failure") {
		t.Errorf("body = %s, must not leak the storage error", response.Body.String())
	}
	if got := effortRecordCount(t, pool); got != 0 {
		t.Errorf("effort_records rows after a storage failure = %d, want 0", got)
	}
}

func TestEffortHTTPRouteUnavailableBelowVersion12(t *testing.T) {
	pool := effortDatabaseAtVersion(t, 11)
	seedEffortParents(t, pool)
	// Composition at schema version 11: the readiness flag is false, so Effort is not supplied.
	handler := api.NewHTTPHandlerWithDependencies(projectpostgres.NewPostgresProjectRepository(pool), api.NewProjectID, api.HTTPDependencies{})

	response := postEffort(handler, effortProjectAID, effortStoryID,
		`{"member_id":"`+effortMemberID+`","work_date":"2026-03-10","description":"Trabajo","hours":1}`)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 because the route is not registered: %s", response.Code, response.Body.String())
	}
	if relationExists(t, pool, "effort_records") {
		t.Errorf("effort_records exists at schema version 11, want it absent so nothing can be persisted")
	}
}
