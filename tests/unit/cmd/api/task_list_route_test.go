package api_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	taskdomain "github.com/valerubio7/software-metrics-and-estimation/internal/task/domain"
)

// listableTaskRepository stands in for the PostgreSQL repository, which reads story tasks.
type listableTaskRepository struct {
	*fakeTaskRepository
	listed    []taskdomain.Task
	listErr   error
	listCalls int
}

func (r *listableTaskRepository) ListByProjectStory(_ context.Context, projectID, storyID string) ([]taskdomain.Task, error) {
	r.listCalls++
	if r.listErr != nil {
		return nil, r.listErr
	}
	tasks := make([]taskdomain.Task, 0, len(r.listed))
	for _, task := range r.listed {
		if task.ProjectID == projectID && task.StoryID == storyID {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func taskDependencies(repository *listableTaskRepository) api.HTTPDependencies {
	return api.HTTPDependencies{Tasks: &api.TaskDependencies{
		Repository: repository,
		GenerateID: func() string { return "5c21cbd4-d9a7-42df-9c3a-c0866f058746" },
		Lister:     repository,
	}}
}

func TestStoryTaskCollectionRouteIsComposedOnlyForListableRepository(t *testing.T) {
	const projectID = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	const storyID = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	hours := 4.5
	repository := &listableTaskRepository{
		fakeTaskRepository: &fakeTaskRepository{},
		listed: []taskdomain.Task{
			{ID: "t1", ProjectID: projectID, SprintID: "b2a6a455-06e2-41ee-b011-7462c71375a0", StoryID: storyID, Title: "Diseñar", EstimatedHours: &hours},
			{ID: "t2", ProjectID: projectID, SprintID: "b2a6a455-06e2-41ee-b011-7462c71375a0", StoryID: storyID, Title: "Probar"},
			{ID: "t3", ProjectID: projectID, SprintID: "b2a6a455-06e2-41ee-b011-7462c71375a0", StoryID: "another-story", Title: "Otra"},
		},
	}
	handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return projectID }, taskDependencies(repository))

	response := serve(handler, http.MethodGet, "/projects/"+projectID+"/stories/"+storyID+"/tasks", "")
	if response.Code != http.StatusOK || repository.listCalls != 1 {
		t.Fatalf("GET tasks = %d (reads %d) body=%s", response.Code, repository.listCalls, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `"project_id":"`+projectID+`"`) || !strings.Contains(body, `"story_id":"`+storyID+`"`) || !strings.Contains(body, `"title":"Diseñar"`) || !strings.Contains(body, `"estimated_hours":null`) {
		t.Fatalf("task collection body = %s", body)
	}
	if strings.Contains(body, "Otra") {
		t.Fatalf("task collection leaked another story: %s", body)
	}

	// A task module without the listing port keeps the collection unreadable: the route is absent.
	withoutLister := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return projectID }, api.HTTPDependencies{
		Tasks: &api.TaskDependencies{Repository: &fakeTaskRepository{}, GenerateID: func() string { return projectID }},
	})
	if got := serve(withoutLister, http.MethodGet, "/projects/"+projectID+"/stories/"+storyID+"/tasks", ""); got.Code != http.StatusNotFound {
		t.Fatalf("GET tasks without lister = %d, want 404", got.Code)
	}
}

func TestStoryTaskCollectionRouteSerializesEmptyArrayAndHidesInternalErrors(t *testing.T) {
	const projectID = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	const storyID = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	empty := &listableTaskRepository{fakeTaskRepository: &fakeTaskRepository{}}
	handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return projectID }, taskDependencies(empty))
	if got := serve(handler, http.MethodGet, "/projects/"+projectID+"/stories/"+storyID+"/tasks", ""); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"tasks":[]`) {
		t.Fatalf("empty tasks = %d %s", got.Code, got.Body.String())
	}

	failing := &listableTaskRepository{
		fakeTaskRepository: &fakeTaskRepository{},
		listErr:            errors.New("connection refused: host=db"),
	}
	failingHandler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return projectID }, taskDependencies(failing))
	got := serve(failingHandler, http.MethodGet, "/projects/"+projectID+"/stories/"+storyID+"/tasks", "")
	if got.Code != http.StatusInternalServerError || strings.Contains(got.Body.String(), "connection refused") {
		t.Fatalf("failing tasks = %d %s, want 500 without internal detail", got.Code, got.Body.String())
	}
}
