package tui_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

func TestSprintHistoryUsesReadOnlyGetAndPreservesResponse(t *testing.T) {
	const body = `{"project_id":"p1","sprints":[{"id":"s1","project_id":"p1","sprint_goal":"Ship","is_closed":true,"stories":[{"id":"story1","title":"Feature","priority":"high","story_points":3,"completed_at":null,"completed":false}]}]}`
	var method, path string
	var requestBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.EscapedPath()
		requestBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	response, err := tui.NewSprintClient(server.URL).History(t.Context(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodGet || path != "/projects/p1/sprints" || len(requestBody) != 0 {
		t.Fatalf("history request = %s %s body=%q; want read-only GET without body", method, path, requestBody)
	}
	if response.StatusCode != http.StatusOK || string(response.Body) != body {
		t.Fatalf("history response = %+v", response)
	}
	var history tui.SprintHistory
	if err := json.Unmarshal(response.Body, &history); err != nil {
		t.Fatal(err)
	}
	story := history.Sprints[0].Stories[0]
	if history.ProjectID != "p1" || history.Sprints[0].SprintGoal != "Ship" || story.ID != "story1" || story.Completed || story.StoryPoints == nil || *story.StoryPoints != 3 || story.CompletedAt != nil {
		t.Fatalf("decoded history = %+v", history)
	}
}

func TestSprintCloseDecodesSuccessfulResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.EscapedPath() != "/projects/p1/sprints/s1/closure" {
			t.Errorf("request = %s %s, want POST closure endpoint", r.Method, r.URL.EscapedPath())
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"project_id":"p1","sprint_id":"s1","is_closed":true}`))
	}))
	defer server.Close()

	response, err := tui.NewSprintClient(server.URL).Close(t.Context(), "p1", "s1")
	if err != nil {
		t.Fatal(err)
	}
	var closure tui.SprintClosure
	if err := json.Unmarshal(response.Body, &closure); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || closure.ProjectID != "p1" || closure.SprintID != "s1" || !closure.IsClosed {
		t.Fatalf("closure = %+v response=%+v", closure, response)
	}
}

func TestSprintCloseUsesPostAndPreservesConflict(t *testing.T) {
	const body = `{"error":"sprint_closed","message":"sprint is already closed"}`
	var method, path string
	var requestBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.EscapedPath()
		requestBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	response, err := tui.NewSprintClient(server.URL).Close(t.Context(), "p1", "s1")
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost || path != "/projects/p1/sprints/s1/closure" || len(requestBody) != 0 {
		t.Fatalf("closure request = %s %s body=%q; want POST without body", method, path, requestBody)
	}
	if response.StatusCode != http.StatusConflict || string(response.Body) != body {
		t.Fatalf("closure response = %+v", response)
	}
}
