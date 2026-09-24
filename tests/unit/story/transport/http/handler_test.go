package transporthttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/story/transport/http"
)

const (
	projectID = "5C21CBD4-D9A7-42DF-9C3A-C0866F058746"
	storyID   = "25c21cbd-d9a7-42df-9c3a-c0866f058746"
	validBody = `{"title":" Registro ","description":"Descripción","priority":"media","acceptance_criteria":["Primero"," Segundo "]}`
)

type fakeRepository struct {
	calls int
	story domain.Story
	err   error
}

func (r *fakeRepository) Create(_ context.Context, story domain.Story) error {
	r.calls++
	r.story = story
	return r.err
}

func request(handler http.Handler, id, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/projects/"+id+"/stories", strings.NewReader(body))
	req.SetPathValue("project_id", id)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func handler(repo *fakeRepository) http.Handler {
	return transporthttp.NewCreateStoryHandler(application.NewCreateStoryUseCase(repo, func() string { return storyID }))
}

func decode(t *testing.T, response *httptest.ResponseRecorder) map[string]json.RawMessage {
	t.Helper()
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q", got)
	}
	var result map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid response JSON: %v", err)
	}
	return result
}

func assertError(t *testing.T, response *httptest.ResponseRecorder, status int, code, message string) map[string]json.RawMessage {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body.String())
	}
	body := decode(t, response)
	if string(body["error"]) != `"`+code+`"` || string(body["message"]) != `"`+message+`"` {
		t.Errorf("error = %s, message = %s", body["error"], body["message"])
	}
	return body
}

func TestCreateStoryReturnsPendingUnestimatedStory(t *testing.T) {
	repo := &fakeRepository{}
	response := request(handler(repo), projectID, validBody)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body.String())
	}
	body := decode(t, response)
	for key, want := range map[string]string{
		"id": `"` + storyID + `"`, "project_id": `"5c21cbd4-d9a7-42df-9c3a-c0866f058746"`,
		"title": `" Registro "`, "description": `"Descripción"`, "priority": `"media"`,
		"status": `"pendiente"`, "story_points": "null",
	} {
		if string(body[key]) != want {
			t.Errorf("%s = %s, want %s", key, body[key], want)
		}
	}
	var criteria []string
	if err := json.Unmarshal(body["acceptance_criteria"], &criteria); err != nil || !reflect.DeepEqual(criteria, []string{"Primero", " Segundo "}) {
		t.Errorf("criteria = %v, error = %v", criteria, err)
	}
	if repo.calls != 1 || repo.story.ProjectID != "5c21cbd4-d9a7-42df-9c3a-c0866f058746" || repo.story.StoryPoints != nil {
		t.Errorf("persisted calls/story = %d/%+v", repo.calls, repo.story)
	}
}

func TestCreateStoryRejectsInvalidJSONShape(t *testing.T) {
	cases := map[string]string{
		"empty": "", "malformed": `{"title":`, "multiple": validBody + validBody,
		"null": "null", "array": `[]`, "string": `"hello"`,
		"wrong title type": `{"title":7}`, "wrong criteria type": `{"acceptance_criteria":[7]}`,
		"null criterion is not a string": `{"title":"Hi","description":"Description","priority":"baja","acceptance_criteria":["ok",null]}`,
		"unknown": `{"other":null}`, "id": `{"id":null}`, "project_id": `{"project_id":null}`,
		"status": `{"status":null}`, "story_points null": `{"story_points":null}`,
		"story_points zero": `{"story_points":0}`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &fakeRepository{}
			response := request(handler(repo), projectID, input)
			assertError(t, response, http.StatusBadRequest, "invalid_request", "request body must be a single valid JSON object with allowed fields")
			if repo.calls != 0 {
				t.Errorf("writes = %d, want zero", repo.calls)
			}
		})
	}
}

func TestCreateStoryRejectsInvalidFieldsBeforeWriting(t *testing.T) {
	cases := []struct{ name, id, input, field string }{
		{"invalid uuid", "not-a-uuid", validBody, "project_id"},
		{"missing title", projectID, `{"description":"Description","priority":"alta","acceptance_criteria":["ok"]}`, "title"},
		{"blank description", projectID, `{"title":"Hi","description":"  ","priority":"alta","acceptance_criteria":["ok"]}`, "description"},
		{"invalid priority", projectID, `{"title":"Hi","description":"Description","priority":"ALTA","acceptance_criteria":["ok"]}`, "priority"},
		{"missing criteria", projectID, `{"title":"Hi","description":"Description","priority":"baja"}`, "acceptance_criteria"},
		{"null criteria", projectID, `{"title":"Hi","description":"Description","priority":"baja","acceptance_criteria":null}`, "acceptance_criteria"},
		{"empty criteria", projectID, `{"title":"Hi","description":"Description","priority":"baja","acceptance_criteria":[]}`, "acceptance_criteria"},
		{"blank criterion", projectID, `{"title":"Hi","description":"Description","priority":"baja","acceptance_criteria":["ok","  "]}`, "acceptance_criteria"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			response := request(handler(repo), tc.id, tc.input)
			body := assertError(t, response, http.StatusUnprocessableEntity, "validation_failed", "one or more fields are invalid")
			var fields map[string]string
			if err := json.Unmarshal(body["fields"], &fields); err != nil || fields[tc.field] == "" {
				t.Errorf("fields = %v, error = %v; want %s", fields, err, tc.field)
			}
			if repo.calls != 0 {
				t.Errorf("writes = %d, want zero", repo.calls)
			}
		})
	}
}

func TestCreateStoryMapsOnlyTypedMissingProjectTo404(t *testing.T) {
	for _, tc := range []struct {
		name          string
		err           error
		status        int
		code, message string
	}{
		{"missing", application.ErrProjectNotFound, 404, "project_not_found", "project not found"},
		{"unexpected", errors.New("secret database password"), 500, "internal_error", "an unexpected error occurred"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{err: tc.err}
			response := request(handler(repo), projectID, validBody)
			assertError(t, response, tc.status, tc.code, tc.message)
			if bytes.Contains(response.Body.Bytes(), []byte("secret")) || repo.calls != 1 {
				t.Errorf("unsafe response or write count: %s, %d", response.Body.String(), repo.calls)
			}
		})
	}
}
