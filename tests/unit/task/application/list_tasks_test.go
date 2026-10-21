package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/task/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/task/domain"
)

type taskLister struct {
	calls     int
	projectID string
	storyID   string
	tasks     []domain.Task
	err       error
}

func (l *taskLister) ListByProjectStory(_ context.Context, projectID, storyID string) ([]domain.Task, error) {
	l.calls++
	l.projectID, l.storyID = projectID, storyID
	return l.tasks, l.err
}

func TestListTasksRejectsInvalidRouteIDsBeforeReading(t *testing.T) {
	for _, tc := range []struct {
		name      string
		projectID string
		storyID   string
		field     string
	}{
		{"invalid project", "not-a-uuid", "e99c05a4-03ea-4c19-8f59-286dd59e7aa1", "project_id"},
		{"invalid story", "82d38423-f02d-4259-9e35-4a291585bb1e", "not-a-uuid", "story_id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lister := &taskLister{}
			_, err := application.NewListTasksUseCase(lister).Execute(context.Background(), application.ListTasksQuery{
				ProjectID: tc.projectID, StoryID: tc.storyID,
			})
			var validation *domain.ValidationError
			if !errors.As(err, &validation) || validation.Fields[tc.field] == "" {
				t.Fatalf("err = %v, want %s validation error", err, tc.field)
			}
			if lister.calls != 0 {
				t.Fatalf("lister called %d times for invalid route", lister.calls)
			}
		})
	}
}

func TestListTasksReturnsCanonicalContextAndTasks(t *testing.T) {
	hours := 4.5
	lister := &taskLister{tasks: []domain.Task{{
		ID: "t1", ProjectID: "82d38423-f02d-4259-9e35-4a291585bb1e",
		SprintID: "b2a6a455-06e2-41ee-b011-7462c71375a0", StoryID: "e99c05a4-03ea-4c19-8f59-286dd59e7aa1",
		Title: "Diseñar", EstimatedHours: &hours,
	}}}
	got, err := application.NewListTasksUseCase(lister).Execute(context.Background(), application.ListTasksQuery{
		ProjectID: "82D38423-F02D-4259-9E35-4A291585BB1E",
		StoryID:   "E99C05A4-03EA-4C19-8F59-286DD59E7AA1",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.ProjectID != "82d38423-f02d-4259-9e35-4a291585bb1e" || got.StoryID != "e99c05a4-03ea-4c19-8f59-286dd59e7aa1" {
		t.Fatalf("context = %q/%q, want canonical ids", got.ProjectID, got.StoryID)
	}
	if lister.projectID != got.ProjectID || lister.storyID != got.StoryID {
		t.Fatalf("lister received %q/%q", lister.projectID, lister.storyID)
	}
	if len(got.Tasks) != 1 || got.Tasks[0].Title != "Diseñar" {
		t.Fatalf("tasks = %#v, want the listed task", got.Tasks)
	}
}

func TestListTasksPreservesAnEmptyCollectionAndPropagatesErrors(t *testing.T) {
	empty := &taskLister{tasks: []domain.Task{}}
	got, err := application.NewListTasksUseCase(empty).Execute(context.Background(), application.ListTasksQuery{
		ProjectID: "82d38423-f02d-4259-9e35-4a291585bb1e",
		StoryID:   "e99c05a4-03ea-4c19-8f59-286dd59e7aa1",
	})
	if err != nil || got.Tasks == nil || len(got.Tasks) != 0 {
		t.Fatalf("empty listing = %#v, err %v; want non-nil empty slice", got.Tasks, err)
	}

	failure := errors.New("storage unavailable")
	failing := &taskLister{err: failure}
	_, err = application.NewListTasksUseCase(failing).Execute(context.Background(), application.ListTasksQuery{
		ProjectID: "82d38423-f02d-4259-9e35-4a291585bb1e",
		StoryID:   "e99c05a4-03ea-4c19-8f59-286dd59e7aa1",
	})
	if !errors.Is(err, failure) {
		t.Fatalf("err = %v, want the lister failure", err)
	}
}
