package transporthttp_test

import (
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

const pointsProjectID = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
const pointsStoryID = "6c21cbd4-d9a7-42df-9c3a-c0866f058746"

type pointsSetter struct {
	calls          int
	project, story string
	points         int
	result         domain.Story
	err            error
}

func (s *pointsSetter) SetStoryPoints(_ context.Context, project, story string, points int) (domain.Story, error) {
	s.calls++
	s.project, s.story, s.points = project, story, points
	return s.result, s.err
}
func pointsRequest(h http.Handler, method, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/projects/"+pointsProjectID+"/stories/"+pointsStoryID+"/story-points", strings.NewReader(body))
	req.SetPathValue("project_id", pointsProjectID)
	req.SetPathValue("story_id", pointsStoryID)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}
func pointsHandler(s *pointsSetter) http.Handler {
	return transporthttp.NewSetStoryPointsHandler(application.NewSetStoryPointsUseCase(s))
}
func TestSetStoryPointsHandlerContract(t *testing.T) {
	stored := domain.Story{ID: pointsStoryID, ProjectID: pointsProjectID, Title: "Stored", Description: "Description", Priority: "alta", Status: domain.StatusPending, StoryPoints: func() *int { n := 13; return &n }(), AcceptanceCriteria: []string{"Ready"}}
	for _, tc := range []struct {
		name, method, body string
		status             int
		errCode            string
		setterErr          error
	}{
		{"wrong method", http.MethodPost, `{"story_points":13}`, 405, "method_not_allowed", nil},
		{"malformed", http.MethodPut, `{"story_points":`, 400, "invalid_request", nil},
		{"unknown field", http.MethodPut, `{"story_points":13,"other":1}`, 400, "invalid_request", nil},
		{"undecided", http.MethodPut, `{"story_points":"?"}`, 400, "invalid_request", nil},
		{"unsupported estimate", http.MethodPut, `{"story_points":4}`, 422, "validation_failed", nil},
		{"missing story", http.MethodPut, `{"story_points":13}`, 404, "story_not_found", application.ErrStoryNotFound},
		{"unexpected failure", http.MethodPut, `{"story_points":13}`, 500, "internal_error", errors.New("database failure")},
		{"valid", http.MethodPut, `{"story_points":13}`, 200, "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setter := &pointsSetter{result: stored, err: tc.setterErr}
			w := pointsRequest(pointsHandler(setter), tc.method, tc.body)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s, want %d", w.Code, w.Body.String(), tc.status)
			}
			if tc.errCode != "" {
				var body map[string]json.RawMessage
				_ = json.Unmarshal(w.Body.Bytes(), &body)
				var code string
				_ = json.Unmarshal(body["error"], &code)
				if code != tc.errCode {
					t.Errorf("error=%q, want %q", code, tc.errCode)
				}
			}
			called := tc.name == "missing story" || tc.name == "unexpected failure" || tc.name == "valid"
			if (setter.calls == 1) != called {
				t.Errorf("setter calls=%d, expected called=%t", setter.calls, called)
			}
			if tc.name == "valid" {
				if setter.project != pointsProjectID || setter.story != pointsStoryID || setter.points != 13 {
					t.Errorf("setter args=(%q, %q, %d), want (%q, %q, 13)", setter.project, setter.story, setter.points, pointsProjectID, pointsStoryID)
				}
				var body map[string]json.RawMessage
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatalf("decode successful response: %v", err)
				}
				for key, want := range map[string]string{"id": "\"" + pointsStoryID + "\"", "project_id": "\"" + pointsProjectID + "\"", "title": "\"Stored\"", "description": "\"Description\"", "priority": "\"alta\"", "story_points": "13", "status": "\"pendiente\"", "acceptance_criteria": "[\"Ready\"]"} {
					if string(body[key]) != want {
						t.Errorf("%s=%s, want %s", key, body[key], want)
					}
				}
			}
		})
	}
}
