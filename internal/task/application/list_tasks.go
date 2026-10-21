package application

import (
	"context"

	"github.com/valerubio7/software-metrics-and-estimation/internal/task/domain"
)

// TaskLister reads the tasks of a project-scoped story in creation order.
type TaskLister interface {
	ListByProjectStory(ctx context.Context, projectID, storyID string) ([]domain.Task, error)
}

// ListTasksQuery identifies the project and story whose tasks are requested.
type ListTasksQuery struct {
	ProjectID string
	StoryID   string
}

// ProjectStoryTasks is the route context plus the task collection.
type ProjectStoryTasks struct {
	ProjectID string
	StoryID   string
	Tasks     []domain.Task
}

// ListTasksUseCase reads a story's tasks without modifying anything.
type ListTasksUseCase struct {
	lister TaskLister
}

// NewListTasksUseCase builds a list-tasks use case with its only dependency.
func NewListTasksUseCase(lister TaskLister) *ListTasksUseCase {
	return &ListTasksUseCase{lister: lister}
}

// Execute is implemented in the GREEN step.
func (u *ListTasksUseCase) Execute(ctx context.Context, query ListTasksQuery) (ProjectStoryTasks, error) {
	return ProjectStoryTasks{}, nil
}
