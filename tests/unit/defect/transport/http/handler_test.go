package transporthttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/domain"
	defecthttp "github.com/valerubio7/software-metrics-and-estimation/internal/defect/transport/http"
)

const (
	projectID = "82d38423-f02d-4259-9e35-4a291585bb1e"
	storyID   = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	sprintID  = "b2a6a455-06e2-41ee-b011-7462c71375a0"
	route     = "/projects/82d38423-f02d-4259-9e35-4a291585bb1e/defects"
)

// stubRepository replaces persistence and lets each case force the sentinel outcome.
type stubRepository struct {
	err error
}

func (s *stubRepository) Create(_ context.Context, _ domain.Defect) error { return s.err }

// countingID hands out a deterministic progression of server-generated identifiers.
func countingID() application.IDGenerator {
	next := 0
	return func() string {
		next++
		return fmt.Sprintf("00000000-0000-0000-0000-%012d", next)
	}
}

// router registers the unit handler on a local ServeMux so PathValue works as in production.
func router(t *testing.T, repository *stubRepository) http.Handler {
	t.Helper()
	useCase := application.NewRegisterDefectUseCase(repository, countingID())
	mux := http.NewServeMux()
	mux.Handle("POST /projects/{project_id}/defects", defecthttp.NewRegisterDefectHandler(useCase))
	return mux
}

func defectBody(t *testing.T, extra string) string {
	t.Helper()
	base := `{"story_id": "` + storyID + `", "sprint_id": "` + sprintID + `", "description": "  El login acepta contraseñas vacías  ", "severity": "alta", "status": "detectado"`
	if extra != "" {
		return base + ", " + extra + "}"
	}
	return base + "}"
}

// serve routes the request through the local mux against the recorder.
func serve(t *testing.T, mux http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	request := httptest.NewRequest(method, path, reader)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

// assertErrorBody decodes one JSON error body and asserts its three standard keys.
func assertErrorBody(t *testing.T, raw []byte, wantError, wantMessage string, wantFields map[string]string) {
	t.Helper()
	var body struct {
		Error   string            `json:"error"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode error body %s: %v", raw, err)
	}
	if body.Error != wantError {
		t.Errorf("error = %q, want %q", body.Error, wantError)
	}
	if wantMessage != "" && body.Message != wantMessage {
		t.Errorf("message = %q, want %q", body.Message, wantMessage)
	}
	if wantFields == nil {
		if body.Fields != nil {
			t.Errorf("fields = %v, want absent", body.Fields)
		}
		return
	}
	if len(body.Fields) != len(wantFields) {
		t.Errorf("fields = %v, want %v", body.Fields, wantFields)
	}
	for field, message := range wantFields {
		if got := body.Fields[field]; got != message {
			t.Errorf("field %q = %q, want %q", field, got, message)
		}
	}
}

func TestRegisterDefectResponds201WithExactBody(t *testing.T) {
	mux := router(t, &stubRepository{})
	response := serve(t, mux, http.MethodPost, route, defectBody(t, ""))

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode 201 body: %v", err)
	}
	if body["id"] != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("id = %v, want the server-generated first identifier", body["id"])
	}
	if body["project_id"] != projectID || body["story_id"] != storyID || body["sprint_id"] != sprintID {
		t.Errorf("identifiers = %v", body)
	}
	if body["description"] != "El login acepta contraseñas vacías" {
		t.Errorf("description = %v, want trimmed", body["description"])
	}
	if body["severity"] != "alta" || body["status"] != "detectado" {
		t.Errorf("catalog values = severity %q status %q", body["severity"], body["status"])
	}
	for _, absent := range []string{"created_at", "seq"} {
		if _, ok := body[absent]; ok {
			t.Errorf("body exposes %v", absent)
		}
	}
	if len(body) != 7 {
		t.Errorf("body has %d keys (%v), want exactly the seven contract keys", len(body), body)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
}

func TestRegisterDefectRejectsMethod(t *testing.T) {
	// Como en effort: el handler se invoca directo (el mux de producción ya responde 405
	// para métodos no-POST); así se prueba el contrato JSON del propio handler.
	useCase := application.NewRegisterDefectUseCase(&stubRepository{}, countingID())
	handler := defecthttp.NewRegisterDefectHandler(useCase)
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			request := httptest.NewRequest(method, route, nil)
			request.SetPathValue("project_id", projectID)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d (body=%s), want 405", response.Code, response.Body.String())
			}
			assertErrorBody(t, response.Body.Bytes(), "method_not_allowed", "only POST is supported", nil)
		})
	}
}

func TestRegisterDefectMalformedBodies(t *testing.T) {
	cases := map[string]string{
		"empty_body":     "",
		"plain_text":     "defect",
		"broken_json":    `{"story_id": "`,
		"array_body":     `[]`,
		"null_body":      `null`,
		"extra_content":  defectBody(t, "") + ` trailing`,
		"unknown_key":    defectBody(t, `"priority": "alta"`),
		"id_injected":    defectBody(t, `"id": "3f5b7a1c-4c49-4a14-a1b1-58b9c3d2e1f0"`),
		"created_at_key": defectBody(t, `"created_at": "2026-10-25T10:00:00Z"`),
		"wrong_type":     `{"story_id": "` + storyID + `", "sprint_id": 17, "description": "Suma", "severity": "alta", "status": "detectado"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			mux := router(t, &stubRepository{})
			response := serve(t, mux, http.MethodPost, route, body)
			if response.Code != http.StatusBadRequest {
				t.Errorf("status = %d (body=%s), want 400", response.Code, response.Body.String())
			}
			assertErrorBody(t, response.Body.Bytes(), "invalid_request", "request body must be a single valid JSON object with allowed fields", nil)
		})
	}
}

func TestRegisterDefectRouteIDField(t *testing.T) {
	mux := router(t, &stubRepository{})
	response := serve(t, mux, http.MethodPost, "/projects/not-a-uuid/defects", defectBody(t, ""))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d (body=%s), want 422", response.Code, response.Body.String())
	}
	assertErrorBody(t, response.Body.Bytes(), "validation_failed", "one or more fields are invalid", map[string]string{
		"project_id": "must be a valid UUID",
	})
}

func TestRegisterDefectFieldErrors(t *testing.T) {
	cases := map[string]struct {
		body string
		want map[string]string
	}{
		"invalid_uuid": {body: `{"story_id": "not-a-uuid", "sprint_id": "` + sprintID + `", "description": "Suma", "severity": "alta", "status": "detectado"}`,
			want: map[string]string{"story_id": "must be a valid UUID"}},
		"missing_story": {body: `{"sprint_id": "` + sprintID + `", "description": "Suma", "severity": "alta", "status": "detectado"}`,
			want: map[string]string{"story_id": "is required"}},
		"empty_fields": {body: `{"story_id": "", "sprint_id": "", "description": "", "severity": "", "status": ""}`,
			want: map[string]string{"story_id": "is required", "sprint_id": "is required", "description": "is required", "severity": "is required", "status": "is required"}},
		"resuelto_status": {body: `{"story_id": "` + storyID + `", "sprint_id": "` + sprintID + `", "description": "Suma", "severity": "alta", "status": "resuelto"}`,
			want: map[string]string{"status": domain.InitialStatusMessage}},
		"accumulated": {body: `{"story_id": "nope", "sprint_id": "", "description": "   ", "severity": "grave", "status": "resuelto"}`,
			want: map[string]string{
				"story_id":    "must be a valid UUID",
				"sprint_id":   "is required",
				"description": "is required",
				"severity":    domain.SeverityMessage,
				"status":      domain.InitialStatusMessage,
			}},
		"nul_description": {body: `{"story_id": "` + storyID + `", "sprint_id": "` + sprintID + `", "description": "Suma\u0000", "severity": "alta", "status": "detectado"}`,
			want: map[string]string{"description": domain.NulMessage}},
		"too_long_description": {body: `{"story_id": "` + storyID + `", "sprint_id": "` + sprintID + `", "description": "` + strings.Repeat("a", 501) + `", "severity": "alta", "status": "detectado"}`,
			want: map[string]string{"description": domain.DescriptionLengthMessage}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			mux := router(t, &stubRepository{})
			response := serve(t, mux, http.MethodPost, route, tc.body)
			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d (body=%s), want 422", response.Code, response.Body.String())
			}
			assertErrorBody(t, response.Body.Bytes(), "validation_failed", "one or more fields are invalid", tc.want)
		})
	}
}

func TestRegisterDefectReferenceErrors(t *testing.T) {
	cases := map[string]struct {
		err     error
		status  int
		codeErr string
		message string
	}{
		"project_not_found": {err: application.ErrProjectNotFound, status: http.StatusNotFound, codeErr: "project_not_found", message: "project not found"},
		"story_not_found":   {err: application.ErrStoryNotFound, status: http.StatusNotFound, codeErr: "story_not_found", message: "story not found"},
		"sprint_not_found":  {err: application.ErrSprintNotFound, status: http.StatusNotFound, codeErr: "sprint_not_found", message: "sprint not found"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			mux := router(t, &stubRepository{err: tc.err})
			response := serve(t, mux, http.MethodPost, route, defectBody(t, ""))
			if response.Code != tc.status {
				t.Fatalf("status = %d (body=%s), want %d", response.Code, response.Body.String(), tc.status)
			}
			assertErrorBody(t, response.Body.Bytes(), tc.codeErr, tc.message, nil)
		})
	}
}

func TestRegisterDefectUnexpectedErrorIsGeneric500(t *testing.T) {
	mux := router(t, &stubRepository{err: errors.New("write to pointer bumped into a brick wall")})
	response := serve(t, mux, http.MethodPost, route, defectBody(t, ""))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	assertErrorBody(t, response.Body.Bytes(), "internal_error", "an unexpected error occurred", nil)
	if bytes.Contains(response.Body.Bytes(), []byte("brick wall")) {
		t.Error("500 body leaks internal details")
	}
}

func TestRegisterDefectErrorBodyShape(t *testing.T) {
	mux := router(t, &stubRepository{err: application.ErrStoryNotFound})
	response := serve(t, mux, http.MethodPost, route, defectBody(t, ""))

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	for _, key := range []string{"error", "message"} {
		if _, ok := body[key]; !ok {
			t.Errorf("error body missing %q: %v", key, body)
		}
	}
	if _, has := body["fields"]; has {
		t.Error("404 body must not carry fields")
	}
}
