package transporthttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/effort/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/effort/domain"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/effort/transport/http"
)

const (
	validProjectID = "82d38423-f02d-4259-9e35-4a291585bb1e"
	validStoryID   = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	validMemberID  = "5c21cbd4-7a0d-4d0a-9d5b-1b3c0a9f2e11"
	validTaskID    = "b2a6a455-06e2-41ee-b011-7462c71375a0"
	generatedID    = "0f8b7c1e-3d3a-4c58-9a57-2f0f4f5f6a77"

	routePattern    = "POST /projects/{project_id}/stories/{story_id}/effort-records"
	hoursMessage    = "must be greater than 0, at most 24 and have at most 2 decimals"
	secretDetail    = "secret db detail"
	invalidReqError = "invalid_request"
	validationError = "validation_failed"
)

// fixedNow es el instante del reloj inyectado: el 10 de marzo de 2026 en UTC.
var fixedNow = time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

// fakeEffortRepository reemplaza a application.EffortRepository: cuenta llamadas, captura el
// registro recibido y devuelve el error configurado.
type fakeEffortRepository struct {
	calls  int
	record domain.EffortRecord
	err    error
}

func (r *fakeEffortRepository) Create(_ context.Context, record domain.EffortRecord) error {
	r.calls++
	r.record = record
	return r.err
}

// newMux monta el handler real sobre un mux local con el patrón de la API para que PathValue funcione.
func newMux(repo *fakeEffortRepository) *http.ServeMux {
	useCase := application.NewRegisterEffortUseCase(
		repo,
		func() string { return generatedID },
		func() time.Time { return fixedNow },
	)
	mux := http.NewServeMux()
	mux.Handle(routePattern, transporthttp.NewRegisterEffortHandler(useCase))
	return mux
}

func post(handler http.Handler, projectID, storyID, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/stories/"+storyID+"/effort-records", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeObject(t *testing.T, response *httptest.ResponseRecorder) map[string]json.RawMessage {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &object); err != nil {
		t.Fatalf("decode response body %q: %v", response.Body.String(), err)
	}
	return object
}

func validBody(extra string) string {
	body := `{"member_id":"` + validMemberID + `","work_date":"2026-03-10","description":"Revisión de código","hours":2.5`
	if extra != "" {
		body += "," + extra
	}
	return body + "}"
}

func TestRegisterEffortHandler(t *testing.T) {
	for _, tc := range []struct {
		name       string
		body       string
		projectID  string
		storyID    string
		repoErr    error
		wantStatus int
		wantError  string
		wantFields map[string]string
		wantCalls  int
	}{
		{name: "success over story", body: validBody(""), wantStatus: http.StatusCreated, wantCalls: 1},
		{name: "success over task", body: validBody(`"task_id":"` + validTaskID + `"`), wantStatus: http.StatusCreated, wantCalls: 1},
		{name: "success with explicit null task", body: validBody(`"task_id":null`), wantStatus: http.StatusCreated, wantCalls: 1},

		{name: "malformed JSON", body: `{`, wantStatus: http.StatusBadRequest, wantError: invalidReqError},
		{name: "null body", body: `null`, wantStatus: http.StatusBadRequest, wantError: invalidReqError},
		{name: "array body", body: `[]`, wantStatus: http.StatusBadRequest, wantError: invalidReqError},
		{name: "two objects", body: validBody("") + validBody(""), wantStatus: http.StatusBadRequest, wantError: invalidReqError},
		{name: "trailing content", body: validBody("") + ` x`, wantStatus: http.StatusBadRequest, wantError: invalidReqError},
		{name: "unknown field status", body: validBody(`"status":"done"`), wantStatus: http.StatusBadRequest, wantError: invalidReqError},
		{name: "unknown field story_id", body: validBody(`"story_id":"` + validStoryID + `"`), wantStatus: http.StatusBadRequest, wantError: invalidReqError},
		{name: "unknown field id", body: validBody(`"id":"` + generatedID + `"`), wantStatus: http.StatusBadRequest, wantError: invalidReqError},
		{name: "unknown field created_at", body: validBody(`"created_at":"2026-03-10T00:00:00Z"`), wantStatus: http.StatusBadRequest, wantError: invalidReqError},
		{
			name:       "hours as string",
			body:       `{"member_id":"` + validMemberID + `","work_date":"2026-03-10","description":"A","hours":"2"}`,
			wantStatus: http.StatusBadRequest, wantError: invalidReqError,
		},
		{
			name:       "description as number",
			body:       `{"member_id":"` + validMemberID + `","work_date":"2026-03-10","description":7,"hours":2}`,
			wantStatus: http.StatusBadRequest, wantError: invalidReqError,
		},
		{
			name:       "work_date as number",
			body:       `{"member_id":"` + validMemberID + `","work_date":20260310,"description":"A","hours":2}`,
			wantStatus: http.StatusBadRequest, wantError: invalidReqError,
		},
		{
			name:       "member_id as number",
			body:       `{"member_id":7,"work_date":"2026-03-10","description":"A","hours":2}`,
			wantStatus: http.StatusBadRequest, wantError: invalidReqError,
		},
		{name: "task_id as number", body: validBody(`"task_id":7`), wantStatus: http.StatusBadRequest, wantError: invalidReqError},

		{
			name: "invalid project route", body: validBody(""), projectID: "not-a-uuid",
			wantStatus: http.StatusUnprocessableEntity, wantError: validationError,
			wantFields: map[string]string{"project_id": "must be a valid UUID"},
		},
		{
			name: "invalid story route", body: validBody(""), storyID: "not-a-uuid",
			wantStatus: http.StatusUnprocessableEntity, wantError: validationError,
			wantFields: map[string]string{"story_id": "must be a valid UUID"},
		},
		{
			name:       "several invalid body fields at once",
			body:       `{"member_id":"` + validMemberID + `","work_date":"2026-02-30","description":"   ","hours":0}`,
			wantStatus: http.StatusUnprocessableEntity, wantError: validationError,
			wantFields: map[string]string{
				"hours":       hoursMessage,
				"description": "is required",
				"work_date":   "must be a valid YYYY-MM-DD date",
			},
		},
		{
			name:       "validation wins over missing member",
			body:       `{"member_id":"` + validMemberID + `","work_date":"2026-03-10","description":"A","hours":0}`,
			repoErr:    application.ErrMemberNotFound,
			wantStatus: http.StatusUnprocessableEntity, wantError: validationError,
			wantFields: map[string]string{"hours": hoursMessage},
		},

		{name: "project not found", body: validBody(""), repoErr: application.ErrProjectNotFound, wantStatus: http.StatusNotFound, wantError: "project_not_found", wantCalls: 1},
		{name: "story not found", body: validBody(""), repoErr: application.ErrStoryNotFound, wantStatus: http.StatusNotFound, wantError: "story_not_found", wantCalls: 1},
		{name: "member not found", body: validBody(""), repoErr: application.ErrMemberNotFound, wantStatus: http.StatusNotFound, wantError: "member_not_found", wantCalls: 1},
		{
			name: "task not found", body: validBody(`"task_id":"` + validTaskID + `"`), repoErr: application.ErrTaskNotFound,
			wantStatus: http.StatusNotFound, wantError: "task_not_found", wantCalls: 1,
		},
		{
			name: "unexpected error", body: validBody(""), repoErr: errors.New(secretDetail),
			wantStatus: http.StatusInternalServerError, wantError: "internal_error", wantCalls: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeEffortRepository{err: tc.repoErr}
			projectID, storyID := validProjectID, validStoryID
			if tc.projectID != "" {
				projectID = tc.projectID
			}
			if tc.storyID != "" {
				storyID = tc.storyID
			}

			response := post(newMux(repo), projectID, storyID, tc.body)

			if response.Code != tc.wantStatus || repo.calls != tc.wantCalls {
				t.Fatalf("status/calls = %d/%d, want %d/%d: %s", response.Code, repo.calls, tc.wantStatus, tc.wantCalls, response.Body.String())
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", contentType)
			}
			if tc.wantError != "" {
				var body struct {
					Error   string            `json:"error"`
					Message string            `json:"message"`
					Fields  map[string]string `json:"fields"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatalf("decode error body %q: %v", response.Body.String(), err)
				}
				if body.Error != tc.wantError {
					t.Errorf("error = %q, want %q (body=%s)", body.Error, tc.wantError, response.Body.String())
				}
				if body.Message == "" {
					t.Errorf("message is empty (body=%s)", response.Body.String())
				}
				for field, message := range tc.wantFields {
					if body.Fields[field] != message {
						t.Errorf("fields[%q] = %q, want %q (body=%s)", field, body.Fields[field], message, response.Body.String())
					}
				}
				if len(body.Fields) != len(tc.wantFields) {
					t.Errorf("fields = %v, want exactly %v", body.Fields, tc.wantFields)
				}
			}
			if strings.Contains(response.Body.String(), secretDetail) {
				t.Error("internal error detail leaked in response body")
			}
		})
	}
}

func TestRegisterEffortHandlerErrorMessages(t *testing.T) {
	for _, tc := range []struct {
		name        string
		repoErr     error
		wantStatus  int
		wantMessage string
	}{
		{"project", application.ErrProjectNotFound, http.StatusNotFound, "project not found"},
		{"story", application.ErrStoryNotFound, http.StatusNotFound, "story not found"},
		{"member", application.ErrMemberNotFound, http.StatusNotFound, "member not found"},
		{"task", application.ErrTaskNotFound, http.StatusNotFound, "task not found"},
		{"unexpected", errors.New(secretDetail), http.StatusInternalServerError, "an unexpected error occurred"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := post(newMux(&fakeEffortRepository{err: tc.repoErr}), validProjectID, validStoryID, validBody(""))

			var body struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if response.Code != tc.wantStatus || body.Message != tc.wantMessage {
				t.Errorf("status/message = %d/%q, want %d/%q", response.Code, body.Message, tc.wantStatus, tc.wantMessage)
			}
		})
	}
}

func TestRegisterEffortHandlerSuccessBodyShape(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want map[string]string
	}{
		{
			name: "over story exposes task_id as null",
			body: validBody(""),
			want: map[string]string{
				"id": `"` + generatedID + `"`, "project_id": `"` + validProjectID + `"`, "story_id": `"` + validStoryID + `"`,
				"task_id": `null`, "member_id": `"` + validMemberID + `"`, "work_date": `"2026-03-10"`,
				"description": `"Revisión de código"`, "hours": `2.5`,
			},
		},
		{
			name: "over task exposes the task identifier",
			body: validBody(`"task_id":"` + validTaskID + `"`),
			want: map[string]string{
				"id": `"` + generatedID + `"`, "project_id": `"` + validProjectID + `"`, "story_id": `"` + validStoryID + `"`,
				"task_id": `"` + validTaskID + `"`, "member_id": `"` + validMemberID + `"`, "work_date": `"2026-03-10"`,
				"description": `"Revisión de código"`, "hours": `2.5`,
			},
		},
		{
			name: "description is returned trimmed",
			body: `{"member_id":"` + validMemberID + `","work_date":"2026-03-09","description":"  Revisión de código  ","hours":1}`,
			want: map[string]string{
				"id": `"` + generatedID + `"`, "project_id": `"` + validProjectID + `"`, "story_id": `"` + validStoryID + `"`,
				"task_id": `null`, "member_id": `"` + validMemberID + `"`, "work_date": `"2026-03-09"`,
				"description": `"Revisión de código"`, "hours": `1`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeEffortRepository{}
			response := post(newMux(repo), validProjectID, validStoryID, tc.body)

			if response.Code != http.StatusCreated || repo.calls != 1 {
				t.Fatalf("status/calls = %d/%d, want 201/1: %s", response.Code, repo.calls, response.Body.String())
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", contentType)
			}
			object := decodeObject(t, response)
			var gotKeys, wantKeys []string
			for key := range object {
				gotKeys = append(gotKeys, key)
			}
			for key := range tc.want {
				wantKeys = append(wantKeys, key)
			}
			sort.Strings(gotKeys)
			sort.Strings(wantKeys)
			if strings.Join(gotKeys, ",") != strings.Join(wantKeys, ",") {
				t.Fatalf("response keys = %v, want exactly %v (no created_at, no seq)", gotKeys, wantKeys)
			}
			for key, want := range tc.want {
				if got := string(object[key]); got != want {
					t.Errorf("%s = %s, want %s", key, got, want)
				}
			}
		})
	}
}

func TestRegisterEffortHandlerRejectsOtherMethods(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			repo := &fakeEffortRepository{}
			useCase := application.NewRegisterEffortUseCase(repo, func() string { return generatedID }, func() time.Time { return fixedNow })
			handler := transporthttp.NewRegisterEffortHandler(useCase)
			request := httptest.NewRequest(method, "/", strings.NewReader(validBody("")))
			request.SetPathValue("project_id", validProjectID)
			request.SetPathValue("story_id", validStoryID)

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusMethodNotAllowed || repo.calls != 0 {
				t.Fatalf("status/calls = %d/%d, want 405/0", response.Code, repo.calls)
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", contentType)
			}
			var body struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Error != "method_not_allowed" {
				t.Errorf("error = %q (decode err %v), want method_not_allowed", body.Error, err)
			}
		})
	}
}

func TestRegisterEffortHandlerEdgeCases(t *testing.T) {
	multibyte500 := strings.Repeat("ñ", 500)

	t.Run("hours are returned without losing precision", func(t *testing.T) {
		for _, hours := range []string{"0.01", "7.25", "24"} {
			body := `{"member_id":"` + validMemberID + `","work_date":"2026-01-05","description":"A","hours":` + hours + `}`
			response := post(newMux(&fakeEffortRepository{}), validProjectID, validStoryID, body)

			if response.Code != http.StatusCreated {
				t.Fatalf("hours %s: status = %d, want 201: %s", hours, response.Code, response.Body.String())
			}
			object := decodeObject(t, response)
			if got := string(object["hours"]); got != hours {
				t.Errorf("hours = %s, want %s", got, hours)
			}
			if got := string(object["work_date"]); got != `"2026-01-05"` {
				t.Errorf("work_date = %s, want \"2026-01-05\"", got)
			}
		}
	})

	t.Run("description of 500 multibyte characters is accepted and returned intact", func(t *testing.T) {
		body := `{"member_id":"` + validMemberID + `","work_date":"2026-03-10","description":"` + multibyte500 + `","hours":1}`
		response := post(newMux(&fakeEffortRepository{}), validProjectID, validStoryID, body)

		if response.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", response.Code, response.Body.String())
		}
		var got struct {
			Description string `json:"description"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil || got.Description != multibyte500 {
			t.Errorf("description length = %d (decode err %v), want 500 characters", len([]rune(got.Description)), err)
		}
	})

	t.Run("description of 501 characters is rejected", func(t *testing.T) {
		repo := &fakeEffortRepository{}
		body := `{"member_id":"` + validMemberID + `","work_date":"2026-03-10","description":"` + multibyte500 + `x","hours":1}`
		response := post(newMux(repo), validProjectID, validStoryID, body)

		if response.Code != http.StatusUnprocessableEntity || repo.calls != 0 {
			t.Fatalf("status/calls = %d/%d, want 422/0: %s", response.Code, repo.calls, response.Body.String())
		}
	})

	t.Run("description with a NUL character is rejected without calling the repository", func(t *testing.T) {
		// The JSON escape (not a raw NUL byte) is sent, so the body stays valid JSON.
		for name, description := range map[string]string{
			"in the middle": "a\\u0000b",
			"at the start":  "\\u0000abc",
			"at the end":    "abc\\u0000",
		} {
			repo := &fakeEffortRepository{}
			body := `{"member_id":"` + validMemberID + `","work_date":"2026-03-10","description":"` + description + `","hours":1}`
			response := post(newMux(repo), validProjectID, validStoryID, body)

			t.Run(name, func(t *testing.T) {
				assertValidationField(t, response, repo, "description", "must not contain NUL characters")
			})
		}
	})

	t.Run("uppercase task_id is returned in canonical lowercase", func(t *testing.T) {
		repo := &fakeEffortRepository{}
		body := validBody(`"task_id":"` + strings.ToUpper(validTaskID) + `"`)
		response := post(newMux(repo), validProjectID, validStoryID, body)

		if response.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", response.Code, response.Body.String())
		}
		if got := string(decodeObject(t, response)["task_id"]); got != `"`+validTaskID+`"` {
			t.Errorf("task_id = %s, want lowercase %q", got, validTaskID)
		}
		if repo.record.TaskID == nil || *repo.record.TaskID != validTaskID {
			t.Errorf("persisted task ID = %v, want %q", repo.record.TaskID, validTaskID)
		}
	})

	t.Run("uppercase route IDs are returned in canonical lowercase", func(t *testing.T) {
		response := post(newMux(&fakeEffortRepository{}), strings.ToUpper(validProjectID), strings.ToUpper(validStoryID), validBody(""))

		if response.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", response.Code, response.Body.String())
		}
		object := decodeObject(t, response)
		if string(object["project_id"]) != `"`+validProjectID+`"` || string(object["story_id"]) != `"`+validStoryID+`"` {
			t.Errorf("route IDs = %s/%s, want canonical lowercase", object["project_id"], object["story_id"])
		}
	})

	t.Run("missing member_id reports member_id", func(t *testing.T) {
		repo := &fakeEffortRepository{}
		response := post(newMux(repo), validProjectID, validStoryID,
			`{"work_date":"2026-03-10","description":"A","hours":1}`)

		assertValidationField(t, response, repo, "member_id", "is required")
	})

	t.Run("non-UUID task_id reports task_id", func(t *testing.T) {
		repo := &fakeEffortRepository{}
		response := post(newMux(repo), validProjectID, validStoryID, validBody(`"task_id":"abc"`))

		assertValidationField(t, response, repo, "task_id", "must be a valid UUID")
	})

	t.Run("invalid project and story routes report only project_id", func(t *testing.T) {
		repo := &fakeEffortRepository{}
		response := post(newMux(repo), "bad-project", "bad-story", validBody(""))

		assertValidationField(t, response, repo, "project_id", "must be a valid UUID")
		var body struct {
			Fields map[string]string `json:"fields"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || len(body.Fields) != 1 {
			t.Errorf("fields = %v (decode err %v), want only project_id", body.Fields, err)
		}
	})

	t.Run("body error wins over route error", func(t *testing.T) {
		repo := &fakeEffortRepository{}
		response := post(newMux(repo), "bad-project", validStoryID, `{`)

		if response.Code != http.StatusBadRequest || repo.calls != 0 {
			t.Fatalf("status/calls = %d/%d, want 400/0", response.Code, repo.calls)
		}
	})

	t.Run("future work_date is rejected against the injected clock", func(t *testing.T) {
		repo := &fakeEffortRepository{}
		response := post(newMux(repo), validProjectID, validStoryID,
			`{"member_id":"`+validMemberID+`","work_date":"2026-03-11","description":"A","hours":1}`)

		assertValidationField(t, response, repo, "work_date", "must not be in the future")
	})
}

// assertValidationField verifica un 422 sin escrituras que informa el campo con su mensaje.
func assertValidationField(t *testing.T, response *httptest.ResponseRecorder, repo *fakeEffortRepository, field, message string) {
	t.Helper()
	if response.Code != http.StatusUnprocessableEntity || repo.calls != 0 {
		t.Fatalf("status/calls = %d/%d, want 422/0: %s", response.Code, repo.calls, response.Body.String())
	}
	var body struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error != validationError || body.Fields[field] != message {
		t.Errorf("error/fields[%q] = %q/%q, want %q/%q (body=%s)", field, body.Error, body.Fields[field], validationError, message, response.Body.String())
	}
}
