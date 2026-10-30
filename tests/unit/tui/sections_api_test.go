package tui_test

import (
	"net/http"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

func TestMemberClientListUsesReadOnlyProjectMembersRoute(t *testing.T) {
	server, rec := recordingServer(t, http.StatusOK, `{"project_id":"p1","members":[]}`)
	response, err := tui.NewMemberClient(server.URL).List(t.Context(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	got := onlyRequest(t, rec)
	if got.Method != http.MethodGet || got.Path != "/projects/p1/members" || len(got.Body) != 0 {
		t.Errorf("request = %+v, want GET /projects/p1/members without body", got)
	}
	if response.StatusCode != http.StatusOK || string(response.Body) != `{"project_id":"p1","members":[]}` {
		t.Errorf("response = %+v, want 200 with the API body", response)
	}
}

func TestTaskClientListByStoryUsesReadOnlyStoryTasksRoute(t *testing.T) {
	server, rec := recordingServer(t, http.StatusOK, `{"project_id":"p1","story_id":"s1","tasks":[]}`)
	response, err := tui.NewTaskClient(server.URL).ListByStory(t.Context(), "p1", "s1")
	if err != nil {
		t.Fatal(err)
	}
	got := onlyRequest(t, rec)
	if got.Method != http.MethodGet || got.Path != "/projects/p1/stories/s1/tasks" || len(got.Body) != 0 {
		t.Errorf("request = %+v, want GET /projects/p1/stories/s1/tasks without body", got)
	}
	if response.StatusCode != http.StatusOK || string(response.Body) != `{"project_id":"p1","story_id":"s1","tasks":[]}` {
		t.Errorf("response = %+v, want 200 with the API body", response)
	}
}

func TestMetricsClientUsesProjectAndSprintGETRoutes(t *testing.T) {
	server, rec := recordingServer(t, http.StatusOK, `{"story_points_planned":0}`)
	client := tui.NewMetricsClient(server.URL)
	if _, err := client.Project(t.Context(), "p/1"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Sprint(t.Context(), "p 1", "s/1"); err != nil {
		t.Fatal(err)
	}
	requests := rec.all()
	if len(requests) != 2 {
		t.Fatalf("requests = %d, want 2", len(requests))
	}
	want := []string{"/projects/p%2F1/metrics", "/projects/p%201/sprints/s%2F1/metrics"}
	for i, request := range requests {
		if request.Method != http.MethodGet || request.Path != want[i] || len(request.Body) != 0 {
			t.Errorf("request[%d] = %+v, want GET %s without body", i, request, want[i])
		}
	}
}

func TestSectionClientsEscapePathSegments(t *testing.T) {
	server, rec := recordingServer(t, http.StatusOK, `{}`)
	if _, err := tui.NewMemberClient(server.URL).List(t.Context(), "p/1"); err != nil {
		t.Fatal(err)
	}
	if _, err := tui.NewTaskClient(server.URL).ListByStory(t.Context(), "p 1", "s/1"); err != nil {
		t.Fatal(err)
	}
	requests := rec.all()
	if len(requests) != 2 {
		t.Fatalf("requests = %d, want 2", len(requests))
	}
	if requests[0].Path != "/projects/p%2F1/members" {
		t.Errorf("member path = %q", requests[0].Path)
	}
	if requests[1].Path != "/projects/p%201/stories/s%2F1/tasks" {
		t.Errorf("task path = %q", requests[1].Path)
	}
}
