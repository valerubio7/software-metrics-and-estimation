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
	otherProjectID = "0f8fad5b-d9cb-469f-a165-70867728950e"
	validUpdate    = `{"title":" Registro ","description":"Descripción","priority":"media","status":"en_progreso","acceptance_criteria":["Primero"," Segundo "],"estimated_hours":8.5}`
	invalidRequest = "request body must be a single valid JSON object with allowed fields"
)

// fakeUpdater records writes and returns the configured stored story, or the story it received.
type fakeUpdater struct {
	calls  int
	story  domain.Story
	stored *domain.Story
	err    error
}

func (u *fakeUpdater) Update(_ context.Context, story domain.Story) (domain.Story, error) {
	u.calls++
	u.story = story
	if u.err != nil {
		return domain.Story{}, u.err
	}
	if u.stored != nil {
		return *u.stored, nil
	}
	return story, nil
}

func updateHandler(updater *fakeUpdater) http.Handler {
	return transporthttp.NewUpdateStoryHandler(application.NewUpdateStoryUseCase(updater))
}

func updateRequest(handler http.Handler, method, project, story, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/projects/"+project+"/stories/"+story, strings.NewReader(body))
	req.SetPathValue("project_id", project)
	req.SetPathValue("story_id", story)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func put(updater *fakeUpdater, project, story, body string) *httptest.ResponseRecorder {
	return updateRequest(updateHandler(updater), http.MethodPut, project, story, body)
}

func decodeFields(t *testing.T, body map[string]json.RawMessage) map[string]string {
	t.Helper()
	var fields map[string]string
	if err := json.Unmarshal(body["fields"], &fields); err != nil {
		t.Fatalf("fields = %s, error = %v", body["fields"], err)
	}
	return fields
}

func TestUpdateStoryReturnsTheFullStoredStory(t *testing.T) {
	points, hours := 5, 8.5
	updater := &fakeUpdater{stored: &domain.Story{
		ID: storyID, ProjectID: projectID, Title: "Stored", Description: "Stored description",
		Priority: "alta", Status: domain.StatusCompleted, StoryPoints: &points,
		AcceptanceCriteria: []string{"Stored criterion"}, EstimatedHours: &hours,
	}}
	response := put(updater, projectID, storyID, validUpdate)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body.String())
	}
	body := decode(t, response)
	if len(body) != 9 {
		t.Errorf("response has %d keys, want 9: %s", len(body), response.Body.String())
	}
	for key, want := range map[string]string{
		"id": `"` + storyID + `"`, "project_id": `"` + projectID + `"`, "title": `"Stored"`,
		"description": `"Stored description"`, "priority": `"alta"`, "status": `"completada"`,
		"story_points": "5", "acceptance_criteria": `["Stored criterion"]`, "estimated_hours": "8.5",
	} {
		if _, present := body[key]; !present || string(body[key]) != want {
			t.Errorf("%s = %s (present = %v), want %s", key, body[key], present, want)
		}
	}
	if updater.calls != 1 {
		t.Errorf("writes = %d, want one", updater.calls)
	}
}

func TestUpdateStoryPassesCanonicalIdentifiersAndAllSixFieldsToTheUpdater(t *testing.T) {
	updater := &fakeUpdater{}
	response := put(updater, strings.ToUpper(projectID), strings.ToUpper(storyID), validUpdate)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body.String())
	}
	hours := 8.5
	want := domain.Story{
		ID: storyID, ProjectID: strings.ToLower(projectID), Title: " Registro ", Description: "Descripción",
		Priority: "media", Status: domain.StatusInProgress, AcceptanceCriteria: []string{"Primero", " Segundo "},
		EstimatedHours: &hours,
	}
	if updater.calls != 1 || !reflect.DeepEqual(updater.story, want) {
		t.Errorf("calls = %d, story = %+v, want %+v", updater.calls, updater.story, want)
	}
}

func TestUpdateStoryMapsOnlyTypedMissingStoryTo404(t *testing.T) {
	var bodies []string
	// The same typed error serves a missing story and a story owned by another project.
	for _, project := range []string{projectID, otherProjectID} {
		updater := &fakeUpdater{err: application.ErrStoryNotFound}
		response := put(updater, project, storyID, validUpdate)
		assertError(t, response, http.StatusNotFound, "story_not_found", "story not found")
		bodies = append(bodies, response.Body.String())
		if updater.calls != 1 {
			t.Errorf("writes = %d, want one", updater.calls)
		}
	}
	if bodies[0] != bodies[1] {
		t.Errorf("not-found bodies differ: %q vs %q", bodies[0], bodies[1])
	}
}

func TestUpdateStoryHidesUnexpectedErrorsBehindAGeneric500(t *testing.T) {
	updater := &fakeUpdater{err: errors.New("secret database password")}
	response := put(updater, projectID, storyID, validUpdate)
	assertError(t, response, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	if bytes.Contains(response.Body.Bytes(), []byte("secret")) || updater.calls != 1 {
		t.Errorf("unsafe response or write count: %s, %d", response.Body.String(), updater.calls)
	}
}

func TestUpdateStoryRejectsUnsupportedMethods(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			updater := &fakeUpdater{}
			response := updateRequest(updateHandler(updater), method, projectID, storyID, validUpdate)
			assertError(t, response, http.StatusMethodNotAllowed, "method_not_allowed", "only PUT is supported")
			if updater.calls != 0 {
				t.Errorf("writes = %d, want zero", updater.calls)
			}
		})
	}
}

func TestUpdateStoryRejectsInvalidJSONShape(t *testing.T) {
	cases := map[string]string{
		"empty": "", "malformed": `{"title":`, "array": `[]`, "string": `"texto"`, "null": "null",
		"two objects":    validUpdate + validUpdate,
		"unknown key":    `{"owner":"x","title":"Hi","description":"D","priority":"alta","status":"pendiente","acceptance_criteria":["ok"],"estimated_hours":null}`,
		"id integer":     `{"id":7}`,
		"id null":        `{"id":null}`,
		"project_id":     `{"project_id":"` + otherProjectID + `"}`,
		"project null":   `{"project_id":null}`,
		"points int":     `{"story_points":8}`,
		"points zero":    `{"story_points":0}`,
		"points null":    `{"story_points":null}`,
		"title number":   `{"title":7,"description":"D","priority":"alta","status":"pendiente","acceptance_criteria":["ok"],"estimated_hours":1}`,
		"criteria text":  `{"title":"Hi","description":"D","priority":"alta","status":"pendiente","acceptance_criteria":"criterio","estimated_hours":1}`,
		"hours string":   `{"title":"Hi","description":"D","priority":"alta","status":"pendiente","acceptance_criteria":["ok"],"estimated_hours":"8"}`,
		"hours boolean":  `{"title":"Hi","description":"D","priority":"alta","status":"pendiente","acceptance_criteria":["ok"],"estimated_hours":true}`,
		"hours object":   `{"title":"Hi","description":"D","priority":"alta","status":"pendiente","acceptance_criteria":["ok"],"estimated_hours":{}}`,
		"null criterion": `{"title":"Hi","description":"D","priority":"alta","status":"pendiente","acceptance_criteria":["ok",null],"estimated_hours":1}`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			updater := &fakeUpdater{}
			response := put(updater, projectID, storyID, input)
			assertError(t, response, http.StatusBadRequest, "invalid_request", invalidRequest)
			if updater.calls != 0 {
				t.Errorf("writes = %d, want zero", updater.calls)
			}
		})
	}
}

func TestUpdateStoryReportsValidationFieldsBeforeWriting(t *testing.T) {
	const (
		required = "is required"
		hours    = "must be greater than 0, at most 99999.99 and have at most 2 decimals"
	)
	base := func(omit string) string {
		parts := map[string]string{
			"title": `"Hi"`, "description": `"D"`, "priority": `"alta"`, "status": `"pendiente"`,
			"acceptance_criteria": `["ok"]`, "estimated_hours": `3`,
		}
		var items []string
		for _, key := range []string{"title", "description", "priority", "status", "acceptance_criteria", "estimated_hours"} {
			if key != omit {
				items = append(items, `"`+key+`":`+parts[key])
			}
		}
		return "{" + strings.Join(items, ",") + "}"
	}
	cases := []struct {
		name, project, story, body string
		want                       map[string]string
	}{
		{"invalid story id", projectID, "abc", validUpdate, map[string]string{"story_id": "must be a valid UUID"}},
		{"invalid project id", "no-es-uuid", storyID, validUpdate, map[string]string{"project_id": "must be a valid UUID"}},
		{"both ids invalid", "no-es-uuid", "abc", validUpdate, map[string]string{
			"project_id": "must be a valid UUID", "story_id": "must be a valid UUID"}},
		{"missing title", projectID, storyID, base("title"), map[string]string{"title": required}},
		{"missing description", projectID, storyID, base("description"), map[string]string{"description": required}},
		{"missing priority", projectID, storyID, base("priority"), map[string]string{"priority": required}},
		{"missing status", projectID, storyID, base("status"), map[string]string{"status": required}},
		{"missing criteria", projectID, storyID, base("acceptance_criteria"), map[string]string{"acceptance_criteria": required}},
		{"missing estimated hours", projectID, storyID, base("estimated_hours"), map[string]string{"estimated_hours": required}},
		{"empty object", projectID, storyID, `{}`, map[string]string{
			"title": required, "description": required, "priority": required, "status": required,
			"acceptance_criteria": required, "estimated_hours": required}},
		{"blank content", projectID, storyID,
			`{"title":" ","description":"","priority":"ALTA","status":"cancelada","acceptance_criteria":[],"estimated_hours":0}`,
			map[string]string{
				"title": required, "description": required, "priority": "must be alta, media or baja",
				"status":              "must be pendiente, en_progreso or completada",
				"acceptance_criteria": "must contain at least one non-blank criterion", "estimated_hours": hours}},
		{"status outside the set", projectID, storyID,
			`{"title":"Hi","description":"D","priority":"alta","status":"Completada","acceptance_criteria":["ok"],"estimated_hours":1}`,
			map[string]string{"status": "must be pendiente, en_progreso or completada"}},
		{"too many decimals", projectID, storyID,
			`{"title":"Hi","description":"D","priority":"alta","status":"pendiente","acceptance_criteria":["ok"],"estimated_hours":1.001}`,
			map[string]string{"estimated_hours": hours}},
		{"invalid id plus empty title", projectID, "abc",
			`{"title":"","description":"D","priority":"alta","status":"pendiente","acceptance_criteria":["ok"],"estimated_hours":1}`,
			map[string]string{"story_id": "must be a valid UUID", "title": required}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			updater := &fakeUpdater{}
			response := put(updater, tc.project, tc.story, tc.body)
			body := assertError(t, response, http.StatusUnprocessableEntity, "validation_failed", "one or more fields are invalid")
			if got := decodeFields(t, body); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("fields = %v, want %v", got, tc.want)
			}
			if updater.calls != 0 {
				t.Errorf("writes = %d, want zero", updater.calls)
			}
		})
	}
}

func TestUpdateStoryDistinguishesNullFromAbsentEstimatedHours(t *testing.T) {
	const withHours = `{"title":"Hi","description":"D","priority":"alta","status":"pendiente","acceptance_criteria":["ok"],"estimated_hours":%s}`
	updater := &fakeUpdater{}
	response := put(updater, projectID, storyID, strings.Replace(withHours, "%s", "null", 1))
	if response.Code != http.StatusOK {
		t.Fatalf("null status = %d; body = %s", response.Code, response.Body.String())
	}
	if body := decode(t, response); string(body["estimated_hours"]) != "null" {
		t.Errorf("estimated_hours = %s, want null", body["estimated_hours"])
	}
	if updater.calls != 1 || updater.story.EstimatedHours != nil {
		t.Errorf("calls = %d, estimated hours = %v", updater.calls, updater.story.EstimatedHours)
	}

	absent := &fakeUpdater{}
	response = put(absent, projectID, storyID,
		`{"title":"Hi","description":"D","priority":"alta","status":"pendiente","acceptance_criteria":["ok"]}`)
	body := assertError(t, response, http.StatusUnprocessableEntity, "validation_failed", "one or more fields are invalid")
	if got := decodeFields(t, body); !reflect.DeepEqual(got, map[string]string{"estimated_hours": "is required"}) || absent.calls != 0 {
		t.Errorf("fields = %v, writes = %d", got, absent.calls)
	}
}

func TestUpdateStoryRespondsFromTheStoredStoryAndPreservesUnicode(t *testing.T) {
	stored := domain.Story{
		ID: storyID, ProjectID: projectID, Title: "Título ñ 日本", Description: "Descripción interna",
		Priority: "baja", Status: domain.StatusPending, AcceptanceCriteria: []string{"Criterio ✓"},
	}
	updater := &fakeUpdater{stored: &stored}
	response := put(updater, projectID, storyID, validUpdate)
	body := decode(t, response)
	var title, description string
	var criteria []string
	if err := json.Unmarshal(body["title"], &title); err != nil || title != stored.Title {
		t.Errorf("title = %q, error = %v; want %q (stored, not requested)", title, err, stored.Title)
	}
	if err := json.Unmarshal(body["description"], &description); err != nil || description != stored.Description {
		t.Errorf("description = %q, error = %v", description, err)
	}
	if err := json.Unmarshal(body["acceptance_criteria"], &criteria); err != nil || !reflect.DeepEqual(criteria, stored.AcceptanceCriteria) {
		t.Errorf("criteria = %v, error = %v", criteria, err)
	}
	if string(body["estimated_hours"]) != "null" || string(body["story_points"]) != "null" {
		t.Errorf("estimated_hours = %s, story_points = %s", body["estimated_hours"], body["story_points"])
	}
}

func TestUpdateStoryPreservesUnicodeSentByTheClient(t *testing.T) {
	updater := &fakeUpdater{}
	body := `{"title":"Título ñ 日本","description":"Descripción interna","priority":"baja","status":"completada","acceptance_criteria":["Criterio ✓"],"estimated_hours":0.01}`
	response := put(updater, projectID, storyID, body)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body.String())
	}
	if updater.story.Title != "Título ñ 日本" || updater.story.Description != "Descripción interna" ||
		!reflect.DeepEqual(updater.story.AcceptanceCriteria, []string{"Criterio ✓"}) {
		t.Errorf("stored content = %+v", updater.story)
	}
}
