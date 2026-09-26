package transporthttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/story/transport/http"
)

// fakeLister records the reads it receives and returns the configured stories and error.
// It exposes reading only, like the port it stands for.
type fakeLister struct {
	calls     int
	projectID string
	stories   []domain.Story
	err       error
}

func (l *fakeLister) ListByProject(_ context.Context, projectID string) ([]domain.Story, error) {
	l.calls++
	l.projectID = projectID
	return l.stories, l.err
}

func listHandler(lister *fakeLister) http.Handler {
	return transporthttp.NewListStoriesHandler(application.NewListStoriesUseCase(lister))
}

func listRequest(handler http.Handler, method, project string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/projects/"+project+"/stories", nil)
	req.SetPathValue("project_id", project)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func getBacklog(lister *fakeLister, project string) *httptest.ResponseRecorder {
	return listRequest(listHandler(lister), http.MethodGet, project)
}

// decodeStories decodes the stories of a backlog response into raw-key objects.
func decodeStories(t *testing.T, body map[string]json.RawMessage) []map[string]json.RawMessage {
	t.Helper()
	var stories []map[string]json.RawMessage
	if err := json.Unmarshal(body["stories"], &stories); err != nil {
		t.Fatalf("stories = %s, error = %v", body["stories"], err)
	}
	return stories
}

func creationOrderedStories() []domain.Story {
	points, hours := 5, 8.5
	return []domain.Story{
		{ID: "id-baja", ProjectID: "5c21cbd4-d9a7-42df-9c3a-c0866f058746", Title: "Baja", Description: "Detalle baja",
			Priority: "baja", Status: domain.StatusPending, AcceptanceCriteria: []string{"Uno"}},
		{ID: "id-alta", ProjectID: "5c21cbd4-d9a7-42df-9c3a-c0866f058746", Title: "Alta", Description: "Detalle alta",
			Priority: "alta", Status: domain.StatusInProgress, StoryPoints: &points,
			AcceptanceCriteria: []string{"Dos", " Tres "}, EstimatedHours: &hours},
		{ID: "id-media", ProjectID: "5c21cbd4-d9a7-42df-9c3a-c0866f058746", Title: "Media", Description: "Detalle media",
			Priority: "media", Status: domain.StatusCompleted, AcceptanceCriteria: []string{"Cuatro"}},
	}
}

func TestListStoriesReturnsTheOrderedBacklogInsideTheContainer(t *testing.T) {
	lister := &fakeLister{stories: creationOrderedStories()}

	response := getBacklog(lister, projectID)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body.String())
	}
	body := decode(t, response)
	if string(body["project_id"]) != `"5c21cbd4-d9a7-42df-9c3a-c0866f058746"` {
		t.Errorf("project_id = %s, want the canonical lowercase identifier", body["project_id"])
	}
	stories := decodeStories(t, body)
	var ids []string
	for _, story := range stories {
		ids = append(ids, string(story["id"]))
	}
	if got, want := strings.Join(ids, ","), `"id-alta","id-media","id-baja"`; got != want {
		t.Errorf("story ids = %s, want priority order %s", got, want)
	}
	if string(stories[0]["story_points"]) != "5" || string(stories[0]["estimated_hours"]) != "8.5" {
		t.Errorf("estimated story = (%s, %s), want (5, 8.5)", stories[0]["story_points"], stories[0]["estimated_hours"])
	}
	for _, story := range stories[1:] {
		if string(story["story_points"]) != "null" || string(story["estimated_hours"]) != "null" {
			t.Errorf("unestimated story %s = (%s, %s), want (null, null)", story["id"], story["story_points"], story["estimated_hours"])
		}
	}
}

func TestListStoriesRespondsWithAnEmptyArrayNeverNull(t *testing.T) {
	// A nil result from the port must still serialize as [] (the empty backlog).
	response := getBacklog(&fakeLister{stories: nil}, projectID)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body.String())
	}
	raw := response.Body.String()
	if !strings.Contains(raw, `"stories":[]`) || strings.Contains(raw, `"stories":null`) {
		t.Errorf("body = %s, want \"stories\":[] and never null", raw)
	}
	if body := decode(t, response); string(body["project_id"]) != `"5c21cbd4-d9a7-42df-9c3a-c0866f058746"` {
		t.Errorf("project_id = %s, want the canonical lowercase identifier", body["project_id"])
	}
}

func TestListStoriesRejectsInvalidProjectIdentifiersBeforeReading(t *testing.T) {
	for _, project := range []string{"no-es-uuid", "abc"} {
		t.Run(project, func(t *testing.T) {
			lister := &fakeLister{stories: creationOrderedStories()}

			response := getBacklog(lister, project)

			body := assertError(t, response, http.StatusUnprocessableEntity, "validation_failed", "one or more fields are invalid")
			if fields := decodeFields(t, body); fields["project_id"] != "must be a valid UUID" {
				t.Errorf("fields = %v, want project_id: must be a valid UUID", fields)
			}
			if _, present := body["stories"]; present || lister.calls != 0 {
				t.Errorf("stories present = %v, reads = %d, want no stories and no reads", present, lister.calls)
			}
		})
	}
}

func TestListStoriesMapsOnlyTheTypedMissingProjectTo404(t *testing.T) {
	lister := &fakeLister{err: application.ErrProjectNotFound}

	response := getBacklog(lister, projectID)

	body := assertError(t, response, http.StatusNotFound, "project_not_found", "project not found")
	if _, present := body["stories"]; present {
		t.Errorf("404 body = %s, want no stories key", response.Body.String())
	}
	if lister.calls != 1 {
		t.Errorf("reads = %d, want one", lister.calls)
	}
}

func TestListStoriesHidesUnexpectedErrorsBehindAGeneric500(t *testing.T) {
	lister := &fakeLister{err: errors.New("secret database password")}

	response := getBacklog(lister, projectID)

	assertError(t, response, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	if bytes.Contains(response.Body.Bytes(), []byte("secret")) {
		t.Errorf("body leaks the internal error: %s", response.Body.String())
	}
}

func TestListStoriesExposesExactlyTheNineStoryKeysAndNoSequence(t *testing.T) {
	response := getBacklog(&fakeLister{stories: creationOrderedStories()}, projectID)

	body := decode(t, response)
	if len(body) != 2 {
		t.Errorf("container has %d keys, want exactly project_id and stories: %s", len(body), response.Body.String())
	}
	stories := decodeStories(t, body)
	if len(stories) != 3 {
		t.Fatalf("stories = %d, want 3", len(stories))
	}
	for _, story := range stories {
		if len(story) != 9 {
			t.Errorf("story %s has %d keys, want 9", story["id"], len(story))
		}
		for _, key := range []string{"id", "project_id", "title", "description", "priority", "status",
			"story_points", "acceptance_criteria", "estimated_hours"} {
			if _, present := story[key]; !present {
				t.Errorf("story %s lacks key %q", story["id"], key)
			}
		}
	}
	if strings.Contains(response.Body.String(), "seq") {
		t.Errorf("body exposes the storage sequence: %s", response.Body.String())
	}
}

func TestListStoriesPassesTheCanonicalIdentifierToTheReader(t *testing.T) {
	lister := &fakeLister{}

	getBacklog(lister, projectID)

	if lister.calls != 1 || lister.projectID != "5c21cbd4-d9a7-42df-9c3a-c0866f058746" {
		t.Errorf("reads = %d, project = %q, want one read of the canonical lowercase identifier", lister.calls, lister.projectID)
	}
}

func TestListStoriesAcceptsHEADAndRejectsOtherMethodsWithoutReading(t *testing.T) {
	head := &fakeLister{stories: creationOrderedStories()}
	if response := listRequest(listHandler(head), http.MethodHead, projectID); response.Code != http.StatusOK || head.calls != 1 {
		t.Errorf("HEAD = status %d, reads %d, want 200 and one read", response.Code, head.calls)
	}

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			lister := &fakeLister{stories: creationOrderedStories()}

			response := listRequest(listHandler(lister), method, projectID)

			assertError(t, response, http.StatusMethodNotAllowed, "method_not_allowed", "only GET is supported")
			if lister.calls != 0 {
				t.Errorf("reads = %d, want none for an unsupported method", lister.calls)
			}
		})
	}
}

func TestListStoriesNeverReturnsAPartialListAlongsideAnError(t *testing.T) {
	// The port misbehaves by returning stories together with an error.
	lister := &fakeLister{stories: creationOrderedStories(), err: errors.New("connection reset")}

	response := getBacklog(lister, projectID)

	body := assertError(t, response, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	if _, present := body["stories"]; present {
		t.Errorf("error body = %s, want no partial stories", response.Body.String())
	}
}

func TestListStoriesRespondsJSONForEverySuccessAndErrorStatus(t *testing.T) {
	for name, lister := range map[string]*fakeLister{
		"200": {stories: creationOrderedStories()},
		"404": {err: application.ErrProjectNotFound},
		"500": {err: errors.New("boom")},
	} {
		t.Run(name, func(t *testing.T) {
			response := getBacklog(lister, projectID)

			if got := response.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("status %d Content-Type = %q, want application/json", response.Code, got)
			}
		})
	}
}
