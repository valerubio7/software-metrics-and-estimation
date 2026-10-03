package application_test

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/task/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/task/domain"
)

// fakeTaskRepository records the single batch call it receives and returns a
// configurable error, standing in for application.TaskRepository.
type fakeTaskRepository struct {
	calls     int
	projectID string
	sprintID  string
	storyID   string
	tasks     []domain.Task
	err       error
}

func (r *fakeTaskRepository) CreateForSprintStory(_ context.Context, projectID, sprintID, storyID string, tasks []domain.Task) error {
	r.calls++
	r.projectID, r.sprintID, r.storyID = projectID, sprintID, storyID
	r.tasks = tasks
	return r.err
}

func counterIDGenerator() (application.IDGenerator, *int) {
	calls := 0
	return func() string {
		calls++
		return "generated-id-" + strconv.Itoa(calls)
	}, &calls
}

func TestCreateTasksUseCaseExecute(t *testing.T) {
	t.Run("success with three tasks assigns IDs in order and preserves content", func(t *testing.T) {
		repo := &fakeTaskRepository{}
		generateID, calls := counterIDGenerator()
		useCase := application.NewCreateTasksUseCase(repo, generateID)
		estimate := 4.5
		command := application.CreateTasksCommand{
			ProjectID: "project-1", SprintID: "sprint-1", StoryID: "story-1",
			Tasks: []application.TaskInput{
				{Title: "Primera", EstimatedHours: &estimate},
				{Title: "Segunda"},
				{Title: "Tercera"},
			},
		}

		result, err := useCase.Execute(context.Background(), command)

		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if repo.calls != 1 {
			t.Fatalf("repository calls = %d, want 1", repo.calls)
		}
		if *calls != 3 {
			t.Fatalf("generator calls = %d, want 3", *calls)
		}
		if repo.projectID != "project-1" || repo.sprintID != "sprint-1" || repo.storyID != "story-1" {
			t.Errorf("repository route = (%s,%s,%s), want (project-1,sprint-1,story-1)", repo.projectID, repo.sprintID, repo.storyID)
		}
		if len(result) != 3 {
			t.Fatalf("result length = %d, want 3", len(result))
		}
		wantIDs := []string{"generated-id-1", "generated-id-2", "generated-id-3"}
		for i, task := range result {
			if task.ID != wantIDs[i] {
				t.Errorf("result[%d].ID = %q, want %q", i, task.ID, wantIDs[i])
			}
		}
		if result[0].Title != "Primera" || result[0].EstimatedHours == nil || *result[0].EstimatedHours != 4.5 {
			t.Errorf("result[0] = %#v, want title Primera with estimate 4.5", result[0])
		}
		if result[1].Title != "Segunda" || result[1].EstimatedHours != nil {
			t.Errorf("result[1] = %#v, want title Segunda with nil estimate", result[1])
		}
		if !reflect.DeepEqual(result, repo.tasks) {
			t.Errorf("returned result = %#v, want identical to what was sent to the repository %#v", result, repo.tasks)
		}
	})

	t.Run("empty batch is rejected without calling the generator or the repository", func(t *testing.T) {
		for _, name := range []string{"nil", "empty"} {
			t.Run(name, func(t *testing.T) {
				repo := &fakeTaskRepository{}
				generateID, calls := counterIDGenerator()
				useCase := application.NewCreateTasksUseCase(repo, generateID)
				var tasks []application.TaskInput
				if name == "empty" {
					tasks = []application.TaskInput{}
				}

				_, err := useCase.Execute(context.Background(), application.CreateTasksCommand{
					ProjectID: "project-1", SprintID: "sprint-1", StoryID: "story-1", Tasks: tasks,
				})

				var validation *domain.ValidationError
				if !errors.As(err, &validation) {
					t.Fatalf("Execute() error = %v, want ValidationError", err)
				}
				if validation.Fields["tasks"] != "must contain at least one task" {
					t.Errorf("Fields[tasks] = %q, want %q", validation.Fields["tasks"], "must contain at least one task")
				}
				if repo.calls != 0 || *calls != 0 {
					t.Errorf("calls = (repo %d, generator %d), want (0, 0)", repo.calls, *calls)
				}
			})
		}
	})

	t.Run("one invalid element among valid ones reports its indexed key without calling the repository", func(t *testing.T) {
		repo := &fakeTaskRepository{}
		generateID, calls := counterIDGenerator()
		useCase := application.NewCreateTasksUseCase(repo, generateID)

		_, err := useCase.Execute(context.Background(), application.CreateTasksCommand{
			ProjectID: "project-1", SprintID: "sprint-1", StoryID: "story-1",
			Tasks: []application.TaskInput{
				{Title: "Primera"},
				{Title: "   "},
				{Title: "Tercera"},
			},
		})

		var validation *domain.ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("Execute() error = %v, want ValidationError", err)
		}
		if len(validation.Fields) != 1 || validation.Fields["tasks[1].title"] != "is required" {
			t.Errorf("Fields = %#v, want only tasks[1].title", validation.Fields)
		}
		if repo.calls != 0 || *calls != 0 {
			t.Errorf("calls = (repo %d, generator %d), want (0, 0)", repo.calls, *calls)
		}
	})

	t.Run("multiple invalid elements accumulate every key without stopping at the first", func(t *testing.T) {
		repo := &fakeTaskRepository{}
		generateID, calls := counterIDGenerator()
		useCase := application.NewCreateTasksUseCase(repo, generateID)
		invalidHours := -1.0

		_, err := useCase.Execute(context.Background(), application.CreateTasksCommand{
			ProjectID: "project-1", SprintID: "sprint-1", StoryID: "story-1",
			Tasks: []application.TaskInput{
				{Title: ""},
				{Title: "Valida"},
				{Title: "Valida", EstimatedHours: &invalidHours},
			},
		})

		var validation *domain.ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("Execute() error = %v, want ValidationError", err)
		}
		want := map[string]string{
			"tasks[0].title":           "is required",
			"tasks[2].estimated_hours": estimatedHoursMessage,
		}
		if !reflect.DeepEqual(validation.Fields, want) {
			t.Errorf("Fields = %#v, want %#v", validation.Fields, want)
		}
		if repo.calls != 0 || *calls != 0 {
			t.Errorf("calls = (repo %d, generator %d), want (0, 0)", repo.calls, *calls)
		}
	})

	t.Run("repository errors are preserved and the result is nil", func(t *testing.T) {
		for _, wantErr := range []error{
			application.ErrProjectNotFound,
			application.ErrSprintNotFound,
			application.ErrStoryNotFound,
			application.ErrStoryNotInSprint,
			errors.New("unexpected storage failure"),
		} {
			t.Run(wantErr.Error(), func(t *testing.T) {
				repo := &fakeTaskRepository{err: wantErr}
				generateID, _ := counterIDGenerator()
				useCase := application.NewCreateTasksUseCase(repo, generateID)

				result, err := useCase.Execute(context.Background(), application.CreateTasksCommand{
					ProjectID: "project-1", SprintID: "sprint-1", StoryID: "story-1",
					Tasks: []application.TaskInput{{Title: "Primera"}},
				})

				if !errors.Is(err, wantErr) {
					t.Errorf("Execute() error = %v, want errors.Is(%v)", err, wantErr)
				}
				if result != nil {
					t.Errorf("Execute() result = %#v, want nil", result)
				}
			})
		}
	})

	t.Run("mutating the command's estimate pointer after Execute does not alter the returned tasks", func(t *testing.T) {
		repo := &fakeTaskRepository{}
		generateID, _ := counterIDGenerator()
		useCase := application.NewCreateTasksUseCase(repo, generateID)
		estimate := 2.5
		command := application.CreateTasksCommand{
			ProjectID: "project-1", SprintID: "sprint-1", StoryID: "story-1",
			Tasks: []application.TaskInput{{Title: "Primera", EstimatedHours: &estimate}},
		}

		result, err := useCase.Execute(context.Background(), command)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		estimate = 999

		if result[0].EstimatedHours == nil || *result[0].EstimatedHours != 2.5 {
			t.Errorf("EstimatedHours = %v, want 2.5 unaffected by later mutation", result[0].EstimatedHours)
		}
	})
}

const estimatedHoursMessage = "must be greater than 0, at most 99999.99 and have at most 2 decimals"
