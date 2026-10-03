// Package application contains the use case that decomposes a story into tasks.
package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/valerubio7/software-metrics-and-estimation/internal/task/domain"
)

var (
	ErrProjectNotFound  = errors.New("project not found")
	ErrSprintNotFound   = errors.New("sprint not found")
	ErrStoryNotFound    = errors.New("story not found")
	ErrStoryNotInSprint = errors.New("story is not assigned to sprint")
)

// TaskInput is the client-supplied content of one task.
type TaskInput struct {
	Title          string
	EstimatedHours *float64
}

// CreateTasksCommand identifies the story within the sprint and the complete batch.
type CreateTasksCommand struct {
	ProjectID string
	SprintID  string
	StoryID   string
	Tasks     []TaskInput
}

// TaskRepository verifies, in one transaction, that the project exists, the sprint and the
// story belong to it and the story is assigned to the sprint, then inserts every task or
// none. It returns ErrProjectNotFound, ErrSprintNotFound, ErrStoryNotFound or
// ErrStoryNotInSprint for the corresponding failure; other errors are returned unchanged.
type TaskRepository interface {
	CreateForSprintStory(ctx context.Context, projectID, sprintID, storyID string, tasks []domain.Task) error
}

// IDGenerator creates identifiers for new tasks.
type IDGenerator func() string

// CreateTasksUseCase validates the complete batch before delegating to persistence.
type CreateTasksUseCase struct {
	repository TaskRepository
	generateID IDGenerator
}

// NewCreateTasksUseCase builds a use case backed by the supplied port and ID generator.
func NewCreateTasksUseCase(repository TaskRepository, generateID IDGenerator) *CreateTasksUseCase {
	return &CreateTasksUseCase{repository: repository, generateID: generateID}
}

// Execute validates the whole batch, generates IDs and persists it atomically. It returns
// the created tasks only after the repository succeeds.
func (u *CreateTasksUseCase) Execute(ctx context.Context, command CreateTasksCommand) ([]domain.Task, error) {
	if len(command.Tasks) == 0 {
		return nil, &domain.ValidationError{Fields: map[string]string{"tasks": "must contain at least one task"}}
	}

	fields := make(map[string]string)
	tasks := make([]domain.Task, len(command.Tasks))
	for i, input := range command.Tasks {
		task, err := domain.NewTask("", command.ProjectID, command.SprintID, command.StoryID, input.Title, input.EstimatedHours)
		var validation *domain.ValidationError
		if errors.As(err, &validation) {
			for field, message := range validation.Fields {
				fields[fmt.Sprintf("tasks[%d].%s", i, field)] = message
			}
			continue
		}
		tasks[i] = task
	}
	if len(fields) > 0 {
		return nil, &domain.ValidationError{Fields: fields}
	}

	for i := range tasks {
		tasks[i].ID = u.generateID()
	}

	if err := u.repository.CreateForSprintStory(ctx, command.ProjectID, command.SprintID, command.StoryID, tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}
