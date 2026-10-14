package transporthttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/transport/http"
)

const listHandlerProjectID = "82d38423-f02d-4259-9e35-4a291585bb1e"

// fakeHistoryReader is a read-only fake of the SprintHistoryReader port.
type fakeHistoryReader struct {
	calls     int
	projectID string
	sprints   []domain.ClosedSprint
	err       error
}

func (r *fakeHistoryReader) ListClosedByProject(_ context.Context, projectID string) ([]domain.ClosedSprint, error) {
	r.calls++
	r.projectID = projectID
	return r.sprints, r.err
}

func newListSprintsHandler(reader *fakeHistoryReader) http.Handler {
	return transporthttp.NewListSprintsHandler(application.NewListSprintsUseCase(reader))
}

func serveList(reader *fakeHistoryReader, method, projectID string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "/", nil)
	request.SetPathValue("project_id", projectID)
	response := httptest.NewRecorder()
	newListSprintsHandler(reader).ServeHTTP(response, request)
	return response
}

func decodeRawObject(t *testing.T, raw []byte) map[string]json.RawMessage {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatalf("body is not a JSON object: %v: %s", err, raw)
	}
	return object
}

func TestListSprintsHandlerReturnsTheContainerInReaderOrder(t *testing.T) {
	reader := &fakeHistoryReader{sprints: []domain.ClosedSprint{
		{ID: "11111111-1111-4111-8111-111111111111", ProjectID: listHandlerProjectID, SprintGoal: "newest", Stories: []domain.PlannedStory{
			{ID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Title: "first", Priority: "alta"},
		}},
		{ID: "22222222-2222-4222-8222-222222222222", ProjectID: listHandlerProjectID, SprintGoal: "oldest", Stories: []domain.PlannedStory{
			{ID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Title: "second", Priority: "baja"},
		}},
	}}

	response := serveList(reader, http.MethodGet, listHandlerProjectID)

	if response.Code != http.StatusOK || reader.calls != 1 || reader.projectID != listHandlerProjectID {
		t.Fatalf("status=%d calls=%d projectID=%q body=%s", response.Code, reader.calls, reader.projectID, response.Body.String())
	}
	var body struct {
		ProjectID string `json:"project_id"`
		Sprints   []struct {
			ID         string `json:"id"`
			SprintGoal string `json:"sprint_goal"`
		} `json:"sprints"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v: %s", err, response.Body.String())
	}
	if body.ProjectID != listHandlerProjectID || len(body.Sprints) != 2 ||
		body.Sprints[0].SprintGoal != "newest" || body.Sprints[1].SprintGoal != "oldest" {
		t.Fatalf("unexpected container or order: %s", response.Body.String())
	}
}

func TestListSprintsHandlerSerializesEmptyListsAsArraysNeverNull(t *testing.T) {
	t.Run("no sprints", func(t *testing.T) {
		response := serveList(&fakeHistoryReader{sprints: nil}, http.MethodGet, listHandlerProjectID)

		if response.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
		if got := string(decodeRawObject(t, response.Body.Bytes())["sprints"]); got != "[]" {
			t.Errorf(`"sprints" = %s, want []`, got)
		}
	})
	t.Run("sprint without stories", func(t *testing.T) {
		reader := &fakeHistoryReader{sprints: []domain.ClosedSprint{
			{ID: "11111111-1111-4111-8111-111111111111", ProjectID: listHandlerProjectID, SprintGoal: "empty", Stories: nil},
		}}

		response := serveList(reader, http.MethodGet, listHandlerProjectID)

		var sprints []map[string]json.RawMessage
		if err := json.Unmarshal(decodeRawObject(t, response.Body.Bytes())["sprints"], &sprints); err != nil || len(sprints) != 1 {
			t.Fatalf("sprints = %v (err %v): %s", sprints, err, response.Body.String())
		}
		if got := string(sprints[0]["stories"]); got != "[]" {
			t.Errorf(`"stories" = %s, want []`, got)
		}
	})
}

func TestListSprintsHandlerRejectsInvalidProjectIDBeforeReadingStorage(t *testing.T) {
	reader := &fakeHistoryReader{}

	response := serveList(reader, http.MethodGet, "no-es-uuid")

	if response.Code != http.StatusUnprocessableEntity || reader.calls != 0 {
		t.Fatalf("status=%d calls=%d body=%s", response.Code, reader.calls, response.Body.String())
	}
	var body struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.Error != "validation_failed" || body.Fields["project_id"] != "must be a valid UUID" {
		t.Errorf("unexpected validation body: %s", response.Body.String())
	}
}

func TestListSprintsHandlerMapsNotFoundWithoutASprintsKey(t *testing.T) {
	response := serveList(&fakeHistoryReader{err: application.ErrProjectNotFound}, http.MethodGet, listHandlerProjectID)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	body := decodeRawObject(t, response.Body.Bytes())
	if string(body["error"]) != `"project_not_found"` {
		t.Errorf("error = %s, want project_not_found", body["error"])
	}
	if _, present := body["sprints"]; present {
		t.Errorf("a 404 must not carry a sprints key: %s", response.Body.String())
	}
}

func TestListSprintsHandlerHidesUnexpectedErrorDetails(t *testing.T) {
	secret := errors.New("private database detail")
	reader := &fakeHistoryReader{
		err:     secret,
		sprints: []domain.ClosedSprint{{ID: "11111111-1111-4111-8111-111111111111", SprintGoal: "must not leak"}},
	}

	response := serveList(reader, http.MethodGet, listHandlerProjectID)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	body := decodeRawObject(t, response.Body.Bytes())
	if string(body["error"]) != `"internal_error"` || string(body["message"]) != `"an unexpected error occurred"` {
		t.Errorf("unexpected error body: %s", response.Body.String())
	}
	if strings.Contains(response.Body.String(), secret.Error()) || strings.Contains(response.Body.String(), "must not leak") {
		t.Errorf("internal detail leaked: %s", response.Body.String())
	}
	if _, present := body["sprints"]; present {
		t.Errorf("a 500 must not carry a sprints key: %s", response.Body.String())
	}
}

func sortedKeys(object map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestListSprintsHandlerExposesExactlyTheSpecifiedFields(t *testing.T) {
	points := 5
	completedAt := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	reader := &fakeHistoryReader{sprints: []domain.ClosedSprint{{
		ID: "11111111-1111-4111-8111-111111111111", ProjectID: listHandlerProjectID, SprintGoal: "goal",
		Stories: []domain.PlannedStory{{
			ID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Title: "story", Priority: "alta",
			StoryPoints: &points, CompletedAt: &completedAt,
		}},
	}}}

	response := serveList(reader, http.MethodGet, listHandlerProjectID)

	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var sprints []map[string]json.RawMessage
	if err := json.Unmarshal(decodeRawObject(t, response.Body.Bytes())["sprints"], &sprints); err != nil || len(sprints) != 1 {
		t.Fatalf("sprints = %v (err %v): %s", sprints, err, response.Body.String())
	}
	wantSprintKeys := []string{"id", "is_closed", "project_id", "sprint_goal", "stories"}
	if got := sortedKeys(sprints[0]); !reflect.DeepEqual(got, wantSprintKeys) {
		t.Errorf("sprint keys = %v, want %v", got, wantSprintKeys)
	}
	if got := string(sprints[0]["is_closed"]); got != "true" {
		t.Errorf("is_closed = %s, want true", got)
	}
	var stories []map[string]json.RawMessage
	if err := json.Unmarshal(sprints[0]["stories"], &stories); err != nil || len(stories) != 1 {
		t.Fatalf("stories = %v (err %v)", stories, err)
	}
	wantStoryKeys := []string{"completed", "completed_at", "id", "priority", "story_points", "title"}
	if got := sortedKeys(stories[0]); !reflect.DeepEqual(got, wantStoryKeys) {
		t.Errorf("story keys = %v, want %v", got, wantStoryKeys)
	}
	for _, forbidden := range []string{"seq", "description", "acceptance_criteria", "estimated_hours", "status"} {
		if strings.Contains(response.Body.String(), `"`+forbidden+`"`) {
			t.Errorf("forbidden field %q exposed: %s", forbidden, response.Body.String())
		}
	}
}

func TestListSprintsHandlerRepresentsCompletionAndNullableFields(t *testing.T) {
	points := 8
	other := time.FixedZone("UTC-3", -3*60*60)
	completedAt := time.Date(2026, 10, 11, 7, 30, 0, 0, other) // 10:30 UTC
	reader := &fakeHistoryReader{sprints: []domain.ClosedSprint{{
		ID: "11111111-1111-4111-8111-111111111111", ProjectID: listHandlerProjectID, SprintGoal: "goal",
		Stories: []domain.PlannedStory{
			{ID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Title: "done", Priority: "alta", StoryPoints: &points, CompletedAt: &completedAt},
			{ID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Title: "pending", Priority: "baja"},
		},
	}}}

	response := serveList(reader, http.MethodGet, listHandlerProjectID)

	var sprints []struct {
		Stories []map[string]json.RawMessage `json:"stories"`
	}
	if err := json.Unmarshal(decodeRawObject(t, response.Body.Bytes())["sprints"], &sprints); err != nil ||
		len(sprints) != 1 || len(sprints[0].Stories) != 2 {
		t.Fatalf("unexpected body (err %v): %s", err, response.Body.String())
	}
	done, pending := sprints[0].Stories[0], sprints[0].Stories[1]
	if string(done["completed"]) != "true" || string(done["story_points"]) != "8" ||
		string(done["completed_at"]) != `"2026-10-11T10:30:00Z"` {
		t.Errorf("completed story = completed:%s story_points:%s completed_at:%s, want true / 8 / UTC instant",
			done["completed"], done["story_points"], done["completed_at"])
	}
	if string(pending["completed"]) != "false" || string(pending["story_points"]) != "null" ||
		string(pending["completed_at"]) != "null" {
		t.Errorf("pending story = completed:%s story_points:%s completed_at:%s, want false / null / null",
			pending["completed"], pending["story_points"], pending["completed_at"])
	}
}

func TestListSprintsHandlerAcceptsHeadAndRejectsOtherMethodsBeforeReading(t *testing.T) {
	t.Run("HEAD is served like GET", func(t *testing.T) {
		reader := &fakeHistoryReader{}

		response := serveList(reader, http.MethodHead, listHandlerProjectID)

		if response.Code != http.StatusOK || reader.calls != 1 {
			t.Fatalf("status=%d calls=%d body=%s", response.Code, reader.calls, response.Body.String())
		}
	})
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run(method+" is rejected", func(t *testing.T) {
			reader := &fakeHistoryReader{}

			response := serveList(reader, method, listHandlerProjectID)

			if response.Code != http.StatusMethodNotAllowed || reader.calls != 0 {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, reader.calls, response.Body.String())
			}
			body := decodeRawObject(t, response.Body.Bytes())
			if string(body["error"]) != `"method_not_allowed"` || string(body["message"]) != `"only GET is supported"` {
				t.Errorf("unexpected 405 body: %s", response.Body.String())
			}
		})
	}
}
