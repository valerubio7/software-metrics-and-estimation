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

func TestSectionClientsEscapePathSegments(t *testing.T) {
	server, rec := recordingServer(t, http.StatusOK, `{}`)
	if _, err := tui.NewMemberClient(server.URL).List(t.Context(), "p/1"); err != nil {
		t.Fatal(err)
	}
	if _, err := tui.NewTaskClient(server.URL).ListByStory(t.Context(), "p 1", "s/1"); err != nil {
		t.Fatal(err)
	}
	if _, err := tui.NewMetricsClient(server.URL).Project(t.Context(), "p 1"); err != nil {
		t.Fatal(err)
	}
	if _, err := tui.NewMetricsClient(server.URL).Sprint(t.Context(), "p/1", "s 1"); err != nil {
		t.Fatal(err)
	}
	requests := rec.all()
	if len(requests) != 4 {
		t.Fatalf("requests = %d, want 4", len(requests))
	}
	if requests[0].Path != "/projects/p%2F1/members" {
		t.Errorf("member path = %q", requests[0].Path)
	}
	if requests[1].Path != "/projects/p%201/stories/s%2F1/tasks" {
		t.Errorf("task path = %q", requests[1].Path)
	}
	if requests[2].Path != "/projects/p%201/metrics" {
		t.Errorf("project metrics path = %q", requests[2].Path)
	}
	if requests[3].Path != "/projects/p%2F1/sprints/s%201/metrics" {
		t.Errorf("sprint metrics path = %q", requests[3].Path)
	}
}

func TestMetricsClientUsesReadOnlyRoutes(t *testing.T) {
	server, rec := recordingServer(t, http.StatusOK, `{"story_points_planned":8}`)
	client := tui.NewMetricsClient(server.URL)
	projectResponse, err := client.Project(t.Context(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	sprintResponse, err := client.Sprint(t.Context(), "p1", "sp1")
	if err != nil {
		t.Fatal(err)
	}
	requests := rec.all()
	if len(requests) != 2 {
		t.Fatalf("requests = %d, want 2", len(requests))
	}
	if requests[0].Method != http.MethodGet || requests[0].Path != "/projects/p1/metrics" || len(requests[0].Body) != 0 {
		t.Errorf("project metrics request = %+v", requests[0])
	}
	if requests[1].Method != http.MethodGet || requests[1].Path != "/projects/p1/sprints/sp1/metrics" || len(requests[1].Body) != 0 {
		t.Errorf("sprint metrics request = %+v", requests[1])
	}
	if projectResponse.StatusCode != http.StatusOK || sprintResponse.StatusCode != http.StatusOK {
		t.Errorf("responses = %+v / %+v", projectResponse, sprintResponse)
	}
}
